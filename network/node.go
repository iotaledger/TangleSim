package network

import "github.com/iotaledger/multivers-simulation/fpcs"

// region Node /////////////////////////////////////////////////////////////////////////////////////////////////////////
type Node interface {
	Setup(peer *Peer, weightDistribution *ConsensusWeightDistribution, fpcs *fpcs.FPCS)
	HandleNetworkMessage(networkMessage interface{})
}

type NodeFactory func() Node

func NodeClosure(closure func() interface{}) NodeFactory {
	return func() Node {
		return closure().(Node)
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
