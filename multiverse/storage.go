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
	slotDB            map[SlotIndex]MessageIDs
	acceptedSlotDB    map[SlotIndex]MessageIDs
	rmc               map[SlotIndex]float64
	genesisTime       time.Time
	ATT               time.Time

	slotMutex sync.Mutex
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
		slotDB:            make(map[SlotIndex]MessageIDs),
		acceptedSlotDB:    make(map[SlotIndex]MessageIDs),
		rmc:               make(map[SlotIndex]float64),
	}
}

func (s *Storage) Setup(genesisTime time.Time) {
	s.genesisTime = genesisTime
	s.ATT = genesisTime
}

func (s *Storage) Store(message *Message) (*MessageMetadata, bool) {
	if _, exists := s.messageDB[message.ID]; exists {
		return &MessageMetadata{}, false
	}
	slotIndex := s.SlotIndex(message.IssuanceTime)
	s.slotMutex.Lock()
	defer s.slotMutex.Unlock()
	if _, exists := s.slotDB[slotIndex]; !exists {
		s.slotDB[slotIndex] = NewMessageIDs()
	}
	if _, exists := s.rmc[slotIndex]; !exists {
		s.NewRMC(slotIndex)
	}
	if message.ManaBurnValue < s.rmc[slotIndex] { // RMC will always be zero if not in ICCA+
		log.Debug("Message dropped due to Mana burn < RMC", message.Issuer, message.ManaBurnValue, s.rmc[slotIndex])
		return &MessageMetadata{}, false // don't store this message if it burns less than RMC
	}
	// store to slot storage
	s.slotDB[slotIndex].Add(message.ID)
	// store message and metadata
	s.messageDB[message.ID] = message
	messageMetadata := &MessageMetadata{
		id:                        message.ID,
		nodePreAcceptanceBitmap:   make([]byte, int(math.Ceil(float64(config.Params.NodesCount)/8.0))),
		nodeAcceptanceBitmap:      make([]byte, int(math.Ceil(float64(config.Params.NodesCount)/8.0))),
		nodePreConfirmationBitmap: make([]byte, int(math.Ceil(float64(config.Params.NodesCount)/8.0))),
		nodeConfirmationBitmap:    make([]byte, int(math.Ceil(float64(config.Params.NodesCount)/8.0))),
		arrivalTime:               time.Now(),
		ready:                     false,
	}
	// check if this should be orphaned
	if s.TooOld(message) {
		messageMetadata.SetOrphanTime(time.Now())
	}
	s.messageMetadataDB[message.ID] = messageMetadata
	// store child references
	s.storeChildReferences(message.ID, s.strongChildrenDB, message.StrongParents)
	s.storeChildReferences(message.ID, s.weakChildrenDB, message.WeakParents)
	return messageMetadata, true
}

func (s *Storage) Message(messageID MessageID) (message *Message) {
	return s.messageDB[messageID]
}

