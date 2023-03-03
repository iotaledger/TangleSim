package multiverse

import (
	"container/heap"
	"container/ring"
	"math"
	"time"

	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/network"
)

// region ICCA Scheduler ////////////////////////////////////////////////////////////////////////////////////////////////////

func (s *ICCAScheduler) initQueues() {
	for i := 0; i < config.NodesCount; i++ {
		issuerQueue := &IssuerQueue{}
		heap.Init(issuerQueue)
		s.issuerQueues[network.PeerID(i)] = issuerQueue
		s.roundRobin.Value = &DRRQueue{
			issuerID: network.PeerID(i),
			q:        issuerQueue,
		}
		s.roundRobin = s.roundRobin.Next()
	}
	if s.roundRobin.Value.(*DRRQueue).issuerID != 0 {
		panic("Incomplete ring")
	}
}

// region ICCA Scheduler ////////////////////////////////////////////////////////////////////////////////////////////////////

type ICCAScheduler struct {
	tangle       *Tangle
	nonReadyMap  map[MessageID]*Message
	accessMana   map[network.PeerID]float64
	deficits     map[network.PeerID]float64
	quanta       map[network.PeerID]float64
	issuerQueues map[network.PeerID]*IssuerQueue
	roundRobin   *ring.Ring

	events *SchedulerEvents
}

func (s *ICCAScheduler) Setup() {
	// setup the initial AccessMana, deficits and quanta when the peer ID is created
	for id := 0; id < config.NodesCount; id++ {
		s.accessMana[network.PeerID(id)] = 0.0
		s.deficits[network.PeerID(id)] = 0.0
		idWeight := s.tangle.WeightDistribution.Weight(network.PeerID(id))
		s.quanta[network.PeerID(id)] = float64(idWeight) / float64(config.NodesTotalWeight)
	}
	// initialise the issuer queues
	s.initQueues()
	s.events.MessageScheduled.Attach(events.NewClosure(func(messageID MessageID) {
		s.tangle.Peer.GossipNetworkMessage(s.tangle.Storage.Message(messageID))
		s.updateChildrenReady(messageID)
		//		log.Debugf("Peer %d Gossiped message %d",
		//	s.tangle.Peer.ID, messageID)
	}))
	s.events.MessageDropped.Attach(events.NewClosure(func(messageID MessageID) {
		s.tangle.Storage.MessageMetadata(messageID).SetDropTime(time.Now())
	}))
	s.tangle.ApprovalManager.Events.MessageConfirmed.Attach(events.NewClosure(func(message *Message, messageMetadata *MessageMetadata, weight uint64, messageIDCounter int64) {
		if config.ConfEligible {
			s.updateChildrenReady(message.ID)
		}
	}))
}

func (s *ICCAScheduler) updateChildrenReady(messageID MessageID) {
	for strongChildID := range s.tangle.Storage.StrongChildren(messageID) {
		if s.tangle.Storage.isReady(strongChildID) {
			s.setReady(strongChildID)
		}
	}
	for weakChildID := range s.tangle.Storage.WeakChildren(messageID) {
		if s.tangle.Storage.isReady(weakChildID) {
			s.setReady(weakChildID)
		}
	}
}

func (s *ICCAScheduler) setReady(messageID MessageID) {
	s.tangle.Storage.MessageMetadata(messageID).SetReady()
	// move from non-ready queue to ready queue if this child is already enqueued
	if m, exists := s.nonReadyMap[messageID]; exists {
		delete(s.nonReadyMap, messageID)
		heap.Push(s.issuerQueues[m.Issuer], *m)
	}
}

func (s *ICCAScheduler) IncrementAccessMana(schedulingRate float64) {
	weights := s.tangle.WeightDistribution.Weights()
	totalWeight := config.NodesTotalWeight
	// every time something is scheduled, we add this much mana in total\
	mana := float64(10)
	for id := range s.accessMana {
		s.accessMana[id] += mana * float64(weights[id]) / float64(totalWeight)
	}
}

func (s *ICCAScheduler) DecreaseNodeAccessMana(nodeID network.PeerID, manaIncrement float64) (newAccessMana float64) {
	s.accessMana[nodeID] -= manaIncrement
	newAccessMana = s.accessMana[nodeID]
	return newAccessMana
}

