package multiverse

import (
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
	s.tangle.Storage.Events.MessageStored.Attach(events.NewClosure(func(messageID MessageID, message *Message, messageMetadata *MessageMetadata) {
		s.Solidify(messageID)
	}))
}

func (s *Solidifier) Solidify(messageID MessageID) {
	message := s.tangle.Storage.Message(messageID)
	// if message is not solid, nothing more to do
	if !s.messageSolid(message) {
		return
	}
	messageMetadata := s.tangle.Storage.MessageMetadata(messageID)
	// if message was already solid, nothing more to do
	if !messageMetadata.SetSolid(true) {
		return
	}
	// if message was not already solid, make sure future cone is solid too.
	s.Events.MessageSolid.Trigger(message.ID)
	strongChildrenIDs := s.tangle.Storage.StrongChildren(message.ID)
	for strongChildID := range strongChildrenIDs {
		s.Solidify(strongChildID)
	}
	weakChildrenIDs := s.tangle.Storage.WeakChildren(message.ID)
	for weakChildID := range weakChildrenIDs {
		s.Solidify(weakChildID)
	}

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
