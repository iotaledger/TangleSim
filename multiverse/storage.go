package multiverse

import (
	"math"
	"sync"
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

	sync.RWMutex
}

func NewStorage() (storage *Storage) {
	return &Storage{
		Events: &StorageEvents{
			MessageStored: events.NewEvent(messageEventCaller),
		},

		messageDB:         make(map[MessageID]*Message),
		messageMetadataDB: make(map[MessageID]*MessageMetadata),
		strongChildrenDB:  make(map[MessageID]MessageIDs),
		weakChildrenDB:    make(map[MessageID]MessageIDs),
	}
}

func (s *Storage) Store(message *Message) {
	messageMetadata := s.storeMessage(message)
	s.storeStrongChildren(message.ID, message.StrongParents)
	s.storeWeakChildren(message.ID, message.WeakParents)
	s.Events.MessageStored.Trigger(message.ID, message, messageMetadata)
}

func (s *Storage) storeMessage(message *Message) *MessageMetadata {
	s.Lock()
	defer s.Unlock()
	if _, exists := s.messageDB[message.ID]; exists {
		return nil
	}

	s.messageDB[message.ID] = message
	messageMetadata := &MessageMetadata{
		id:          message.ID,
		weightSlice: make([]byte, int(math.Ceil(float64(config.NodesCount)/8.0))),
		arrivalTime: time.Now(),
		ready:       false,
	}
	s.messageMetadataDB[message.ID] = messageMetadata
	return messageMetadata
}

func (s *Storage) Message(messageID MessageID) (message *Message) {
	s.RLock()
	defer s.RUnlock()
	return s.messageDB[messageID]
}

func (s *Storage) MessageMetadata(messageID MessageID) (messageMetadata *MessageMetadata) {
	s.RLock()
	defer s.RUnlock()
	return s.messageMetadataDB[messageID]
}

func (s *Storage) StrongChildren(messageID MessageID) (strongChildren MessageIDs) {
	s.RLock()
	defer s.RUnlock()
	return s.strongChildrenDB[messageID]
}

func (s *Storage) WeakChildren(messageID MessageID) (weakChildren MessageIDs) {
	s.RLock()
	defer s.RUnlock()
	return s.weakChildrenDB[messageID]
}

func (s *Storage) storeStrongChildren(messageID MessageID, parents MessageIDs) {
	s.Lock()
	defer s.Unlock()
	for parent := range parents {
		if _, exists := s.strongChildrenDB[parent]; !exists {
			s.strongChildrenDB[parent] = NewMessageIDs()
		}

		s.strongChildrenDB[parent].Add(messageID)
	}
}

func (s *Storage) storeWeakChildren(messageID MessageID, parents MessageIDs) {
	s.Lock()
	defer s.Unlock()
	for parent := range parents {
		if _, exists := s.weakChildrenDB[parent]; !exists {
			s.weakChildrenDB[parent] = NewMessageIDs()
		}

		s.weakChildrenDB[parent].Add(messageID)
	}
}

func (s *Storage) isReady(messageID MessageID) bool {
	if !s.MessageMetadata(messageID).Solid() {
		return false
	}
	message := s.Message(messageID)
	for strongParentID := range message.StrongParents {
		if strongParentID == Genesis {
			continue
		}
		strongParentMetadata := s.MessageMetadata(strongParentID)
		if strongParentMetadata == nil {
			panic("Strong Parent Metadata is empty")
		}
		if !strongParentMetadata.Eligible() {
			return false
		}
	}
	for weakParentID := range message.WeakParents {
		weakParentMetadata := s.MessageMetadata(weakParentID)
		if weakParentID == Genesis {
			continue
		}
		if !weakParentMetadata.Eligible() {
			return false
		}
	}
	return true
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region StorageEvents ////////////////////////////////////////////////////////////////////////////////////////////////

type StorageEvents struct {
	MessageStored *events.Event
}

func messageEventCaller(handler interface{}, params ...interface{}) {
	handler.(func(MessageID, *Message, *MessageMetadata))(params[0].(MessageID), params[1].(*Message), params[2].(*MessageMetadata))
}

func messageIDEventCaller(handler interface{}, params ...interface{}) {
	handler.(func(MessageID))(params[0].(MessageID))
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
