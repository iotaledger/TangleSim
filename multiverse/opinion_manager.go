package multiverse

import (
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/network"
)

// region OpinionManager ///////////////////////////////////////////////////////////////////////////////////////////////

type OpinionManagerInterface interface {
	Events() *OpinionManagerEvents
	ApprovalWeights() map[Color]uint64
	Setup()
	FormOpinion(messageID MessageID)
	Opinion() Color
	SetOpinion(opinion Color)
	WeightsUpdated()
	UpdateWeights(messageID MessageID) (updated bool)
	UpdateConfirmation(oldOpinion Color, maxOpinion Color)
	Tangle() *Tangle
}

type OpinionManager struct {
	events *OpinionManagerEvents

	tangle          *Tangle
	ownOpinion      Color
	peerOpinions    map[network.PeerID]*Opinion
	approvalWeights map[Color]uint64
	colorConfirmed  bool
}

func NewOpinionManager(tangle *Tangle) (opinionManager *OpinionManager) {
	return &OpinionManager{
		events: &OpinionManagerEvents{
			OpinionFormed:             events.NewEvent(messageIDEventCaller),
			OpinionChanged:            events.NewEvent(opinionChangedEventHandler),
			ApprovalWeightUpdated:     events.NewEvent(approvalWeightUpdatedHandler),
			MinConfirmedWeightUpdated: events.NewEvent(approvalWeightUpdatedHandler),
			ColorConfirmed:            events.NewEvent(colorEventHandler),
			ColorUnconfirmed:          events.NewEvent(reorgEventHandler),
			GodMessageProcessed:       events.NewEvent(GodMessageProcessedHandler),
		},

		tangle:          tangle,
		peerOpinions:    make(map[network.PeerID]*Opinion),
		approvalWeights: make(map[Color]uint64),
		colorConfirmed:  false,
	}
}

func (o *OpinionManager) ApprovalWeights() map[Color]uint64 {
	return o.approvalWeights
}

func (o *OpinionManager) Events() *OpinionManagerEvents {
	return o.events
}

func (o *OpinionManager) Tangle() *Tangle {
	return o.tangle
}

func (o *OpinionManager) Setup() {
	o.tangle.Booker.Events.MessageBooked.Attach(events.NewClosure(o.FormOpinion))
}

// FormOpinion of the current tangle.
// The opinion is determined by the color with the most approvalWeight.
func (o *OpinionManager) FormOpinion(messageID MessageID) {
	defer o.events.OpinionFormed.Trigger(messageID)

	if updated := o.UpdateWeights(messageID); !updated {
		return
	}
	// Here we accumulate the approval weights in our local tangle.
	o.WeightsUpdated()
}

func (o *OpinionManager) UpdateWeights(messageID MessageID) (updated bool) {
	message := o.tangle.Storage.Message(messageID)
	messageMetadata := o.tangle.Storage.MessageMetadata(messageID)

	if messageMetadata.InheritedColor() == UndefinedColor {
		return
	}

	lastOpinion, exist := o.peerOpinions[message.Issuer]
	if !exist {
		lastOpinion = &Opinion{
			PeerID: message.Issuer,
		}
		o.peerOpinions[message.Issuer] = lastOpinion
	}

	if message.SequenceNumber <= lastOpinion.SequenceNumber {
		return
	}
	lastOpinion.SequenceNumber = message.SequenceNumber

	if lastOpinion.Color == messageMetadata.InheritedColor() {
		return
	}

	if exist {
		// We calculate the approval weight of the branch based on the node who issued the message to the branch (i.e., it already voted for the branch).
		o.approvalWeights[lastOpinion.Color] -= o.tangle.WeightDistribution.Weight(message.Issuer)
		o.events.ApprovalWeightUpdated.Trigger(lastOpinion.Color, int64(-o.tangle.WeightDistribution.Weight(message.Issuer)))

		// Record the min confirmed weight
		// When the weight of the color < confirmation threshold, but the color is still not unconfirmed yet.
		if o.colorConfirmed && o.ownOpinion == lastOpinion.Color && !o.checkColorConfirmed(o.ownOpinion) {
			o.events.MinConfirmedWeightUpdated.Trigger(lastOpinion.Color, int64(o.approvalWeights[lastOpinion.Color]))
		}
	}

	// We calculate the approval weight of the branch based on the node who issued the message to the branch (i.e., it already voted for the branch).
	o.approvalWeights[messageMetadata.InheritedColor()] += o.tangle.WeightDistribution.Weight(message.Issuer)
	o.events.ApprovalWeightUpdated.Trigger(messageMetadata.InheritedColor(), int64(o.tangle.WeightDistribution.Weight(message.Issuer)))

	lastOpinion.Color = messageMetadata.InheritedColor()
	updated = true

	if int64(message.Issuer) >= int64(config.NodesCount)-int64(3) {
		if o.tangle.Peer.ID == 0 {
			log.Debugf("Peer %d processed god request msg id %d", o.tangle.Peer.ID, message.ID)
		}
		o.events.GodMessageProcessed.Trigger(o.tangle.Peer.ID, message.ID)
	}
	return
}

