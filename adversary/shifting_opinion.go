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

func (s *ShiftingOpinionNode) AssignColor(color multiverse.Color) {
	s.Tangle().OpinionManager.SetOpinion(color)
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
	aw := make(map[multiverse.Color]uint64)
	for key, value := range sm.ApprovalWeights() {
		aw[key] = value
	}
	// more than one color present
	if len(aw) > 1 {
		maxOpinion := sm.getMaxOpinion(aw)
		delete(aw, maxOpinion)
	}

	newOpinion := sm.getMaxOpinion(aw)
	oldOpinion := sm.Opinion()
	if newOpinion != oldOpinion {
		sm.SetOpinion(newOpinion)
	}
	sm.UpdateConfirmation(oldOpinion, newOpinion)
}

func (sm *ShiftingOpinionManager) getMaxOpinion(aw map[multiverse.Color]uint64) multiverse.Color {
	maxApprovalWeight := uint64(0)
	maxOpinion := multiverse.UndefinedColor
	for color, approvalWeight := range aw {
		if approvalWeight > maxApprovalWeight || approvalWeight == maxApprovalWeight && color < maxOpinion || maxOpinion == multiverse.UndefinedColor {
			maxApprovalWeight = approvalWeight
			maxOpinion = color
		}
	}
	return maxOpinion
}

func (sm *ShiftingOpinionManager) Setup() {
	sm.Tangle().Booker.Events.MessageBooked.Detach(events.NewClosure(sm.OpinionManager.FormOpinion))
	sm.Tangle().Booker.Events.MessageBooked.Attach(events.NewClosure(sm.FormOpinion))
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