func (s *ICCAScheduler) BurnValue() (float64, bool) {
	return 0.0, true // always just burn 0 mana for ICCA for now.
}

func (s *ICCAScheduler) EnqueueMessage(messageID MessageID) {
	s.tangle.Storage.MessageMetadata(messageID).SetEnqueueTime(time.Now())
	m := s.tangle.Storage.Message(messageID)
	// Check if the message is ready to decide which queue to append to
	if s.tangle.Storage.isReady(messageID) {
		//log.Debugf("Ready Message Enqueued")
		s.tangle.Storage.MessageMetadata(messageID).SetReady()
		heap.Push(s.issuerQueues[m.Issuer], *m)
	} else {
		//log.Debug("Not Ready Message Enqueued")
		s.tangle.Storage.MessageMetadata(messageID).SetReady()
		s.nonReadyMap[messageID] = s.tangle.Storage.Message(messageID)
	}
	s.events.MessageEnqueued.Trigger(s.issuerQueues[m.Issuer].Len(), len(s.nonReadyMap))
}

func (s *ICCAScheduler) ScheduleMessage() {
	rounds, selectedIssuerID := s.selectIssuer()
	if selectedIssuerID == network.PeerID(-1) {
		return
	}
	for id := 0; id < config.NodesCount; id++ {
		// increment all deficits by the number of rounds needed.
		s.deficits[network.PeerID(id)] = math.Min(
			s.deficits[network.PeerID(id)]+rounds*s.quanta[network.PeerID(id)],
			config.MaxDeficit,
		)
	}
	for id := s.roundRobin.Value.(*DRRQueue).issuerID; id != selectedIssuerID; id = s.roundRobin.Value.(*DRRQueue).issuerID {
		// increment all the issuers before the selected issuer by one more round.
		s.deficits[id] = math.Min(
			s.deficits[id]+s.quanta[id],
			config.MaxDeficit,
		)
		s.roundRobin = s.roundRobin.Next()
	}
	// now the ring is pointing to the selected issuer and deficits are updated.
	// pop the message from the chosen issuer's queue
	m := s.roundRobin.Value.(*DRRQueue).q.Pop().(Message)
	// decrement its deficit
	s.deficits[s.roundRobin.Value.(*DRRQueue).issuerID]-- // assumes work==1
	// schedule the message
	s.tangle.Storage.MessageMetadata(m.ID).SetScheduleTime(time.Now())
	s.updateChildrenReady(m.ID)
	s.events.MessageScheduled.Trigger(m.ID)
}

func (s *ICCAScheduler) selectIssuer() (rounds float64, issuerID network.PeerID) {
	rounds = math.MaxFloat64
	issuerID = network.PeerID(-1)
	for i := 0; i < config.NodesCount; i++ {
		if s.roundRobin.Value.(*DRRQueue).q.Len() == 0 {
			s.roundRobin = s.roundRobin.Next()
			continue
		}
		id := s.roundRobin.Value.(*DRRQueue).issuerID
		r := (math.Max(1-s.deficits[network.PeerID(id)], 0) / s.quanta[network.PeerID(id)])
		if r < rounds {
			rounds = r
			issuerID = id
		}
		s.roundRobin = s.roundRobin.Next()
	}
	return
}

func (s *ICCAScheduler) Events() *SchedulerEvents {
	return s.events
}

func (s *ICCAScheduler) ReadyLen() int {
	return s.issuerQueues[s.tangle.Peer.ID].Len() // return length of own ready queue only
}

func (s *ICCAScheduler) NonReadyLen() int {
	return len(s.nonReadyMap)
}

func (s *ICCAScheduler) GetNodeAccessMana(nodeID network.PeerID) (mana float64) {
	mana = s.accessMana[nodeID]
	return mana
}

func (s *ICCAScheduler) GetMaxManaBurn() (mana float64) {
	return 0.0
}

func (s *ICCAScheduler) IssuerQueueLen(issuer network.PeerID) int {
	return s.issuerQueues[issuer].Len()
}

func (s *ICCAScheduler) Deficit(issuer network.PeerID) float64 {
	return s.deficits[issuer]
}
