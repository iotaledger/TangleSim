package multiverse

import (
	"github.com/iotaledger/hive.go/datastructure/walker"
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/network"
	"math"
)

/ region ApprovalManager ///////////////////////////////////////////////////////////////////////////////////////////////////

type ApprovalManager struct {
	tangle *Tangle
	consensusWeightDistribution *network.ConsensusWeightDistribution
	//Events *ApprovalEvents
}

func NewApprovalManager(tangle *Tangle, cwd *network.ConsensusWeightDistribution) *ApprovalManager {
	return &ApprovalManager{
		tangle:                      tangle,
		consensusWeightDistribution: cwd,
	}
}

func (a *ApprovalManager) Setup() {
	s.tangle.Storage.Events.MessageStored.Attach(events.NewClosure(a.ApproveMessages))
}

func (a *ApprovalManager) ApproveMessages(messageID MessageID) {
	a.tangle.Utils.WalkMessagesAndMetadata(func(message *Message, messageMetadata *MessageMetadata, walker *walker.Walker) {
		weight := a.consensusWeightDistribution.Weight(message.Issuer)
		byteIndex := math.Floor(float64(message.Issuer / 8))
		weightByte := message.WeightSlice[int(byteIndex)]
		mod := message.Issuer % 8
		weightByte |= 1 << mod


		for strongChildID := range s.tangle.Storage.StrongChildren(message.ID) {
			walker.Push(strongChildID)
		}
		for weakChildID := range s.tangle.Storage.WeakChildren(message.ID) {
			walker.Push(weakChildID)
		}
		//TODO ask Hans about revisit elements, why do we need it?
	}, NewMessageIDs(messageID), true)
}