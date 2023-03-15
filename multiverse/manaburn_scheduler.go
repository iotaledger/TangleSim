package multiverse

import (
	"container/heap"
	"math/rand"
	"time"

	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/network"
)

// region ManaBurn Scheduler ////////////////////////////////////////////////////////////////////////////////////////////////////

type MBScheduler struct {
	tangle      *Tangle
	readyQueue  *PriorityQueue
	nonReadyMap map[MessageID]*Message
	accessMana  map[network.PeerID]float64

	events *SchedulerEvents
}

func (s *MBScheduler) Setup() {
	// Setup the initial AccessMana when the peer ID is created
	for id := 0; id < config.NodesCount; id++ {
		s.accessMana[network.PeerID(id)] = 100000
	}
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

func (s *MBScheduler) BurnValue(issuanceTime time.Time) (burn float64, ok bool) {
	peerID := s.tangle.Peer.ID
	switch policy := config.BurnPolicies[peerID]; BurnPolicyType(policy) {
	case NoBurn:
		return 0.0, true
	case Anxious:
		burn = s.GetNodeAccessMana(peerID)
		ok = true
		return
	case Greedy:
		burn = s.GetMaxManaBurn() + config.ExtraBurn
		ok = burn <= s.GetNodeAccessMana(peerID)
		return
	case RandomGreedy:
		burn = s.GetMaxManaBurn() + config.ExtraBurn*rand.Float64()
		ok = burn <= s.GetNodeAccessMana(peerID)
		return
	default:
		panic("invalid burn policy")
	}
}

func (s *MBScheduler) IncrementAccessMana(schedulingRate float64) {
	weights := s.tangle.WeightDistribution.Weights()
	totalWeight := config.NodesTotalWeight
	// every time something is scheduled, we add this much mana in total\
	mana := float64(10)
	for id := range s.accessMana {
		s.accessMana[id] += mana * float64(weights[id]) / float64(totalWeight)
	}
}

func (s *MBScheduler) DecreaseNodeAccessMana(nodeID network.PeerID, manaIncrement float64) (newAccessMana float64) {
	s.accessMana[nodeID] -= manaIncrement
	newAccessMana = s.accessMana[nodeID]
	return newAccessMana
}

func (s *MBScheduler) ReadyLen() int {
	return s.readyQueue.Len()
}

func (s *MBScheduler) NonReadyLen() int {
	return len(s.nonReadyMap)
}

func (s *MBScheduler) GetNodeAccessMana(nodeID network.PeerID) (mana float64) {
	mana = s.accessMana[nodeID]
	return mana
}

func (s *MBScheduler) Events() *SchedulerEvents {
	return s.events
}

func (s *MBScheduler) updateChildrenReady(messageID MessageID) {
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

func (s *MBScheduler) setReady(messageID MessageID) {
	s.tangle.Storage.MessageMetadata(messageID).SetReady()
	// move from non ready queue to ready queue if this child is already enqueued
	if m, exists := s.nonReadyMap[messageID]; exists {
		delete(s.nonReadyMap, messageID)
		heap.Push(s.readyQueue, *m)
		s.BufferManagement()
	}
}

func (s *MBScheduler) IsEmpty() bool {
	return s.readyQueue.Len() == 0
}

func (s *MBScheduler) GetMaxManaBurn() float64 {
	if s.readyQueue.Len() > 0 {
		return (*s.readyQueue)[0].ManaBurnValue
	} else {
		return 0.0
	}
}

func (s *MBScheduler) ScheduleMessage() {
	// pop the Message from top of the priority queue and consume the accessMana
	if !s.IsEmpty() {
		m := heap.Pop(s.readyQueue).(Message)
		if m.Issuer != s.tangle.Peer.ID { // already deducted Mana for own blocks
			s.DecreaseNodeAccessMana(m.Issuer, m.ManaBurnValue)
		}
		s.tangle.Storage.MessageMetadata(m.ID).SetScheduleTime(time.Now())
		s.updateChildrenReady(m.ID)
		s.events.MessageScheduled.Trigger(m.ID)
	}
}

func (s *MBScheduler) EnqueueMessage(messageID MessageID) {
	s.tangle.Storage.MessageMetadata(messageID).SetEnqueueTime(time.Now())
	// Check if the message is ready to decide which queue to append to
	if s.tangle.Storage.isReady(messageID) {
		//log.Debugf("Ready Message Enqueued")
		s.tangle.Storage.MessageMetadata(messageID).SetReady()
		m := *s.tangle.Storage.Message(messageID)
		heap.Push(s.readyQueue, m)
	} else {
		//log.Debug("Not Ready Message Enqueued")
		s.tangle.Storage.MessageMetadata(messageID).SetReady()
		s.nonReadyMap[messageID] = s.tangle.Storage.Message(messageID)
	}
	s.events.MessageEnqueued.Trigger(s.readyQueue.Len(), len(s.nonReadyMap))
	s.BufferManagement()
}

func (s *MBScheduler) BufferManagement() {
	for s.readyQueue.Len() > config.MaxBuffer {
		tail := s.readyQueue.tail()
		heap.Remove(s.readyQueue, tail) // remove the lowest burn value/ issuance time
	}
}

func (s *MBScheduler) IssuerQueueLen(issuer network.PeerID) int {
	return 0
}

func (s *MBScheduler) Deficit(issuer network.PeerID) float64 {
	return 0.0
}

func (s *MBScheduler) RateSetter() bool {
	return true
}
