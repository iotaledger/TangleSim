package multiverse

import (
	"sync/atomic"
	"time"

	"github.com/iotaledger/hive.go/types"
	"github.com/iotaledger/multivers-simulation/network"
)

// region Message //////////////////////////////////////////////////////////////////////////////////////////////////////

type Message struct {
	ID               MessageID
	StrongParents    MessageIDs
	WeakParents      MessageIDs
	SequenceNumber   uint64
	Issuer           network.PeerID
	Payload          Color
	WeightSlice      []byte
	IssuanceTime     time.Time
	confirmationTime time.Time
}

// endregion Message ///////////////////////////////////////////////////////////////////////////////////////////////////

// region MessageMetadata //////////////////////////////////////////////////////////////////////////////////////////////

type MessageMetadata struct {
	id             MessageID
	solid          bool
	inheritedColor Color
}

func (m *MessageMetadata) ID() (messageID MessageID) {
	return m.id
}

func (m *MessageMetadata) SetSolid(solid bool) (modified bool) {
	if solid == m.solid {
		return
	}

	m.solid = solid
	modified = true

	return
}

func (m *MessageMetadata) Solid() (solid bool) {
	return m.solid
}

func (m *MessageMetadata) SetInheritedColor(color Color) (modified bool) {
	if color == m.inheritedColor {
		return
	}

	m.inheritedColor = color
	modified = true

	return
}

func (m *MessageMetadata) InheritedColor() (color Color) {
	return m.inheritedColor
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region MessageRequest ///////////////////////////////////////////////////////////////////////////////////////////////

type MessageRequest struct {
	MessageID MessageID
	Issuer    network.PeerID
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region MessageID ////////////////////////////////////////////////////////////////////////////////////////////////////

type MessageID int64

var (
	Genesis MessageID

	messageIDCounter int64
)

func NewMessageID() MessageID {
	return MessageID(atomic.AddInt64(&messageIDCounter, 1))
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region MessageIDs ///////////////////////////////////////////////////////////////////////////////////////////////////

type MessageIDs map[MessageID]types.Empty

func NewMessageIDs(messageIDs ...MessageID) (newMessageIDs MessageIDs) {
	newMessageIDs = make(MessageIDs)
	for _, messageID := range messageIDs {
		newMessageIDs[messageID] = types.Void
	}

	return
}

func (m MessageIDs) Add(messageID MessageID) {
	m[messageID] = types.Void
}

func (m MessageIDs) Trim(length int) {
	counter := 0
	for messageID := range m {
		if counter == length {
			delete(m, messageID)
		}
		counter++
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region Color ////////////////////////////////////////////////////////////////////////////////////////////////////////

type Color int64

func (c Color) String() string {
	switch c {
	case 0:
		return "Color(Undefined)"
	case 1:
		return "Color(Blue)"
	case 2:
		return "Color(Red)"
	case 3:
		return "Color(Green)"
	default:
		return "Color(Unknown)"
	}
}

var (
	UndefinedColor Color
	Blue           = Color(1)
	Red            = Color(2)
	Green          = Color(3)
)

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
