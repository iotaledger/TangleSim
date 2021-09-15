package adversary

import (
	"fmt"
	"github.com/iotaledger/multivers-simulation/multiverse"
)

// region HonestNode ///////////////////////////////////////////////////////////////////////////////////////////////////

type HonestNode struct {
	*multiverse.Node
}

func (h *HonestNode) SetupOpinionManager() {
	// no change in behavior of honest node's opinion manager
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region ShiftingOpinionNode ///////////////////////////////////////////////////////////////////////////////////////////////////

type ShiftingOpinionNode struct {
	*multiverse.Node
}

func NewShiftingOpinionNode() interface{} {
	node := multiverse.NewNode().(*multiverse.Node)
	return &ShiftingOpinionNode{
		node,
	}

}

func (s *ShiftingOpinionNode) SetupOpinionManager() {
	om := s.Tangle().OpinionManager
	s.Tangle().OpinionManager = NewShiftingOpinionManager(om)
}

type ShiftingOpinionManager struct {
	*multiverse.OpinionManager
}

func NewShiftingOpinionManager(om multiverse.OpinionManagerInterface) *ShiftingOpinionManager {
	return &ShiftingOpinionManager{
		om.(*multiverse.OpinionManager),
	}
}

func (sm *ShiftingOpinionManager) WeightsUpdated() {
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
	fmt.Println("I'm shadowed opinion manager")
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
