package multiverse

import (
	"container/heap"
	"fmt"
	"sync"
	"time"

	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/network"
)

// region Scheduler //////////////////////////////////////////////////////////////////////////////////////////////

// Priority Queue for Message
type MessageHeap []Message

type Scheduler struct {
	tangle        *Tangle
	readyQueue    *MessageHeap
	nonReadyQueue *MessageHeap
	accessMana    map[network.PeerID]float64

	manaMutex sync.RWMutex
	Events    *SchedulerEvents
}

func NewScheduler(tangle *Tangle) (mq *Scheduler) {
	readyHeap := &MessageHeap{}
	heap.Init(readyHeap)
	nonReadyHeap := &MessageHeap{}
	heap.Init(nonReadyHeap)
	return &Scheduler{
		tangle:        tangle,
		readyQueue:    readyHeap,
		nonReadyQueue: nonReadyHeap,
		accessMana:    make(map[network.PeerID]float64, config.NodesCount),
		Events: &SchedulerEvents{
			MessageScheduled: events.NewEvent(messageIDEventCaller),
			MessageDropped:   events.NewEvent(messageIDEventCaller),
		},
	}
}

func (s *Scheduler) Setup(tangle *Tangle) {
	// Setup the initial AccessMana when the peer ID is created
	s.Events.MessageScheduled.Attach(events.NewClosure(func(messageID MessageID) {
		s.tangle.Storage.MessageMetadata(messageID).SetScheduleTime(time.Now())
		s.updateChildrenReady(messageID)
	}))
	s.Events.MessageDropped.Attach(events.NewClosure(func(messageID MessageID) {
		s.tangle.Storage.MessageMetadata(messageID).SetDropTime(time.Now())
	}))
}

func (s *Scheduler) updateChildrenReady(messageID MessageID) {
	for strongChildID := range s.tangle.Storage.strongChildrenDB[messageID] {
		if s.messageReady(strongChildID) {
			s.tangle.Storage.MessageMetadata(strongChildID).SetReady()
		}
	}
	for weakChildID := range s.tangle.Storage.weakChildrenDB[messageID] {
		if s.messageReady(weakChildID) {
			s.tangle.Storage.MessageMetadata(weakChildID).SetReady()
		}
	}
}

func (s *Scheduler) messageReady(messageID MessageID) bool {
	if !s.tangle.Storage.MessageMetadata(messageID).Solid() {
		return false
	}
	message := s.tangle.Storage.Message(messageID)
	for strongParentID := range message.StrongParents {
		if strongParentID > 0 {
			strongParentMetadata := s.tangle.Storage.MessageMetadata(strongParentID)
			if strongParentMetadata == nil {
				fmt.Println("Strong Parent Metadata is empty")
			}
			if !strongParentMetadata.Scheduled() && !strongParentMetadata.Confirmed() {
				return false
			}
		}
	}
	for weakParentID := range message.WeakParents {
		weakParentMetadata := s.tangle.Storage.MessageMetadata(weakParentID)
		if weakParentID > 0 {
			if !weakParentMetadata.Scheduled() && !weakParentMetadata.Confirmed() {
				return false
			}
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
	s.manaMutex.Lock()
	defer s.manaMutex.Unlock()
	s.accessMana[nodeID] += manaIncrement
}

func (s *Scheduler) IncrementAccessMana() {
	s.manaMutex.Lock()
	defer s.manaMutex.Unlock()
	weights := s.tangle.WeightDistribution.Weights()
	totalWeight := config.NodesTotalWeight
	for id := range s.accessMana {
		s.accessMana[id] += float64(weights[id]) / float64(totalWeight)
	}
}

func (s *Scheduler) DecreaseNodeAccessMana(nodeID network.PeerID, manaIncrement float64) (newAccessMana float64) {
	s.manaMutex.Lock()
	defer s.manaMutex.Unlock()
	s.accessMana[nodeID] -= manaIncrement
	return s.accessMana[nodeID]
}

func (s *Scheduler) SetNodeAccessMana(nodeID network.PeerID, newMana float64) {
	s.manaMutex.Lock()
	defer s.manaMutex.Unlock()
	s.accessMana[nodeID] = newMana
}

func (s *Scheduler) GetNodeAccessMana(nodeID network.PeerID) (mana float64) {
	s.manaMutex.RLock()
	defer s.manaMutex.RUnlock()
	return s.accessMana[nodeID]
}

func (s *Scheduler) GetMaxManaBurn() float64 {
	if s.nonReadyQueue.Len() > 0 {
		return (*s.readyQueue)[0].ManaBurnValue
	} else {
		return 0.0
	}
}

func (s *Scheduler) ScheduleMessage() (Message, float64, bool) {
	// Consume the accessMana and pop the Message
	if s.IsEmpty() {
		return Message{}, 0.0, false
	} else {
		m := heap.Pop(s.readyQueue).(Message)
		newAccessMana := s.DecreaseNodeAccessMana(m.Issuer, m.ManaBurnValue)
		s.Events.MessageScheduled.Trigger(m.ID)
		return m, newAccessMana, true
	}
}

func (s *Scheduler) EnqueueMessage(messageID MessageID) {
	s.tangle.Storage.MessageMetadata(messageID).SetEnqueueTime(time.Now())
	// Check if the message is ready to decide which queue to append to
	if s.messageReady(messageID) {
		s.tangle.Storage.MessageMetadata(messageID).SetReady()
	}
	m := *s.tangle.Storage.Message(messageID)
	heap.Push(s.readyQueue, m)
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
