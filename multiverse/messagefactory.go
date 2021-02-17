package multiverse

import (
	"sync/atomic"
)

// region MessageFactory ///////////////////////////////////////////////////////////////////////////////////////////////

type MessageFactory struct {
	tangle         *Tangle
	sequenceNumber uint64
}

func NewMessageFactory(tangle *Tangle) (messageFactory *MessageFactory) {
	return &MessageFactory{
		tangle: tangle,
	}
}

func (m *MessageFactory) CreateMessage(payload Color) (message *Message) {
	strongParents, weakParents := m.tangle.TipManager.Tips()

	return &Message{
		ID:             NewMessageID(),
		StrongParents:  strongParents,
		WeakParents:    weakParents,
		SequenceNumber: atomic.AddUint64(&m.sequenceNumber, 1),
		Issuer:         m.tangle.Peer.ID,
		Payload:        payload,
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
