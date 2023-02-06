package multiverse

import (
	"container/heap"
	"time"

	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/config"
)

// region Scheduler //////////////////////////////////////////////////////////////////////////////////////////////

// Priority Queue for Message
type MessageHeap []Message

type Scheduler struct {
	tangle        *Tangle
	priorityQueue *MessageHeap
	accessMana    float64

	Events *SchedulerEvents
}

func NewScheduler(tangle *Tangle) (mq *Scheduler) {
	h := &MessageHeap{}
	heap.Init(h)
	return &Scheduler{
		tangle:        tangle,
		priorityQueue: h,
		accessMana:    0.0,
		Events: &SchedulerEvents{
			MessageEnqueued:  events.NewEvent(messageIDEventCaller),
			MessageScheduled: events.NewEvent(messageIDEventCaller),
			MessageDropped:   events.NewEvent(messageIDEventCaller),
		},
	}
}

func (s *Scheduler) Setup(tangle *Tangle) {
	// Setup the initial AccessMana when the peer ID is created
	s.accessMana = config.NodeInitAccessMana[tangle.Peer.ID]
	s.Events.MessageEnqueued.Attach(events.NewClosure(func(messageID MessageID) {
		messageMetaData := *s.tangle.Storage.MessageMetadata(messageID)
		messageMetaData.SetEnqueueTime(time.Now())
	}))
	s.Events.MessageScheduled.Attach(events.NewClosure(func(messageID MessageID) {
		messageMetaData := *s.tangle.Storage.MessageMetadata(messageID)
		messageMetaData.SetScheduleTime(time.Now())
	}))
	s.Events.MessageDropped.Attach(events.NewClosure(func(messageID MessageID) {
		messageMetaData := *s.tangle.Storage.MessageMetadata(messageID)
		messageMetaData.SetDropTime(time.Now())
	}))
}

func (s *Scheduler) IsEmpty() bool {
	return s.priorityQueue.Len() == 0
}

func (s *Scheduler) PriorityQueueLen() int {
	return s.priorityQueue.Len()
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
	return (*s.priorityQueue)[0].ManaBurnValue
}

func (s *Scheduler) ScheduleMessage() (Message, float64) {
	// Consume the accessMana and pop the Message
	m := heap.Pop(s.priorityQueue).(Message)
	s.accessMana -= m.ManaBurnValue
	s.Events.MessageScheduled.Trigger(m.ID)
	return m, s.accessMana
}

func (s *Scheduler) EnqueueMessage(m Message) {
	heap.Push(s.priorityQueue, m)
	s.Events.MessageEnqueued.Trigger(m.ID)
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
	MessageEnqueued  *events.Event
	MessageScheduled *events.Event
	MessageDropped   *events.Event
}
