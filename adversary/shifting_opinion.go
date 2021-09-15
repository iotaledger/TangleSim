package adversary

import (
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/multiverse"
)

// region ShiftingOpinionNode ///////////////////////////////////////////////////////////////////////////////////////////////////

type ShiftingOpinionNode struct {
	*multiverse.Node
}

func NewShiftingOpinionNode() interface{} {
	node := multiverse.NewNode().(*multiverse.Node)
	shiftingNode := &ShiftingOpinionNode{
		node,
	}
	shiftingNode.setupOpinionManager()
	return shiftingNode

}

func (s *ShiftingOpinionNode) setupOpinionManager() {
	om := s.Tangle().OpinionManager
	s.Tangle().OpinionManager = NewShiftingOpinionManager(om)
	s.Tangle().OpinionManager.Setup()
}

type ShiftingOpinionManager struct {
	*multiverse.OpinionManager
}

func NewShiftingOpinionManager(om multiverse.OpinionManagerInterface) *ShiftingOpinionManager {
	return &ShiftingOpinionManager{
		om.(*multiverse.OpinionManager),
	}
}

func (sm *ShiftingOpinionManager) FormOpinion(messageID multiverse.MessageID) {
	defer sm.Events().OpinionFormed.Trigger(messageID)

	if updated := sm.UpdateWeights(messageID); !updated {
		return
	}

	sm.weightsUpdated()
}

func (sm *ShiftingOpinionManager) weightsUpdated() {
	maxApprovalWeight := uint64(0)
	maxOpinion := multiverse.UndefinedColor
	for color, approvalWeight := range sm.ApprovalWeights() {
		if approvalWeight > maxApprovalWeight || approvalWeight == maxApprovalWeight && color < maxOpinion {
			maxApprovalWeight = approvalWeight
			maxOpinion = color
		}
	}

	if oldOpinion := sm.Opinion(); maxOpinion != oldOpinion {
		sm.Events().OpinionChanged.Trigger(oldOpinion, oldOpinion)
	}
}

func (sm *ShiftingOpinionManager) Setup() {
	sm.Tangle().Booker.Events.MessageBooked.Detach(events.NewClosure(sm.OpinionManager.FormOpinion))
	sm.Tangle().Booker.Events.MessageBooked.Attach(events.NewClosure(sm.FormOpinion))
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
