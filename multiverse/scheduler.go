package multiverse

import (
	"container/heap"
	"fmt"
	"time"

	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/config"
)

// region Scheduler //////////////////////////////////////////////////////////////////////////////////////////////

// Priority Queue for Message
type MessageHeap []Message

type Scheduler struct {
	tangle        *Tangle
	readyQueue    *MessageHeap
	nonReadyQueue *MessageHeap
	accessMana    float64

	Events *SchedulerEvents
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
		accessMana:    0.0,
		Events: &SchedulerEvents{
			MessageScheduled: events.NewEvent(messageIDEventCaller),
			MessageDropped:   events.NewEvent(messageIDEventCaller),
		},
	}
}

func (s *Scheduler) Setup(tangle *Tangle) {
	// Setup the initial AccessMana when the peer ID is created
	s.accessMana = config.NodeInitAccessMana[tangle.Peer.ID]
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
func (s *Scheduler) IncreaseAccessMana(manaIncrement float64) {
	s.accessMana += manaIncrement
}

func (s *Scheduler) DecreaseAccessMana(manaIncrement float64) (newAccessMana float64) {
	s.accessMana -= manaIncrement
	return s.accessMana
}

func (s *Scheduler) SetAccessMana(mana float64) {
	s.accessMana = mana
}

func (s *Scheduler) GetAccessMana() (mana float64) {
	return s.accessMana
}

func (s *Scheduler) GetMaxManaBurn() float64 {
	return (*s.readyQueue)[0].ManaBurnValue
}

func (s *Scheduler) ScheduleMessage() (Message, float64) {
	// Consume the accessMana and pop the Message
	m := heap.Pop(s.readyQueue).(Message)
	s.accessMana -= m.ManaBurnValue
	s.Events.MessageScheduled.Trigger(m.ID)
	return m, s.accessMana
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
