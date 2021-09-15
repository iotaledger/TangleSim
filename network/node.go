package network

// region Node /////////////////////////////////////////////////////////////////////////////////////////////////////////
type Node interface {
	Setup(peer *Peer, weightDistribution *ConsensusWeightDistribution)
	HandleNetworkMessage(networkMessage interface{})
}

type NodeFactory func() Node

func NodeClosure(closure func() interface{}) NodeFactory {
	return func() Node {
		return closure().(Node)
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
