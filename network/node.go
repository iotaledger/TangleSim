package network

import "time"

// region Node /////////////////////////////////////////////////////////////////////////////////////////////////////////
type Node interface {
	Setup(peer *Peer, weightDistribution *ConsensusWeightDistribution, bandwidthDistribution *BandwidthDistribution, genesisTime time.Time)
	HandleNetworkMessage(networkMessage interface{})
}

type NodeFactory func() Node

func NodeClosure(closure func() interface{}) NodeFactory {
	return func() Node {
		return closure().(Node)
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
