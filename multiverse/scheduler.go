package multiverse

import (
	"container/heap"
	"fmt"
	"time"

	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/network"
)

// region Scheduler //////////////////////////////////////////////////////////////////////////////////////////////

// Priority Queue for Message
type MessageHeap []Message

type Scheduler struct {
	tangle      *Tangle
	readyQueue  *MessageHeap
	nonReadyMap map[MessageID]*Message
	accessMana  map[network.PeerID]float64

	Events *SchedulerEvents
}

func NewScheduler(tangle *Tangle) (mq *Scheduler) {
	readyHeap := &MessageHeap{}
	heap.Init(readyHeap)
	return &Scheduler{
		tangle:      tangle,
		readyQueue:  readyHeap,
		nonReadyMap: make(map[MessageID]*Message),
		accessMana:  make(map[network.PeerID]float64, config.NodesCount),
		Events: &SchedulerEvents{
			MessageScheduled: events.NewEvent(messageIDEventCaller),
			MessageDropped:   events.NewEvent(messageIDEventCaller),
		},
	}
}

func (s *Scheduler) Setup(tangle *Tangle) {
	// Setup the initial AccessMana when the peer ID is created
	for id := 0; id < config.NodesCount; id++ {
		s.accessMana[network.PeerID(id)] = 0.0
	}
	s.Events.MessageScheduled.Attach(events.NewClosure(func(messageID MessageID) {
		s.tangle.Peer.GossipNetworkMessage(s.tangle.Storage.Message(messageID))
		//		log.Debugf("Peer %d Gossiped message %d",
		//	s.tangle.Peer.ID, messageID)
	}))
	s.Events.MessageDropped.Attach(events.NewClosure(func(messageID MessageID) {
		s.tangle.Storage.MessageMetadata(messageID).SetDropTime(time.Now())
	}))
}

func (s *Scheduler) updateChildrenReady(messageID MessageID) {
	for strongChildID := range s.tangle.Storage.StrongChildren(messageID) {
		if s.isReady(strongChildID) {
			s.setReady(strongChildID)
		}
	}
	for weakChildID := range s.tangle.Storage.WeakChildren(messageID) {
		if s.isReady(weakChildID) {
			s.tangle.Storage.MessageMetadata(weakChildID).SetReady()
		}
	}
}

func (s *Scheduler) setReady(messageID MessageID) {
	s.tangle.Storage.MessageMetadata(messageID).SetReady()
	// move from non ready queue to ready queue if this child is already enqueued
	if m, exists := s.nonReadyMap[messageID]; exists {
		delete(s.nonReadyMap, messageID)
		heap.Push(s.readyQueue, *m)
	}
}

func (s *Scheduler) isReady(messageID MessageID) bool {
	if !s.tangle.Storage.MessageMetadata(messageID).Solid() {
		return false
	}
	message := s.tangle.Storage.Message(messageID)
	for strongParentID := range message.StrongParents {
		if strongParentID == Genesis {
			continue
		}
		strongParentMetadata := s.tangle.Storage.MessageMetadata(strongParentID)
		if strongParentMetadata == nil {
			fmt.Println("Strong Parent Metadata is empty")
		}
		if !strongParentMetadata.Scheduled() && !strongParentMetadata.Confirmed() {
			return false
		}

	}
	for weakParentID := range message.WeakParents {
		weakParentMetadata := s.tangle.Storage.MessageMetadata(weakParentID)
		if weakParentID == Genesis {
			continue
		}
		if !weakParentMetadata.Scheduled() && !weakParentMetadata.Confirmed() {
			return false
		}
	}
	return true
}

func (s *Scheduler) IsEmpty() bool {
	return s.readyQueue.Len() == 0
}

func (s *Scheduler) ReadyLen() int {
	return s.readyQueue.Len()
}
func (s *Scheduler) IncreaseNodeAccessMana(nodeID network.PeerID, manaIncrement float64) {
	s.accessMana[nodeID] += manaIncrement
}

func (s *Scheduler) IncrementAccessMana(schedulingRate float64) {
	weights := s.tangle.WeightDistribution.Weights()
	totalWeight := config.NodesTotalWeight
	for id := range s.accessMana {
		s.accessMana[id] += float64(weights[id]) / float64(totalWeight) / schedulingRate
	}
}

func (s *Scheduler) DecreaseNodeAccessMana(nodeID network.PeerID, manaIncrement float64) (newAccessMana float64) {
	s.accessMana[nodeID] -= manaIncrement
	newAccessMana = s.accessMana[nodeID]
	return newAccessMana
}

func (s *Scheduler) GetNodeAccessMana(nodeID network.PeerID) (mana float64) {
	mana = s.accessMana[nodeID]
	return mana
}

func (s *Scheduler) GetMaxManaBurn() float64 {
	if s.readyQueue.Len() > 0 {
		return (*s.readyQueue)[0].ManaBurnValue
	} else {
		return 0.0
	}
}

func (s *Scheduler) ScheduleMessage() (Message, float64, bool) {
	// Consume the accessMana and pop the Message
	if s.IsEmpty() {
		//log.Debugf("Scheduler is empty: Peer %d", s.tangle.Peer.ID)
		return Message{}, 0.0, false
	} else {
		m := heap.Pop(s.readyQueue).(Message)
		newAccessMana := s.DecreaseNodeAccessMana(m.Issuer, m.ManaBurnValue)
		s.tangle.Storage.MessageMetadata(m.ID).SetScheduleTime(time.Now())
		s.updateChildrenReady(m.ID)
		log.Debugf("Peer %d Scheduled message %d: from issuer %d with access mana %f and message ManaBurnValue %f",
			s.tangle.Peer.ID, m.ID, m.Issuer, newAccessMana, m.ManaBurnValue)
		s.Events.MessageScheduled.Trigger(m.ID)
		return m, newAccessMana, true
	}
}

func (s *Scheduler) EnqueueMessage(messageID MessageID) {
	s.tangle.Storage.MessageMetadata(messageID).SetEnqueueTime(time.Now())
	// Check if the message is ready to decide which queue to append to
	if s.isReady(messageID) {
		//log.Debugf("Ready Message Enqueued")
		s.tangle.Storage.MessageMetadata(messageID).SetReady()
		m := *s.tangle.Storage.Message(messageID)
		heap.Push(s.readyQueue, m)
	} else {
		//log.Debug("Not Ready Message Enqueued")
		s.tangle.Storage.MessageMetadata(messageID).SetReady()
		s.nonReadyMap[messageID] = s.tangle.Storage.Message(messageID)
	}
}

func (h MessageHeap) Len() int { return len(h) }
func (h MessageHeap) Less(i, j int) bool {
	return h[i].ManaBurnValue > h[j].ManaBurnValue
}
func (h MessageHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *MessageHeap) Push(m any) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, m.(Message))
}

func (h *MessageHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// Scheduler ///////////////////////////////////////////////////////////////////////////////////////////////////////////

type SchedulerEvents struct {
	MessageScheduled *events.Event
	MessageDropped   *events.Event
}
