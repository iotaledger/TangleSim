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

func (m *MessageFactory) CreateMessage(payload Color) (*Message, bool) {
	strongParents, weakParents := m.tangle.TipManager.Tips()
	issuanceTime := time.Now()
	if burn, ok := m.tangle.Scheduler.BurnValue(issuanceTime); ok {
		m.tangle.Scheduler.DecreaseNodeAccessMana(m.tangle.Peer.ID, burn) // decrease the nodes own Mana when the message is created
		message := &Message{
			ID:             NewMessageID(),
			StrongParents:  strongParents,
			WeakParents:    weakParents,
			SequenceNumber: atomic.AddUint64(&m.sequenceNumber, 1),
			Issuer:         m.tangle.Peer.ID,
			Payload:        payload,
			IssuanceTime:   issuanceTime,
			ManaBurnValue:  burn,
		}
		return message, ok
	} else {
		return nil, false
	}
}

func (m *MessageFactory) SequenceNumber() uint64 {
	return atomic.AddUint64(&m.sequenceNumber, 1)
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
