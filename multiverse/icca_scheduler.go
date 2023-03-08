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
	readyLen     int

	events *SchedulerEvents
}

func (s *ICCAScheduler) Setup() {
	// Setup the initial AccessMana, deficits and quanta when the peer ID is created
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
	// move from non ready queue to ready queue if this child is already enqueued
	if m, exists := s.nonReadyMap[messageID]; exists {
		delete(s.nonReadyMap, messageID)
		heap.Push(s.issuerQueues[m.Issuer], *m)
		s.readyLen += 1
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

func (s *ICCAScheduler) BurnValue(issuanceTime time.Time) (float64, bool) {
	slotIndex := SlotIndex(float64(issuanceTime.Sub(s.tangle.Storage.genesisTime)) / float64(config.SlotTime))
	RMC := s.tangle.Storage.RMC(slotIndex)
	return RMC, s.GetNodeAccessMana(s.tangle.Peer.ID) >= RMC
}

func (s *ICCAScheduler) EnqueueMessage(messageID MessageID) {
	s.tangle.Storage.MessageMetadata(messageID).SetEnqueueTime(time.Now())
	m := s.tangle.Storage.Message(messageID)
	// Check if the message is ready to decide which queue to append to
	if s.tangle.Storage.isReady(messageID) {
		//log.Debugf("Ready Message Enqueued")
		s.tangle.Storage.MessageMetadata(messageID).SetReady()
		heap.Push(s.issuerQueues[m.Issuer], *m)
		s.readyLen += 1
	} else {
		//log.Debug("Not Ready Message Enqueued")
		s.tangle.Storage.MessageMetadata(messageID).SetReady()
		s.nonReadyMap[messageID] = s.tangle.Storage.Message(messageID)
	}
	s.events.MessageEnqueued.Trigger(s.issuerQueues[m.Issuer].Len(), len(s.nonReadyMap))
	s.BufferManagement()
}

func (s *ICCAScheduler) BufferManagement() {
	for s.readyLen > config.MaxBuffer {
		issuerID := 0
		maxScaledLen := 0.0
		for id := 0; id < config.NodesCount; id++ {
			scaledLen := float64(s.issuerQueues[network.PeerID(id)].Len()) / s.quanta[network.PeerID(id)]
			if scaledLen >= maxScaledLen {
				maxScaledLen = scaledLen
				issuerID = id
			}
		}
		heap.Pop(s.issuerQueues[network.PeerID(issuerID)]) // drop head
		s.readyLen -= 1
	}
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
	s.readyLen -= 1
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
	return s.readyLen
}

func (s *ICCAScheduler) NonReadyLen() int {
	return len(s.nonReadyMap)
}

func (s *ICCAScheduler) GetNodeAccessMana(nodeID network.PeerID) (mana float64) {
	mana = s.accessMana[nodeID]
	return mana
}

func (s *ICCAScheduler) GetMaxManaBurn() (maxManaBurn float64) {
	for id := 0; id < config.NodesCount; id++ {
		q := s.issuerQueues[network.PeerID(id)]
		if q.Len() > 0 {
			maxManaBurn = math.Max(maxManaBurn, (*q)[0].ManaBurnValue)
		}
	}
	return
}

func (s *ICCAScheduler) IssuerQueueLen(issuer network.PeerID) int {
	return s.issuerQueues[issuer].Len()
}

func (s *ICCAScheduler) Deficit(issuer network.PeerID) float64 {
	return s.deficits[issuer]
}
