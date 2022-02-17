package adversary

import (
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/multiverse"
)

// region CreatingOpinionNode ///////////////////////////////////////////////////////////////////////////////////////////////////

type CreatingOpinionNode struct {
	*multiverse.Node
}

func NewCreatingOpinionNode() interface{} {
	node := multiverse.NewNode().(*multiverse.Node)
	creatingOpinionNode := &CreatingOpinionNode{
		node,
	}
	creatingOpinionNode.setupOpinionManager()
	return creatingOpinionNode

}

func (s *CreatingOpinionNode) setupOpinionManager() {
	om := s.Tangle().OpinionManager
	s.Tangle().OpinionManager = NewCreatingOpinionManager(om)
	s.Tangle().OpinionManager.Setup()
}

func (s *CreatingOpinionNode) AssignColor(color multiverse.Color) {
	s.Tangle().OpinionManager.SetOpinion(color)
}

type CreatingOpinionManager struct {
	*multiverse.OpinionManager
}

func NewCreatingOpinionManager(om multiverse.OpinionManagerInterface) *CreatingOpinionManager {
	return &CreatingOpinionManager{
		om.(*multiverse.OpinionManager),
	}
}

func (sm *CreatingOpinionManager) FormOpinion(messageID multiverse.MessageID) {
	defer sm.Events().OpinionFormed.Trigger(messageID)
	if updated := sm.UpdateWeights(messageID); !updated {
		return
	}

	sm.weightsUpdated()
}

func (sm *CreatingOpinionManager) weightsUpdated() {
	// Always confirm on a new color, mod 30 to avoid out of memory
	oldOpinion := sm.Opinion()
	newOpinion := (int(oldOpinion) + 1) % 20
	newOpinionColor := multiverse.ColorFromInt(newOpinion)
	sm.SetOpinion(newOpinionColor)
	sm.UpdateConfirmation(oldOpinion, newOpinionColor)
}

func (sm *CreatingOpinionManager) Setup() {
	sm.Tangle().Booker.Events.MessageBooked.Detach(events.NewClosure(sm.OpinionManager.FormOpinion))
	sm.Tangle().Booker.Events.MessageBooked.Attach(events.NewClosure(sm.FormOpinion))
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
