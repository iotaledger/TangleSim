package multiverse

import (
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/logger"
	"github.com/iotaledger/multivers-simulation/network"
)

var log = logger.New("Multiverse")

// region Node /////////////////////////////////////////////////////////////////////////////////////////////////////////

type Node struct {
	Peer   *network.Peer
	Tangle *Tangle
}

func NewNode() network.Node {
	return &Node{
		Tangle: NewTangle(),
	}
}

func (n *Node) Setup(peer *network.Peer, weightDistribution *network.ConsensusWeightDistribution) {
	defer log.Debugf("%s: Setting up Multiverse ... [DONE]", peer)

	n.Peer = peer
	n.Tangle.Setup(peer, weightDistribution)
	n.Tangle.Requester.Events.Request.Attach(events.NewClosure(func(messageID MessageID) {
		n.Peer.GossipNetworkMessage(&MessageRequest{MessageID: messageID, Issuer: n.Peer.ID})
	}))
	n.Tangle.Booker.Events.MessageBooked.Attach(events.NewClosure(func(messageID MessageID) {
		n.Peer.GossipNetworkMessage(n.Tangle.Storage.Message(messageID))
	}))
}

func (n *Node) IssuePayload(payload Color) {
	n.Peer.Socket <- payload
}

func (n *Node) HandleNetworkMessage(networkMessage interface{}) {
	switch receivedNetworkMessage := networkMessage.(type) {
	case *MessageRequest:
		if requestedMessage := n.Tangle.Storage.Message(receivedNetworkMessage.MessageID); requestedMessage != nil {
			n.Peer.Neighbors[receivedNetworkMessage.Issuer].Send(requestedMessage)
		}
	case *Message:
		n.Tangle.ProcessMessage(receivedNetworkMessage)
	case Color:
		n.Tangle.ProcessMessage(n.Tangle.MessageFactory.CreateMessage(receivedNetworkMessage))
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
