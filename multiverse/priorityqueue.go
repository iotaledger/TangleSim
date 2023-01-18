package multiverse

import (
	"container/heap"
)

// region MessagePriorityQueue //////////////////////////////////////////////////////////////////////////////////////////////

// Priority Queue for MessageMetaData
type MessageHeap []MessageMetadata

// Priority Queue for MessageMetadata
type MessagePriorityQueue struct {
	tangle        *Tangle
	priorityQueue *MessageHeap
}

func NewMessagePriorityQueue(tangle *Tangle) (mq *MessagePriorityQueue) {
	h := &MessageHeap{}
	heap.Init(h)
	return &MessagePriorityQueue{
		tangle:        tangle,
		priorityQueue: h,
	}
}

func (mq *MessagePriorityQueue) PopMessageMetadata() MessageMetadata {
	return heap.Pop(mq.priorityQueue).(MessageMetadata)
}

func (mq *MessagePriorityQueue) PushMessageMetadata(m MessageMetadata) {
	heap.Push(mq.priorityQueue, m)
}

func (h MessageHeap) Len() int { return len(h) }
func (h MessageHeap) Less(i, j int) bool {
	return h[i].manaBurnValue > h[j].manaBurnValue
}
func (h MessageHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *MessageHeap) Push(m any) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, m.(MessageMetadata))
}

func (h *MessageHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// MessagePriorityQueue ///////////////////////////////////////////////////////////////////////////////////////////////////////////
