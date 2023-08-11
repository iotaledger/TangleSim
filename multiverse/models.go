package multiverse

import (
	"sync/atomic"
	"time"

	"github.com/iotaledger/hive.go/types"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/network"
)

// region Slot //////////////////////////////////////////////////////////////////////////////////////////////////////

type SlotIndex int

// endregion Slot ///////////////////////////////////////////////////////////////////////////////////////////////////

// region Message //////////////////////////////////////////////////////////////////////////////////////////////////////

type Message struct {
	Validation bool

	ID             MessageID
	StrongParents  MessageIDs
	WeakParents    MessageIDs
	SequenceNumber uint64
	Issuer         network.PeerID
	Payload        Color
	IssuanceTime   time.Time
	ManaBurnValue  float64
}

// endregion Message ///////////////////////////////////////////////////////////////////////////////////////////////////

// region MessageMetadata //////////////////////////////////////////////////////////////////////////////////////////////

type MessageMetadata struct {
	id               MessageID
	solid            bool
	ready            bool
	inheritedColor   Color
	weightSlice      []byte
	weight           uint64
	confirmationTime time.Time
	orphanTime       time.Time
	arrivalTime      time.Time
	enqueueTime      time.Time
	scheduleTime     time.Time
	dropTime         time.Time
}

func (m *MessageMetadata) ArrivalTime() time.Time {
	return m.arrivalTime
}

func (m *MessageMetadata) WeightByte(index int) byte {
	return m.weightSlice[index]
}

func (m *MessageMetadata) SetWeightByte(index int, weight byte) {
	m.weightSlice[index] = weight
}

func (m *MessageMetadata) SetWeightSlice(weightSlice []byte) {
	m.weightSlice = weightSlice
}

func (m *MessageMetadata) Weight() uint64 {
	return m.weight
}

func (m *MessageMetadata) AddWeight(weight uint64) {
	m.weight += weight
}

func (m *MessageMetadata) SetWeight(weight uint64) {
	m.weight = weight
}

func (m *MessageMetadata) ConfirmationTime() time.Time {
	return m.confirmationTime
}

func (m *MessageMetadata) SetConfirmationTime(confirmationTime time.Time) {
	m.confirmationTime = confirmationTime
}

func (m *MessageMetadata) SetOrphanTime(orphanTime time.Time) {
	m.orphanTime = orphanTime
}

func (m *MessageMetadata) SetEnqueueTime(enqueueTime time.Time) {
	m.enqueueTime = enqueueTime
}

func (m *MessageMetadata) SetScheduleTime(scheduleTime time.Time) {
	m.scheduleTime = scheduleTime
}

func (m *MessageMetadata) SetDropTime(dropTime time.Time) {
	m.dropTime = dropTime
}

func (m *MessageMetadata) ID() (messageID MessageID) {
	return m.id
}

func (m *MessageMetadata) Ready() bool {
	return m.ready
}

func (m *MessageMetadata) SetReady() {
	m.ready = true
}

func (m *MessageMetadata) Scheduled() bool {
	return !m.scheduleTime.IsZero()
}

func (m *MessageMetadata) Confirmed() bool {
	return !m.confirmationTime.IsZero()
}

func (m *MessageMetadata) Orphaned() bool {
	return !m.orphanTime.IsZero()
}

func (m *MessageMetadata) Eligible() bool { // a message is ready if all parents are eligible = either scheduled or confirmed
	return m.Scheduled() || (m.Confirmed() && config.Params.ConfEligible)
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

// Trim the MessageIDs to only retain `length` size
func (m MessageIDs) Trim(length int) {
	counter := 0
	for messageID := range m {
		if counter == length {
			delete(m, messageID)
			continue
		}
		counter++
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region Color ////////////////////////////////////////////////////////////////////////////////////////////////////////

// The Color is used to ease of observation of Peer opinions and the ownOpinion based on the approvalWeights
// The maxOpinion is the Opinion with the highest Color value and the maxApprovalWeight
//
// The approvalWeights stores the accumulated weights of each Color for messages
//   - The message will have an associated Color inherited from its parents
//   - The Color of a message is assigned from `IssuePayload`
//   - The strongTips/weakTips will be selected from the TipSet[ownOpinion]
//
// The different color values are used as a tie breaker, i.e., when 2 colors have the same weight, the larger color value
// opinion will be regarded as the ownOpinion. Each color simply represents a perception of a certain state of a tangle
// where different conflicts are approved.
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

func ColorFromInt(i int) Color {
	switch i {
	case 0:
		return UndefinedColor
	case 1:
		return Blue
	case 2:
		return Red
	case 3:
		return Green
	default:
		return UndefinedColor
	}
}

func ColorFromStr(s string) Color {
	switch s {
	case "":
		return UndefinedColor
	case "B":
		return Blue
	case "R":
		return Red
	case "G":
		return Green
	default:
		return UndefinedColor
	}
}

var (
	UndefinedColor Color
	Blue           = Color(1)
	Red            = Color(2)
	Green          = Color(3)
)

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
