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
	sameOpinionNode := &CreatingOpinionNode{
		node,
	}
	sameOpinionNode.setupOpinionManager()
	return sameOpinionNode

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
	log.Info("FormOpinion")
	if updated := sm.UpdateWeights(messageID); !updated {
		return
	}

	sm.weightsUpdated()
}

func (sm *CreatingOpinionManager) weightsUpdated() {
	// Always confirm on a new color
	oldOpinion := sm.Opinion()
	newOpinion := int(oldOpinion) + 1
	newOpinionColor := multiverse.ColorFromInt(newOpinion)
	sm.SetOpinion(newOpinionColor)
	sm.UpdateConfirmation(oldOpinion, newOpinionColor)
}

func (n *CreatingOpinionNode) IssuePayload(payload multiverse.Color) {
	// Always issue a new colored payload
	oldOpinion := n.Tangle().OpinionManager.Opinion()
	newOpinion := int(oldOpinion) + 1
	newOpinionColor := multiverse.ColorFromInt(newOpinion)
	n.Tangle().OpinionManager.SetOpinion(newOpinionColor)
	hackedPayload := newOpinionColor
	n.Peer().Socket <- hackedPayload
}

func (sm *CreatingOpinionManager) Setup() {
	sm.Tangle().Booker.Events.MessageBooked.Detach(events.NewClosure(sm.OpinionManager.FormOpinion))
	sm.Tangle().Booker.Events.MessageBooked.Attach(events.NewClosure(sm.FormOpinion))
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
