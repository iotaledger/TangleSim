package multiverse

import (
	"math"
	"time"

	"github.com/iotaledger/hive.go/datastructure/walker"
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/network"
)

// region ApprovalManager ///////////////////////////////////////////////////////////////////////////////////////////////////

type ApprovalManager struct {
	tangle                      *Tangle
	consensusWeightDistribution *network.ConsensusWeightDistribution
}

func NewApprovalManager(tangle *Tangle, cwd *network.ConsensusWeightDistribution) *ApprovalManager {
	return &ApprovalManager{
		tangle:                      tangle,
		consensusWeightDistribution: cwd,
	}
}

func (a *ApprovalManager) Setup() {
	a.tangle.Storage.Events.MessageStored.Attach(events.NewClosure(a.ApproveMessages))
}

func (a *ApprovalManager) ApproveMessages(messageID MessageID) {

	issuingMessage := a.tangle.Storage.messageDB[messageID]
	byteIndex := math.Floor(float64(issuingMessage.Issuer / 8))
	mod := issuingMessage.Issuer % 8
	weight := a.consensusWeightDistribution.Weight(issuingMessage.Issuer)

	a.tangle.Utils.WalkMessagesAndMetadata(func(message *Message, messageMetadata *MessageMetadata, walker *walker.Walker) {

		weightByte := message.WeightSlice[int(byteIndex)]
		if weightByte&(1<<mod) != 0 {
			weightByte |= 1 << mod
			message.Weight += weight

			if float64(message.Weight) >= config.MessageWeightThreshold*float64(a.consensusWeightDistribution.TotalWeight()) {
				message.ConfirmationTime = time.Now()
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

// 1 (Node 3) <- 5 (Node 2) <- 6 (Node 1)
