package multiverse

import (
	"github.com/iotaledger/multivers-simulation/network"
)

// region Node /////////////////////////////////////////////////////////////////////////////////////////////////////////

type Node struct {
	Peer *network.Peer
}

func NewNode() network.Node {
	return &Node{}
}

func (n *Node) Setup(peer *network.Peer, weightDistribution *network.ConsensusWeightDistribution) {
	n.Peer = peer
}

func (n *Node) HandleNetworkMessage(networkMessage interface{}) {
	panic("implement me")
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
