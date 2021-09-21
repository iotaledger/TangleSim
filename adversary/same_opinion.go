package adversary

import (
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/multiverse"
)

// region ShiftingOpinionNode ///////////////////////////////////////////////////////////////////////////////////////////////////

type SameOpinionNode struct {
	*multiverse.Node
}

func NewSameOpinionNode() interface{} {
	node := multiverse.NewNode().(*multiverse.Node)
	shiftingNode := &SameOpinionNode{
		node,
	}
	shiftingNode.setupOpinionManager()
	return shiftingNode

}

func (s *SameOpinionNode) setupOpinionManager() {
	om := s.Tangle().OpinionManager
	s.Tangle().OpinionManager = NewShiftingOpinionManager(om)
	s.Tangle().OpinionManager.Setup()
}

func (s *SameOpinionNode) AssignColor(color multiverse.Color) {
	s.Tangle().OpinionManager.SetOpinion(color)
}

type SameOpinionManager struct {
	*multiverse.OpinionManager
}

func NewSameOpinionManager(om multiverse.OpinionManagerInterface) *SameOpinionManager {
	return &SameOpinionManager{
		om.(*multiverse.OpinionManager),
	}
}

func (sm *SameOpinionManager) FormOpinion(messageID multiverse.MessageID) {
	defer sm.Events().OpinionFormed.Trigger(messageID)

	if updated := sm.UpdateWeights(messageID); !updated {
		return
	}

	sm.weightsUpdated()
}

func (sm *SameOpinionManager) weightsUpdated() {
	// do nothing
}

func (sm *SameOpinionManager) Setup() {
	sm.Tangle().Booker.Events.MessageBooked.Detach(events.NewClosure(sm.OpinionManager.FormOpinion))
	sm.Tangle().Booker.Events.MessageBooked.Attach(events.NewClosure(sm.FormOpinion))
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
