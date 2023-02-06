package multiverse

import (
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/logger"
	"github.com/iotaledger/multivers-simulation/network"
)

var log = logger.New("Multiverse")

// region Node /////////////////////////////////////////////////////////////////////////////////////////////////////////

type NodeInterface interface {
	Peer() *network.Peer
	Tangle() *Tangle
	IssuePayload(payload Color)
}

type Node struct {
	peer   *network.Peer
	tangle *Tangle
}

func NewNode() interface{} {
	return &Node{
		tangle: NewTangle(),
	}
}

func (n *Node) Peer() *network.Peer {
	return n.peer
}

func (n *Node) Tangle() *Tangle {
	return n.tangle
}

func (n *Node) Setup(peer *network.Peer, weightDistribution *network.ConsensusWeightDistribution) {
	defer log.Debugf("%s: Setting up Multiverse ... [DONE]", peer)

	n.peer = peer
	n.tangle.Setup(peer, weightDistribution)
	n.tangle.Requester.Events.Request.Attach(events.NewClosure(func(messageID MessageID) {
		n.peer.GossipNetworkMessage(&MessageRequest{MessageID: messageID, Issuer: n.peer.ID})
	}))
	n.tangle.Booker.Events.MessageBooked.Attach(events.NewClosure(func(messageID MessageID) {
		// Push the message to the scheduling buffer
		n.tangle.Scheduler.EnqueueMessage(*n.tangle.Storage.Message(messageID))
	}))
}

// IssuePayload sends the Color to the socket for creating a new Message
func (n *Node) IssuePayload(payload Color) {
	n.peer.Socket <- payload
}

func (n *Node) HandleNetworkMessage(networkMessage interface{}) {
	switch receivedNetworkMessage := networkMessage.(type) {
	case *MessageRequest:
		if requestedMessage := n.tangle.Storage.Message(receivedNetworkMessage.MessageID); requestedMessage != nil {
			n.peer.Neighbors[receivedNetworkMessage.Issuer].Send(requestedMessage)
		}
	case *Message:
		n.tangle.ProcessMessage(receivedNetworkMessage)
	case Color:
		n.tangle.ProcessMessage(n.tangle.MessageFactory.CreateMessage(receivedNetworkMessage))
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
