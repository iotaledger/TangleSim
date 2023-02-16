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

	messageDB         map[MessageID]*Message
	messageMetadataDB map[MessageID]*MessageMetadata
	strongChildrenDB  map[MessageID]MessageIDs
	weakChildrenDB    map[MessageID]MessageIDs

	// mutex sync.RWMutex
}

func NewStorage() (storage *Storage) {
	return &Storage{
		Events: &StorageEvents{
			MessageStored: events.NewEvent(messageIDEventCaller),
		},

		messageDB:         make(map[MessageID]*Message),
		messageMetadataDB: make(map[MessageID]*MessageMetadata),
		strongChildrenDB:  make(map[MessageID]MessageIDs),
		weakChildrenDB:    make(map[MessageID]MessageIDs),
	}
}

func (s *Storage) Store(message *Message) {
	// s.mutex.Lock()
	// s.mutex.Unlock()
	if _, exists := s.messageDB[message.ID]; exists {
		return
	}

	s.messageDB[message.ID] = message
	s.messageMetadataDB[message.ID] = &MessageMetadata{
		id:          message.ID,
		weightSlice: make([]byte, int(math.Ceil(float64(config.NodesCount)/8.0))),
		arrivalTime: time.Now(),
		ready:       false,
	}
	s.storeChildReferences(message.ID, s.strongChildrenDB, message.StrongParents)
	s.storeChildReferences(message.ID, s.weakChildrenDB, message.WeakParents)
	s.Events.MessageStored.Trigger(message.ID)
}

func (s *Storage) Message(messageID MessageID) (message *Message) {
	// s.mutex.RLock()
	// defer s.mutex.RUnlock()
	return s.messageDB[messageID]
}

func (s *Storage) MessageMetadata(messageID MessageID) (messageMetadata *MessageMetadata) {
	// s.mutex.RLock()
	// defer s.mutex.RUnlock()
	return s.messageMetadataDB[messageID]
}

func (s *Storage) StrongChildren(messageID MessageID) (strongChildren MessageIDs) {
	// s.mutex.RLock()
	// defer s.mutex.RUnlock()
	return s.strongChildrenDB[messageID]
}

func (s *Storage) WeakChildren(messageID MessageID) (weakChildren MessageIDs) {
	// s.mutex.RLock()
	// defer s.mutex.RUnlock()
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
