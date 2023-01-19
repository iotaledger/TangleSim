package multiverse

import (
	"sync/atomic"
	"time"
)

// region MessageFactory ///////////////////////////////////////////////////////////////////////////////////////////////

type MessageFactory struct {
	tangle         *Tangle
	sequenceNumber uint64
	numberOfNodes  uint64
}

func NewMessageFactory(tangle *Tangle, numberOfNodes uint64) (messageFactory *MessageFactory) {
	return &MessageFactory{
		tangle:        tangle,
		numberOfNodes: numberOfNodes,
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
		IssuanceTime:   time.Now(),
		ManaBurnValue:  BurnMana(m.tangle),
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
