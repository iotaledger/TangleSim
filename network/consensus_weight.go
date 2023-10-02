package network

type WeightGenerator func(nodeCount int, nodeTotalWeight float64) []uint64
type EqualWeightGenerator func(validatorNodeCount int, nonValidatorNodeCount int, totalWeight int) []uint64

// region ConsensusWeightDistribution //////////////////////////////////////////////////////////////////////////////////

type ConsensusWeightDistribution struct {
	weights       map[PeerID]uint64
	totalWeight   uint64
	largestWeight uint64
}

func NewConsensusWeightDistribution() *ConsensusWeightDistribution {
	return &ConsensusWeightDistribution{
		weights: make(map[PeerID]uint64),
	}
}

func (c *ConsensusWeightDistribution) SetWeight(peerID PeerID, weight uint64) {
	if existingWeight, exists := c.weights[peerID]; exists {
		c.totalWeight -= existingWeight

		if c.largestWeight == existingWeight {
			c.rescanForLargestWeight()
		}
	}

	c.weights[peerID] = weight
	c.totalWeight += weight

	if weight > c.largestWeight {
		c.largestWeight = weight
	}
}

func (c *ConsensusWeightDistribution) Weight(peerID PeerID) uint64 {
	return c.weights[peerID]
}

func (c *ConsensusWeightDistribution) Weights() map[PeerID]uint64 {
	return c.weights
}

func (c *ConsensusWeightDistribution) TotalWeight() uint64 {
	return c.totalWeight
}

func (c *ConsensusWeightDistribution) LargestWeight() uint64 {
	return c.largestWeight
}

func (c *ConsensusWeightDistribution) rescanForLargestWeight() {
	c.largestWeight = 0
	for _, weight := range c.weights {
		if weight > c.largestWeight {
			c.largestWeight = weight
		}
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
