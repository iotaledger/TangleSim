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
			MessageConfirmed:     events.NewEvent(messageIDEventCaller),
			MessageWeightUpdated: events.NewEvent(messageIDEventCaller),
		},
	}
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
		if weightByte&(1<<mod) != 0 {
			weightByte |= 1 << mod
			message.Weight += weight
			a.Events.MessageWeightUpdated.Trigger(message.ID)
			if float64(message.Weight) >= config.MessageWeightThreshold*float64(a.tangle.WeightDistribution.TotalWeight()) {
				message.ConfirmationTime = time.Now()
				a.Events.MessageConfirmed.Trigger(a.tangle.Peer.ID, message.Issuer, message.ID, message.IssuanceTime, message.ConfirmationTime, message.Weight)
			}

			for strongParentID := range message.StrongParents {
				walker.Push(strongParentID)
			}

			for weakParentID := range message.WeakParents {
				walker.Push(weakParentID)
			}
		}

		//TODO ask Hans about revisit elements, why do we need it?
	}, NewMessageIDs(messageID), true)
}

// region SolidifierEvents /////////////////////////////////////////////////////////////////////////////////////////////

type ApprovalWeightEvents struct {
	MessageConfirmed     *events.Event
	MessageWeightUpdated *events.Event
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
