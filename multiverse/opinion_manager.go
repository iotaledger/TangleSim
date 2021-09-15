package multiverse

import (
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/network"
)

// region OpinionManager ///////////////////////////////////////////////////////////////////////////////////////////////

type OpinionManager struct {
	Events *OpinionManagerEvents

	tangle          *Tangle
	ownOpinion      Color
	peerOpinions    map[network.PeerID]*Opinion
	approvalWeights map[Color]uint64
	colorConfirmed  bool
}

func NewOpinionManager(tangle *Tangle) (opinionManager *OpinionManager) {
	return &OpinionManager{
		Events: &OpinionManagerEvents{
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

func (o *OpinionManager) Setup() {
	o.tangle.Booker.Events.MessageBooked.Attach(events.NewClosure(o.FormOpinion))
}

// Form the opinion of the current tangle.
// The opinion is determined by the color with the most approvalWeight.
func (o *OpinionManager) FormOpinion(messageID MessageID) {
	defer o.Events.OpinionFormed.Trigger(messageID)

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
			// Note that here we calculate the accumulated weights in our local tangle,
			// so we accumulate the weight of node who is processing the booked message.
			// Also note that the node has not voted for this branch yet.
			o.Events.ColorUnconfirmed.Trigger(lastOpinion.Color, int64(o.tangle.WeightDistribution.Weight(o.tangle.Peer.ID)))
		}

		// We calculate the approval weight of the branch based on the node who issued the message to the branch (i.e., it already voted for the branch).
		o.approvalWeights[lastOpinion.Color] -= o.tangle.WeightDistribution.Weight(message.Issuer)
		o.Events.ApprovalWeightUpdated.Trigger(lastOpinion.Color, int64(-o.tangle.WeightDistribution.Weight(message.Issuer)))
	}
	lastOpinion.Color = messageMetadata.InheritedColor()

	// We calculate the approval weight of the branch based on the node who issued the message to the branch (i.e., it already voted for the branch).
	o.approvalWeights[messageMetadata.InheritedColor()] += o.tangle.WeightDistribution.Weight(message.Issuer)
	o.Events.ApprovalWeightUpdated.Trigger(messageMetadata.InheritedColor(), int64(o.tangle.WeightDistribution.Weight(message.Issuer)))

	if !o.colorConfirmed && float64(o.approvalWeights[lastOpinion.Color]) > float64(config.NodesTotalWeight)*config.MessageWeightThreshold {
		o.colorConfirmed = true
		// Here we accumulate the approval weights in our local tangle.
		o.Events.ColorConfirmed.Trigger(lastOpinion.Color, int64(o.tangle.WeightDistribution.Weight(o.tangle.Peer.ID)))
	}
	// Here we accumulate the approval weights in our local tangle.
	o.weightsUpdated(o.tangle.Peer.ID)
}

func (o *OpinionManager) Opinion() Color {
	return o.ownOpinion
}

// Update the opinions counter and ownOpinion based on the highest peer color value and maxApprovalWeight
// Each Color has approvalWeight. The Color with maxApprovalWeight determines the ownOpinion
func (o *OpinionManager) weightsUpdated(peerID network.PeerID) {
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
		o.Events.OpinionChanged.Trigger(oldOpinion, maxOpinion, int64(o.tangle.WeightDistribution.Weight(peerID)))
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
	handler.(func(Color, Color, int64))(params[0].(Color), params[1].(Color), params[2].(int64))
}
func colorEventHandler(handler interface{}, params ...interface{}) {
	handler.(func(Color, int64))(params[0].(Color), params[1].(int64))
}

func approvalWeightUpdatedHandler(handler interface{}, params ...interface{}) {
	handler.(func(Color, int64))(params[0].(Color), params[1].(int64))
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
