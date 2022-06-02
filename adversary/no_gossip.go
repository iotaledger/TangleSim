package adversary

import (
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/multiverse"
)

// region NoGossipNode ///////////////////////////////////////////////////////////////////////////////////////////////////

type NoGossipNode struct {
	*multiverse.Node
}

func NewNoGossipNode() interface{} {
	node := multiverse.NewNode().(*multiverse.Node)
	noGossipNode := &NoGossipNode{
		node,
	}
	noGossipNode.UpdateGossipBehavior()
	return noGossipNode
}

func (n *NoGossipNode) UpdateGossipBehavior() {
	n.Tangle().Booker.Events.MessageBooked.Attach(events.NewClosure(func(messageID multiverse.MessageID) {
		// do nothing - no gossiping
	}))
	n.Tangle().Requester.Events.Request.Attach(events.NewClosure(func(messageID multiverse.MessageID) {
		// do nothing - no answering requests for missing messages
	}))
}

func (n *NoGossipNode) AssignColor(color multiverse.Color) {
	// do nothing - leave undefined color
}

func (n *NoGossipNode) IssuePayload(payload multiverse.Color) {
	// do nothing - this node will not issue DS message, to not allow other nodes count his opinion for any of colors
	// user needs to define other adv group that will issue DS
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////
