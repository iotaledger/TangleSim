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

func (m *MessageFactory) CreateMessage(payload Color) (message *Message, ok bool) {
	strongParents, weakParents := m.tangle.TipManager.Tips()
	burn := BurnMana(m.tangle)
	manabalance := m.tangle.Scheduler.tangle.Scheduler.GetNodeAccessMana(m.tangle.Peer.ID)
	if burn <= manabalance {
		m.tangle.Scheduler.DecreaseNodeAccessMana(m.tangle.Peer.ID, burn) // decrease the nodes own Mana when the message is created
		message = &Message{
			ID:             NewMessageID(),
			StrongParents:  strongParents,
			WeakParents:    weakParents,
			SequenceNumber: atomic.AddUint64(&m.sequenceNumber, 1),
			Issuer:         m.tangle.Peer.ID,
			Payload:        payload,
			IssuanceTime:   time.Now(),
			ManaBurnValue:  burn,
		}
		return message, true
	} else {
		return nil, false
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
