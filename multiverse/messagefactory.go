package multiverse

import (
	"github.com/iotaledger/hive.go/events"
	"sync/atomic"
	"time"
)

// region MessageFactory ///////////////////////////////////////////////////////////////////////////////////////////////

type MessageFactory struct {
	tangle         *Tangle
	sequenceNumber uint64
	numberOfNodes  uint64
	Events         *MessageFactoryEvents
}

func NewMessageFactory(tangle *Tangle, numberOfNodes uint64) (messageFactory *MessageFactory) {
	return &MessageFactory{
		tangle:        tangle,
		numberOfNodes: numberOfNodes,
		Events: &MessageFactoryEvents{
			MessageCreated: events.NewEvent(messageEventCaller),
		},
	}
}

func (m *MessageFactory) CreateMessage(payload Color) (message *Message) {
	strongParents, weakParents := m.tangle.TipManager.Tips()

	msg := &Message{
		ID:             NewMessageID(),
		StrongParents:  strongParents,
		WeakParents:    weakParents,
		SequenceNumber: atomic.AddUint64(&m.sequenceNumber, 1),
		Issuer:         m.tangle.Peer.ID,
		Payload:        payload,
		IssuanceTime:   time.Now(),
	}
	m.Events.MessageCreated.Trigger(msg)
	return msg
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region MessageFactoryEvents  ///////////////////////////////////////////////////////////////////////////////////////////////////////////

type MessageFactoryEvents struct {
	MessageCreated *events.Event
}

func messageEventCaller(handler interface{}, params ...interface{}) {
	handler.(func(message *Message))(params[0].(*Message))
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
