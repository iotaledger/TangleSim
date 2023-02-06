package multiverse

import (
	"math"
	"time"

	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/config"
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
	s.messageMetadataDB[message.ID] = &MessageMetadata{
		id:          message.ID,
		weightSlice: make([]byte, int(math.Ceil(float64(config.NodesCount)/8.0))),
		arrivalTime: time.Now(),
	}
	s.storeChildReferences(message.ID, s.strongChildrenDB, message.StrongParents)
	s.storeChildReferences(message.ID, s.weakChildrenDB, message.WeakParents)

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

func (s *Storage) storeChildReferences(messageID MessageID, childReferenceDB map[MessageID]MessageIDs, parents MessageIDs) {
	for parent := range parents {
		if _, exists := childReferenceDB[parent]; !exists {
			childReferenceDB[parent] = NewMessageIDs()
		}

		childReferenceDB[parent].Add(messageID)
	}
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
