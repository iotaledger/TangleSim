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
	WeightsUpdated()
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
			OpinionFormed:         events.NewEvent(messageIDEventCaller),
			OpinionChanged:        events.NewEvent(opinionChangedEventHandler),
			ApprovalWeightUpdated: events.NewEvent(approvalWeightUpdatedHandler),
			ColorConfirmed:        events.NewEvent(colorEventHandler),
			ColorUnconfirmed:      events.NewEvent(colorEventHandler),
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

func (o *OpinionManager) Setup() {
	o.tangle.Booker.Events.MessageBooked.Attach(events.NewClosure(o.FormOpinion))
}

// Form the opinion of the current tangle.
// The opinion is determined by the color with the most approvalWeight.
func (o *OpinionManager) FormOpinion(messageID MessageID) {
	defer o.events.OpinionFormed.Trigger(messageID)

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
		if o.colorConfirmed && float64(o.approvalWeights[lastOpinion.Color]) > float64(config.NodesTotalWeight)*config.MessageWeightThreshold {
			o.colorConfirmed = false
			o.events.ColorUnconfirmed.Trigger(lastOpinion.Color)
		}

		o.approvalWeights[lastOpinion.Color] -= o.tangle.WeightDistribution.Weight(message.Issuer)
		o.events.ApprovalWeightUpdated.Trigger(lastOpinion.Color, int64(-o.tangle.WeightDistribution.Weight(message.Issuer)))
	}
	lastOpinion.Color = messageMetadata.InheritedColor()

	o.approvalWeights[messageMetadata.InheritedColor()] += o.tangle.WeightDistribution.Weight(message.Issuer)
	o.events.ApprovalWeightUpdated.Trigger(messageMetadata.InheritedColor(), int64(o.tangle.WeightDistribution.Weight(message.Issuer)))

	if !o.colorConfirmed && float64(o.approvalWeights[lastOpinion.Color]) > float64(config.NodesTotalWeight)*config.MessageWeightThreshold {
		o.colorConfirmed = true
		o.events.ColorConfirmed.Trigger(lastOpinion.Color)
	}
	o.WeightsUpdated()
}

func (o *OpinionManager) Opinion() Color {
	return o.ownOpinion
}

func (o *OpinionManager) SetOpinion(opinion Color) {
	o.ownOpinion = opinion
}

// Update the opinions counter and ownOpinion based on the highest peer color value and maxApprovalWeight
// Each Color has approvalWeight. The Color with maxApprovalWeight determines the ownOpinion
func (o *OpinionManager) WeightsUpdated() {
	maxApprovalWeight := uint64(0)
	maxOpinion := UndefinedColor
	for color, approvalWeight := range o.approvalWeights {
		if approvalWeight > maxApprovalWeight || approvalWeight == maxApprovalWeight && color < maxOpinion {
			maxApprovalWeight = approvalWeight
			maxOpinion = color
		}
	}

	if oldOpinion := o.ownOpinion; maxOpinion != oldOpinion {
		o.ownOpinion = maxOpinion
		o.events.OpinionChanged.Trigger(oldOpinion, maxOpinion)
	}
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
	OpinionFormed         *events.Event
	OpinionChanged        *events.Event
	ApprovalWeightUpdated *events.Event
	ColorConfirmed        *events.Event
	ColorUnconfirmed      *events.Event
}

func opinionChangedEventHandler(handler interface{}, params ...interface{}) {
	handler.(func(Color, Color))(params[0].(Color), params[1].(Color))
}
func colorEventHandler(handler interface{}, params ...interface{}) {
	handler.(func(Color))(params[0].(Color))
}

func approvalWeightUpdatedHandler(handler interface{}, params ...interface{}) {
	handler.(func(Color, int64))(params[0].(Color), params[1].(int64))
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
