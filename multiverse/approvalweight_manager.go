package multiverse

import (
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
			MessagePreAccepted:                  events.NewEvent(approvalEventCaller),
			MessageAccepted:                     events.NewEvent(approvalEventCaller),
			MessagePreConfirmed:                 events.NewEvent(approvalEventCaller),
			MessageConfirmed:                    events.NewEvent(approvalEventCaller),
			MessagePreAcceptanceWeightUpdated:   events.NewEvent(approvalEventCaller),
			MessageAcceptanceWeightUpdated:      events.NewEvent(approvalEventCaller),
			MessagePreConfirmationWeightUpdated: events.NewEvent(approvalEventCaller),
			MessageConfirmationWeightUpdated:    events.NewEvent(approvalEventCaller),
			MessageWitnessWeightUpdated:         events.NewEvent(witnessWeightEventCaller),
		},
	}
}

func approvalEventCaller(handler interface{}, params ...interface{}) {
	handler.(func(*Message, *MessageMetadata, uint64, int64))(params[0].(*Message), params[1].(*MessageMetadata), params[2].(uint64), params[3].(int64))
}

func witnessWeightEventCaller(handler interface{}, params ...interface{}) {
	handler.(func(*Message, uint64))(params[0].(*Message), params[1].(uint64))
}

func (a *ApprovalManager) Setup() {
	a.tangle.Solidifier.Events.MessageSolid.Attach(events.NewClosure(a.TrackAcceptanceWeight))
}

// TODO (confirmation): Add a similar function: 2) TrackPreAcceptanceWeight
// TODO (confirmation): Add a similar function: 3) TrackPreConfirmationWeight
// TODO (confirmation): Add a similar function: 4) TrackConfirmationWeight
func (a *ApprovalManager) TrackAcceptanceWeight(messageID MessageID) {
	issuingMessage := a.tangle.Storage.Message(messageID)
	byteIndex := issuingMessage.Issuer / 8
	nodeBitmapBitLocation := issuingMessage.Issuer % 8

	if !issuingMessage.Validation {
		return
	}

	weight := a.tangle.WeightDistribution.Weight(issuingMessage.Issuer)
	a.tangle.Utils.WalkMessagesAndMetadata(func(message *Message, messageMetadata *MessageMetadata, walker *walker.Walker) {
		if int(a.tangle.Peer.ID) == config.Params.MonitoredWitnessWeightPeer && messageMetadata.ID() == MessageID(config.Params.MonitoredWitnessWeightMessageID) {
			// log.Infof("Peer %d Message %d Witness Weight %d", a.tangle.Peer.ID, messageMetadata.id, messageMetadata.weight)
			a.Events.MessageWitnessWeightUpdated.Trigger(message, messageMetadata.Weight())
		}

		nodeBitmapByte := messageMetadata.PreAcceptedBitmapByte(int(byteIndex))
		if nodeBitmapByte&(1<<nodeBitmapBitLocation) == 0 {
			nodeBitmapByte |= 1 << nodeBitmapBitLocation
			messageMetadata.SetPreAcceptedBitmapByte(int(byteIndex), nodeBitmapByte)
			messageMetadata.AddWeight(weight)
			a.Events.MessagePreAcceptanceWeightUpdated.Trigger(message, messageMetadata, messageMetadata.Weight())
			if float64(messageMetadata.Weight()) >= config.Params.ConfirmationThreshold*float64(a.tangle.WeightDistribution.TotalWeight()) &&
				!messageMetadata.Confirmed() && !messageMetadata.Orphaned() {
				// check if this should be orphaned
				now := time.Now()
				if a.tangle.Storage.TooOld(message) {
					messageMetadata.SetOrphanTime(now)
				} else {
					messageMetadata.SetConfirmationTime(now)
					a.Events.MessageConfirmed.Trigger(message, messageMetadata, messageMetadata.Weight(), messageIDCounter)
				}
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

// region ApprovalWeightEvents /////////////////////////////////////////////////////////////////////////////////////////////

type ApprovalWeightEvents struct {
	MessagePreAccepted                  *events.Event
	MessageAccepted                     *events.Event
	MessagePreConfirmed                 *events.Event
	MessageConfirmed                    *events.Event
	MessagePreAcceptanceWeightUpdated   *events.Event
	MessageAcceptanceWeightUpdated      *events.Event
	MessagePreConfirmationWeightUpdated *events.Event
	MessageConfirmationWeightUpdated    *events.Event
	MessageWitnessWeightUpdated         *events.Event
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
