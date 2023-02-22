package multiverse

import (
	"container/heap"
	"container/ring"
	"math/rand"
	"time"

	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/network"
)

// region Scheduler Interface //////////////////////////////////////////////////////////////////////////////////////////////

// Priority Queue for Message
type PriorityQueue []Message

type IssuerQueue []Message

type DRRQueue struct {
	issuerID network.PeerID
	q        *IssuerQueue
}

type BurnPolicyType int

const (
	NoBurn       BurnPolicyType = 0
	Anxious      BurnPolicyType = 1
	Greedy       BurnPolicyType = 2
	RandomGreedy BurnPolicyType = 3
)

type Scheduler interface {
	Setup()
	IncrementAccessMana(float64)
	DecreaseNodeAccessMana(network.PeerID, float64) float64
	BurnValue() (float64, bool)
	EnqueueMessage(MessageID)
	ScheduleMessage()
	Events() *SchedulerEvents
}

func NewScheduler(tangle *Tangle) (s Scheduler) {
	if config.SchedulerType == "ManaBurn" {
		readyHeap := &PriorityQueue{}
		heap.Init(readyHeap)
		s = &MBScheduler{
			tangle:      tangle,
			readyQueue:  readyHeap,
			nonReadyMap: make(map[MessageID]*Message),
			accessMana:  make(map[network.PeerID]float64, config.NodesCount),
			events: &SchedulerEvents{
				MessageScheduled: events.NewEvent(messageIDEventCaller),
				MessageDropped:   events.NewEvent(messageIDEventCaller),
				MessageEnqueued:  events.NewEvent(schedulerEventCaller),
			},
		}
	} else if config.SchedulerType == "ICCA" {
		s = &ICCAScheduler{
			tangle:       tangle,
			nonReadyMap:  make(map[MessageID]*Message),
			accessMana:   make(map[network.PeerID]float64, config.NodesCount),
			deficits:     make(map[network.PeerID]float64, config.NodesCount),
			issuerQueues: make(map[network.PeerID]*IssuerQueue, config.NodesCount),
			issuerRing:   ring.New(config.NodesCount),
			events: &SchedulerEvents{
				MessageScheduled: events.NewEvent(messageIDEventCaller),
				MessageDropped:   events.NewEvent(messageIDEventCaller),
				MessageEnqueued:  events.NewEvent(schedulerEventCaller),
			},
		}
	} else {
		panic("Invalid Scheduler type")
	}
	return
}

// region ICCA Scheduler ////////////////////////////////////////////////////////////////////////////////////////////////////

func (s *ICCAScheduler) initQueues() {
	for i := 0; i < config.NodesCount; i++ {
		issuerQueue := &IssuerQueue{}
		s.issuerQueues[network.PeerID(i)] = issuerQueue
		s.issuerRing.Value = &DRRQueue{
			issuerID: network.PeerID(i),
			q:        issuerQueue,
		}
		s.issuerRing.Next()
	}
}

// region ICCA Scheduler ////////////////////////////////////////////////////////////////////////////////////////////////////
type ICCAScheduler struct {
	tangle       *Tangle
	nonReadyMap  map[MessageID]*Message
	accessMana   map[network.PeerID]float64
	deficits     map[network.PeerID]float64
	issuerQueues map[network.PeerID]*IssuerQueue
	issuerRing   *ring.Ring

	events *SchedulerEvents
}

