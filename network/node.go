package network

type Node interface {
	Setup(peer *Peer, weightDistribution *ConsensusWeightDistribution)
	HandleNetworkMessage(networkMessage interface{})
}

type NodeFactory func() Node
