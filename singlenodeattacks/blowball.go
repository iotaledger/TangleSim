package singlenodeattacks

import (
	"time"

	"github.com/iotaledger/hive.go/types"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/multiverse"
)

type BlowballNode struct {
	*multiverse.Node

	nearTSCSet *multiverse.TipSet
}

func NewBlowballNode() interface{} {
	node := multiverse.NewNode().(*multiverse.Node)
	blowBallNode := &BlowballNode{
		Node:       node,
		nearTSCSet: multiverse.NewTipSet(nil),
	}
	return blowBallNode
}

func (n *BlowballNode) IssuePayload(payload multiverse.Color) {
	// create a blow ball
	tm := n.Tangle().TipManager
	tipSet := tm.TipSet(multiverse.UndefinedColor)
	oldestMessageID := tm.WalkForOldestUnconfirmed(tipSet)
	oldestMessage := n.CreateMessage(oldestMessageID, payload)
	// gossip and process oldest message
	n.Tangle().ProcessMessage(oldestMessage)
	n.Peer().GossipNetworkMessage(oldestMessage)

	// create and issue blowball
	blowBall := n.CreateBlowBall(oldestMessage, payload)
	for _, message := range blowBall {
		n.Tangle().ProcessMessage(message)
		n.Peer().GossipNetworkMessage(message)
	}
}

func (n *BlowballNode) CreateBlowBall(centerMessage *multiverse.Message, payload multiverse.Color) map[multiverse.MessageID]*multiverse.Message {
	blowBallMessages := make(map[multiverse.MessageID]*multiverse.Message)
	for i := 0; i < config.Params.BlowballSize; i++ {
		m := n.CreateMessage(centerMessage.ID, payload)
		blowBallMessages[m.ID] = m
	}
	return blowBallMessages
}

func (n *BlowballNode) CreateMessage(parent multiverse.MessageID, payload multiverse.Color) *multiverse.Message {
	// create a new message
	strongParents := multiverse.MessageIDs{parent: types.Void}
	weakParents := multiverse.MessageIDs{}
	m := &multiverse.Message{
		ID:             multiverse.NewMessageID(),
		StrongParents:  strongParents,
		WeakParents:    weakParents,
		SequenceNumber: n.Tangle().MessageFactory.SequenceNumber(),
		Issuer:         n.Tangle().Peer.ID,
		Payload:        payload,
		IssuanceTime:   time.Now(),
	}
	return m
}

func (n *BlowballNode) AssignColor(color multiverse.Color) {
	// no need to assign the color to the node for now
}