func (s *ICCAScheduler) Setup() {
	// initialise the issuer queues
	s.initQueues()
	s.events.MessageScheduled.Attach(events.NewClosure(func(messageID MessageID) {
		s.tangle.Peer.GossipNetworkMessage(s.tangle.Storage.Message(messageID))
		//		log.Debugf("Peer %d Gossiped message %d",
		//	s.tangle.Peer.ID, messageID)
	}))
	s.events.MessageDropped.Attach(events.NewClosure(func(messageID MessageID) {
		s.tangle.Storage.MessageMetadata(messageID).SetDropTime(time.Now())
	}))
	s.tangle.ApprovalManager.Events.MessageConfirmed.Attach(events.NewClosure(func(message *Message, messageMetadata *MessageMetadata, weight uint64, messageIDCounter int64) {
		s.updateChildrenReady(message.ID)
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
	}
}

func (s *ICCAScheduler) IncrementAccessMana(schedulingRate float64) {
	weights := s.tangle.WeightDistribution.Weights()
	totalWeight := config.NodesTotalWeight
	// every time something is scheduled, we add this much mana in total\
	mana := float64(10)
	for id := range s.accessMana {
		s.accessMana[id] += mana * schedulingRate * float64(weights[id]) / float64(totalWeight)
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
	// TODO: implement DRR scheduler
}

func (s *ICCAScheduler) Events() *SchedulerEvents {
	return s.events
}

// region ManaBurn Scheduler ////////////////////////////////////////////////////////////////////////////////////////////////////

type MBScheduler struct {
	tangle      *Tangle
	readyQueue  *PriorityQueue
	nonReadyMap map[MessageID]*Message
	accessMana  map[network.PeerID]float64

	events *SchedulerEvents
}

func (s *MBScheduler) Setup() {
	s.events.MessageScheduled.Attach(events.NewClosure(func(messageID MessageID) {
		s.tangle.Peer.GossipNetworkMessage(s.tangle.Storage.Message(messageID))
		//		log.Debugf("Peer %d Gossiped message %d",
		//	s.tangle.Peer.ID, messageID)
	}))
	s.events.MessageDropped.Attach(events.NewClosure(func(messageID MessageID) {
		s.tangle.Storage.MessageMetadata(messageID).SetDropTime(time.Now())
	}))
	s.tangle.ApprovalManager.Events.MessageConfirmed.Attach(events.NewClosure(func(message *Message, messageMetadata *MessageMetadata, weight uint64, messageIDCounter int64) {
		s.updateChildrenReady(message.ID)
	}))
}

func (s *MBScheduler) BurnValue() (burn float64, ok bool) {
	peerID := s.tangle.Peer.ID
	switch policy := config.BurnPolicies[peerID]; BurnPolicyType(policy) {
	case NoBurn:
		return 0.0, true
	case Anxious:
		burn = s.getNodeAccessMana(peerID)
		ok = true
		return
	case Greedy:
		burn = s.getMaxManaBurn() + config.ExtraBurn
		ok = burn <= s.getNodeAccessMana(peerID)
		return
	case RandomGreedy:
		burn = s.getMaxManaBurn() + config.ExtraBurn*rand.Float64()
		ok = burn <= s.getNodeAccessMana(peerID)
		return
	default:
		return 0.0, true
	}
}

func (s *MBScheduler) IncrementAccessMana(schedulingRate float64) {
	weights := s.tangle.WeightDistribution.Weights()
	totalWeight := config.NodesTotalWeight
	// every time something is scheduled, we add this much mana in total\
	mana := float64(10)
	for id := range s.accessMana {
		s.accessMana[id] += mana * schedulingRate * float64(weights[id]) / float64(totalWeight)
	}
}

func (s *MBScheduler) DecreaseNodeAccessMana(nodeID network.PeerID, manaIncrement float64) (newAccessMana float64) {
	s.accessMana[nodeID] -= manaIncrement
	newAccessMana = s.accessMana[nodeID]
	return newAccessMana
}

func (s *MBScheduler) getNodeAccessMana(nodeID network.PeerID) (mana float64) {
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
	}
}

func (s *MBScheduler) IsEmpty() bool {
	return s.readyQueue.Len() == 0
}

func (s *MBScheduler) ReadyLen() int {
	return s.readyQueue.Len()
}

func (s *MBScheduler) getMaxManaBurn() float64 {
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
}

// region Priority Queue ////////////////////////////////////////////////////////////////////////////////
func (h PriorityQueue) Len() int { return len(h) }
func (h PriorityQueue) Less(i, j int) bool {
	return h[i].ManaBurnValue > h[j].ManaBurnValue
}
func (h PriorityQueue) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *PriorityQueue) Push(m any) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, m.(Message))
}

func (h *PriorityQueue) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// region Issuer Queue ////////////////////////////////////////////////////////////////////////////////
func (h IssuerQueue) Len() int { return len(h) }
func (h IssuerQueue) Less(i, j int) bool {
	return float64(h[i].IssuanceTime.Nanosecond()) > float64(h[j].IssuanceTime.Nanosecond())
}
func (h IssuerQueue) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *IssuerQueue) Push(m any) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, m.(Message))
}

func (h *IssuerQueue) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// Scheduler Events ///////////////////////////////////////////////////////////////////////////////////////////////////////////

type SchedulerEvents struct {
	MessageScheduled *events.Event
	MessageDropped   *events.Event
	MessageEnqueued  *events.Event
}

func schedulerEventCaller(handler interface{}, params ...interface{}) {
	handler.(func(int, int))(params[0].(int), params[1].(int))
}