func (s *Storage) MessageMetadata(messageID MessageID) (messageMetadata *MessageMetadata) {
	return s.messageMetadataDB[messageID]
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

func (s *Storage) SlotIndex(messageTime time.Time) SlotIndex {
	timeSinceGenesis := messageTime.Sub(s.genesisTime)
	return SlotIndex(float64(timeSinceGenesis) / (float64(config.Params.SlotTime) * float64(config.Params.SlowdownFactor)))
}

func (s *Storage) Slot(index SlotIndex) MessageIDs {
	return s.slotDB[index]
}

func (s *Storage) AcceptedSlot(index SlotIndex) MessageIDs {
	return s.acceptedSlotDB[index]
}

// Get the messages count per slot
func (s *Storage) MessagesCountPerSlot() map[SlotIndex]int {
	s.slotMutex.Lock()
	defer s.slotMutex.Unlock()
	counts := make(map[SlotIndex]int)
	for slotIndex, messages := range s.slotDB {
		counts[slotIndex] = len(messages)
	}
	return counts
}

// Get the total messages counts in range of slots
func (s *Storage) MessagesCountInRange(startSlotIndex SlotIndex, endSlotIndex SlotIndex) int {
	count := 0
	for slotIndex := startSlotIndex; slotIndex < endSlotIndex; slotIndex++ {
		if _, exists := s.slotDB[slotIndex]; !exists {
			continue
		}
		count += len(s.slotDB[slotIndex])
	}
	return count
}

func (s *Storage) RMC(slotIndex SlotIndex) float64 {
	s.slotMutex.Lock()
	defer s.slotMutex.Unlock()
	if _, exists := s.slotDB[slotIndex]; !exists {
		s.NewRMC(slotIndex)
		s.slotDB[slotIndex] = NewMessageIDs()
	}
	return s.rmc[slotIndex]
}

// func (s *Storage) NewRMC(currentSlotIndex SlotIndex) {
// 	currentSlotStartTime := s.genesisTime.Add(time.Duration(float64(currentSlotIndex)*float64(config.Params.SlowdownFactor)) * config.Params.SlotTime)
// 	if config.Params.SchedulerType != "ICCA+" {
// 		s.rmc[currentSlotIndex] = 0.0
// 		return
// 	}
// 	if currentSlotIndex == SlotIndex(0) {
// 		s.rmc[currentSlotIndex] = config.Params.InitialRMC
// 		return
// 	}
// 	s.rmc[currentSlotIndex] = s.rmc[currentSlotIndex-SlotIndex(1)] // keep RMC the same by default
// 	if currentSlotStartTime.After(s.genesisTime.Add(config.Params.RMCTime * time.Duration(config.Params.SlowdownFactor))) {
// 		n := len(s.AcceptedSlot(s.SlotIndex(currentSlotStartTime.Add(-config.Params.RMCTime)))) // number of messages k slots in the past
// 		if n < int(config.Params.LowerRMCThreshold) {
// 			s.rmc[currentSlotIndex] = math.Max(config.Params.RMCmin, s.rmc[currentSlotIndex]*config.Params.AlphaRMC)
// 		} else if n > int(config.Params.UpperRMCThreshold) {
// 			s.rmc[currentSlotIndex] = math.Min(config.Params.RMCmax, s.rmc[currentSlotIndex]*config.Params.BetaRMC)
// 		}
// 	}
// }

func (s *Storage) NewRMC(currentSlotIndex SlotIndex) {
	currentSlotStartTime := s.genesisTime.Add(time.Duration(float64(currentSlotIndex)*float64(config.Params.SlowdownFactor)) * config.Params.SlotTime)
	if config.Params.SchedulerType != "ICCA+" {
		s.rmc[currentSlotIndex] = 0.0
		return
	}
	if currentSlotIndex == SlotIndex(0) {
		s.rmc[currentSlotIndex] = config.Params.InitialRMC
		return
	}
	s.rmc[currentSlotIndex] = s.rmc[currentSlotIndex-SlotIndex(1)] // keep RMC the same by default

	// Update the RMC every RMCPeriodUpdate
	if currentSlotStartTime.After(s.genesisTime.Add(config.Params.RMCTime * time.Duration(config.Params.SlowdownFactor))) {
		// log.Debugf("CurrentSlotIndex %d", currentSlotIndex)
		if int(currentSlotIndex)%config.Params.RMCPeriodUpdate == 0 {
			traffic := s.MessagesCountInRange(
				currentSlotIndex-SlotIndex(config.Params.MinCommittableAge/config.Params.SlotTime)-SlotIndex(config.Params.RMCPeriodUpdate),
				currentSlotIndex-SlotIndex(config.Params.MinCommittableAge/config.Params.SlotTime)) / config.Params.RMCPeriodUpdate

			// currentSlotIndex-SlotIndex(config.Params.RMCTime/config.Params.SlotTime)-SlotIndex(config.Params.RMCPeriodUpdate),
			// currentSlotIndex-SlotIndex(config.Params.RMCTime/config.Params.SlotTime))

			// a := currentSlotIndex-SlotIndex(config.Params.RMCTime/config.Params.SlotTime)-SlotIndex(config.Params.RMCPeriodUpdate)
			// b := currentSlotIndex-SlotIndex(config.Params.RMCTime/config.Params.SlotTime)

			// traffic := 0
			// for i := 0; i < config.Params.RMCPeriodUpdate; i++ {
			// 	// traffic += len(s.AcceptedSlot(s.SlotIndex(currentSlotStartTime.Add(-config.Params.MinCommittableAge-time.Duration(i) * config.Params.SlotTime))))
			// 	traffic += len(s.AcceptedSlot(s.SlotIndex(currentSlotStartTime.Add(-config.Params.RMCTime -time.Duration(i) * config.Params.SlotTime)))) // number of messages k slots in the past
			// }
			// MessagesCountInRange
			// log.Debugf("Traffic: %d, Slot: %d, Slot a: %d, Slot b: %d", traffic, currentSlotIndex, a, b)
			// traffic = traffic
			// log.Debugf("Enter Branch, traffic after division: %d", traffic)

			// Modified
			// if traffic < config.Params.RMCPeriodUpdate*int(config.Params.LowerRMCThreshold) {
			// 	s.rmc[currentSlotIndex] = math.Max(config.Params.RMCmin, s.rmc[currentSlotIndex]*config.Params.AlphaRMC)
			// } else if traffic > config.Params.RMCPeriodUpdate*int(config.Params.UpperRMCThreshold) {
			// 	s.rmc[currentSlotIndex] = math.Min(config.Params.RMCmax, s.rmc[currentSlotIndex]*config.Params.BetaRMC)
			// }

			// log.Debugf("Traffic: %d", traffic)
			if traffic < int(config.Params.LowerRMCThreshold) {
				for i := 0; i < config.Params.RMCPeriodUpdate; i++ {
					s.rmc[currentSlotIndex+SlotIndex(i)] = math.Max(
						s.rmc[currentSlotIndex-SlotIndex(1)]-config.Params.RMCdecrease, config.Params.RMCmin)
				}
				// log.Debugf("LOW!!!!, rmc = %f", s.rmc[currentSlotIndex])
			} else if traffic > int(config.Params.UpperRMCThreshold) {
				for i := 0; i < config.Params.RMCPeriodUpdate; i++ {
					s.rmc[currentSlotIndex+SlotIndex(i)] = math.Min(
						s.rmc[currentSlotIndex-SlotIndex(1)]+config.Params.RMCincrease, config.Params.RMCmax)
				}
				// log.Debugf("HIGH!!!!, rmc = %f", s.rmc[currentSlotIndex])
			} else {
				for i := 0; i < config.Params.RMCPeriodUpdate; i++ {
					s.rmc[currentSlotIndex+SlotIndex(i)] = s.rmc[currentSlotIndex-SlotIndex(1)]
				}
			}
		}
	}
}

func (s *Storage) TooOld(message *Message) bool {
	return message.IssuanceTime.Before(s.ATT.Add(-config.Params.MinCommittableAge * time.Duration(config.Params.SlowdownFactor)))
}

func (s *Storage) AddToAcceptedSlot(message *Message) {
	s.slotMutex.Lock()
	defer s.slotMutex.Unlock()
	slotIndex := s.SlotIndex(message.IssuanceTime)
	if _, exists := s.acceptedSlotDB[slotIndex]; !exists {
		s.acceptedSlotDB[slotIndex] = NewMessageIDs()
	}
	// store to accepted slot storage
	s.acceptedSlotDB[slotIndex].Add(message.ID)
	// update accepted tange time
	if message.IssuanceTime.After(s.ATT) {
		s.ATT = message.IssuanceTime
	}
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
