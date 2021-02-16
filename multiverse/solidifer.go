package multiverse

import (
	"github.com/iotaledger/hive.go/datastructure/walker"
	"github.com/iotaledger/hive.go/events"
)

// region Solidifier ///////////////////////////////////////////////////////////////////////////////////////////////////

type Solidifier struct {
	tangle *Tangle
	Events *SolidifierEvents
}

func NewSolidifier(tangle *Tangle) *Solidifier {
	return &Solidifier{
		tangle: tangle,
		Events: &SolidifierEvents{
			MessageSolid:   events.NewEvent(messageIDEventCaller),
			MessageMissing: events.NewEvent(messageIDEventCaller),
		},
	}
}

func (s *Solidifier) Setup() {
	s.tangle.Storage.Events.MessageStored.Attach(events.NewClosure(s.Solidify))
}

func (s *Solidifier) Solidify(messageID MessageID) {
	s.tangle.Utils.WalkMessagesAndMetadata(func(message *Message, messageMetadata *MessageMetadata, walker *walker.Walker) {
		if !s.messageSolid(message) {
			return
		}

		if !messageMetadata.SetSolid(true) {
			return
		}

		s.Events.MessageSolid.Trigger(message.ID)

		for strongChildID := range s.tangle.Storage.StrongChildren(message.ID) {
			walker.Push(strongChildID)
		}
		for weakChildID := range s.tangle.Storage.WeakChildren(message.ID) {
			walker.Push(weakChildID)
		}
	}, NewMessageIDs(messageID), true)
}

func (s *Solidifier) messageSolid(message *Message) (isSolid bool) {
	isSolid = true
	if !s.parentsSolid(message.StrongParents) {
		isSolid = false
	}
	if !s.parentsSolid(message.WeakParents) {
		isSolid = false
	}

	return
}

func (s *Solidifier) parentsSolid(parentMessageIDs MessageIDs) (parentsSolid bool) {
	parentsSolid = true
	for parentMessageID := range parentMessageIDs {
		if parentMessageID == Genesis {
			continue
		}

		parentMessageMetadata := s.tangle.Storage.MessageMetadata(parentMessageID)
		if parentMessageMetadata == nil {
			s.Events.MessageMissing.Trigger(parentMessageID)

			parentsSolid = false
			continue
		}

		if !parentMessageMetadata.Solid() {
			parentsSolid = false
		}
	}

	return
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region SolidifierEvents /////////////////////////////////////////////////////////////////////////////////////////////

type SolidifierEvents struct {
	MessageSolid   *events.Event
	MessageMissing *events.Event
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
