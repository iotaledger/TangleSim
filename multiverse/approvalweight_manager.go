package multiverse

import (
	"math"
	"time"

	"github.com/iotaledger/hive.go/datastructure/walker"
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/config"
)

// region ApprovalManager ///////////////////////////////////////////////////////////////////////////////////////////////////

type ApprovalManager struct {
	tangle *Tangle
	Events *ApprovalWeightEvents
}

func NewApprovalManager(tangle *Tangle) *ApprovalManager {
	return &ApprovalManager{
		tangle: tangle,
		Events: &ApprovalWeightEvents{
			MessageConfirmed:     events.NewEvent(approvalEventCaller),
			MessageWeightUpdated: events.NewEvent(approvalEventCaller),
		},
	}
}

func approvalEventCaller(handler interface{}, params ...interface{}) {
	handler.(func(*Message, uint64))(params[0].(*Message), params[1].(uint64))
}

func (a *ApprovalManager) Setup() {
	a.tangle.Storage.Events.MessageStored.Attach(events.NewClosure(a.ApproveMessages))
}

func (a *ApprovalManager) ApproveMessages(messageID MessageID) {

	issuingMessage := a.tangle.Storage.messageDB[messageID]
	byteIndex := math.Floor(float64(issuingMessage.Issuer / 8))
	mod := issuingMessage.Issuer % 8
	weight := a.tangle.WeightDistribution.Weight(issuingMessage.Issuer)
	a.tangle.Utils.WalkMessagesAndMetadata(func(message *Message, messageMetadata *MessageMetadata, walker *walker.Walker) {

		weightByte := message.WeightSlice[int(byteIndex)]
		if weightByte&(1<<mod) == 0 {
			weightByte |= 1 << mod
			message.Weight += weight
			a.Events.MessageWeightUpdated.Trigger(message)
			if float64(message.Weight) >= config.MessageWeightThreshold*float64(a.tangle.WeightDistribution.TotalWeight()) {
				message.ConfirmationTime = time.Now()
				a.Events.MessageConfirmed.Trigger(message, message.Weight)
			}

			for strongParentID := range message.StrongParents {
				walker.Push(strongParentID)
			}

			for weakParentID := range message.WeakParents {
				walker.Push(weakParentID)
			}
		}

	}, NewMessageIDs(messageID), false)
}

// region SolidifierEvents /////////////////////////////////////////////////////////////////////////////////////////////

type ApprovalWeightEvents struct {
	MessageConfirmed     *events.Event
	MessageWeightUpdated *events.Event
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
