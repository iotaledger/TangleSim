package multiverse

import (
	"github.com/iotaledger/hive.go/events"
)

// region Storage //////////////////////////////////////////////////////////////////////////////////////////////////////

type Storage struct {
	Events *StorageEvents

	tangle            *Tangle
	messageDB         map[MessageID]*Message
	messageMetadataDB map[MessageID]*MessageMetadata
	strongChildrenDB  map[MessageID]MessageIDs
	weakChildrenDB    map[MessageID]MessageIDs
}

func NewStorage(tangle *Tangle) (storage *Storage) {
	return &Storage{
		Events: &StorageEvents{
			MessageStored: events.NewEvent(messageIDEventCaller),
		},

		tangle:            tangle,
		messageDB:         make(map[MessageID]*Message),
		messageMetadataDB: make(map[MessageID]*MessageMetadata),
		strongChildrenDB:  make(map[MessageID]MessageIDs),
		weakChildrenDB:    make(map[MessageID]MessageIDs),
	}
}

func (s *Storage) Store(message *Message) {
	if _, exists := s.messageDB[message.ID]; exists {
		return
	}

	s.messageDB[message.ID] = message
	s.messageMetadataDB[message.ID] = &MessageMetadata{id: message.ID}
	s.strongChildrenDB[message.ID] = message.StrongParents
	s.weakChildrenDB[message.ID] = message.WeakParents

	s.Events.MessageStored.Trigger(message.ID)
}

func (s *Storage) Message(messageID MessageID) (message *Message) {
	message = s.messageDB[messageID]
	return
}

func (s *Storage) MessageMetadata(messageID MessageID) (messageMetadata *MessageMetadata) {
	messageMetadata = s.messageMetadataDB[messageID]
	return
}

func (s *Storage) StrongChildren(messageID MessageID) (strongChildren MessageIDs) {
	return s.strongChildrenDB[messageID]
}

func (s *Storage) WeakChildren(messageID MessageID) (weakChildren MessageIDs) {
	return s.weakChildrenDB[messageID]
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region StorageEvents ////////////////////////////////////////////////////////////////////////////////////////////////

type StorageEvents struct {
	MessageStored *events.Event
}

func messageIDEventCaller(handler interface{}, params ...interface{}) {
	handler.(func(MessageID))(params[0].(MessageID))
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