func (o *OpinionManager) Opinion() Color {
	return o.ownOpinion
}

func (o *OpinionManager) SetOpinion(opinion Color) {
	if oldOpinion := o.ownOpinion; oldOpinion != opinion {
		o.events.OpinionChanged.Trigger(oldOpinion, opinion, int64(o.Tangle().WeightDistribution.Weight(o.Tangle().Peer.ID)), o.tangle.Peer.ID)
	}
	o.ownOpinion = opinion
}

func (o *OpinionManager) UpdateConfirmation(oldOpinion Color, maxOpinion Color) {
	if o.colorConfirmed && maxOpinion != oldOpinion {
		o.colorConfirmed = false
		o.Events().ColorUnconfirmed.Trigger(oldOpinion, int64(o.approvalWeights[o.ownOpinion]), int64(o.tangle.WeightDistribution.Weight(o.tangle.Peer.ID)))
	}

	if o.checkColorConfirmed(maxOpinion) && !o.colorConfirmed {
		// Here we accumulate the approval weights in our local tangle.
		o.Events().ColorConfirmed.Trigger(maxOpinion, int64(o.tangle.WeightDistribution.Weight(o.tangle.Peer.ID)))
		o.colorConfirmed = true
	}
}

// WeightsUpdated updates the opinions counter and ownOpinion based on the highest peer color value and maxApprovalWeight
// Each Color has approvalWeight. The Color with maxApprovalWeight determines the ownOpinion
func (o *OpinionManager) WeightsUpdated() {
	maxOpinion := GetMaxOpinion(o.approvalWeights)
	oldOpinion := o.ownOpinion
	if maxOpinion != oldOpinion {
		o.ownOpinion = maxOpinion
		o.Events().OpinionChanged.Trigger(oldOpinion, maxOpinion, int64(o.tangle.WeightDistribution.Weight(o.tangle.Peer.ID)))
	}
	o.UpdateConfirmation(oldOpinion, maxOpinion)
}

func (o *OpinionManager) checkColorConfirmed(newOpinion Color) bool {
	if config.WeightThresholdAbsolute {
		return float64(o.approvalWeights[newOpinion]) > float64(config.NodesTotalWeight)*config.WeightThreshold
	} else {
		aw := make(map[Color]uint64)
		for key, value := range o.approvalWeights {
			if key != newOpinion {
				aw[key] = value
			}
		}
		alternativeOpinion := GetMaxOpinion(aw)
		return float64(o.approvalWeights[newOpinion])-float64(o.approvalWeights[alternativeOpinion]) > float64(config.NodesTotalWeight)*config.WeightThreshold
	}
}

func GetMaxOpinion(aw map[Color]uint64) Color {
	maxApprovalWeight := uint64(0)
	maxOpinion := UndefinedColor
	for color, approvalWeight := range aw {
		if approvalWeight > maxApprovalWeight || approvalWeight == maxApprovalWeight && color < maxOpinion || maxOpinion == UndefinedColor {
			maxApprovalWeight = approvalWeight
			maxOpinion = color
		}
	}
	return maxOpinion
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region Opinion //////////////////////////////////////////////////////////////////////////////////////////////////////

type Opinion struct {
	PeerID         network.PeerID
	Color          Color
	SequenceNumber uint64
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region OpinionManagerEvents /////////////////////////////////////////////////////////////////////////////////////////

type OpinionManagerEvents struct {
	OpinionFormed             *events.Event
	OpinionChanged            *events.Event
	ApprovalWeightUpdated     *events.Event
	MinConfirmedWeightUpdated *events.Event
	ColorConfirmed            *events.Event
	ColorUnconfirmed          *events.Event
	GodMessageProcessed       *events.Event
}

func opinionChangedEventHandler(handler interface{}, params ...interface{}) {
	handler.(func(Color, Color, int64))(params[0].(Color), params[1].(Color), params[2].(int64))
}
func colorEventHandler(handler interface{}, params ...interface{}) {
	handler.(func(Color, int64))(params[0].(Color), params[1].(int64))
}
func reorgEventHandler(handler interface{}, params ...interface{}) {
	handler.(func(Color, int64, int64))(params[0].(Color), params[1].(int64), params[2].(int64))
}
func approvalWeightUpdatedHandler(handler interface{}, params ...interface{}) {
	handler.(func(Color, int64))(params[0].(Color), params[1].(int64))
}
func GodMessageProcessedHandler(handler interface{}, params ...interface{}) {
	handler.(func(peerID network.PeerID, id MessageID))(params[0].(network.PeerID), params[1].(MessageID))
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
