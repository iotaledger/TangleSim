package network

// region AccessWeightDistribution //////////////////////////////////////////////////////////////////////////////////

type AccessWeightDistribution struct {
	weights       map[PeerID]uint64
	totalWeight   uint64
	largestWeight uint64
}

func NewAccessWeightDistribution() *AccessWeightDistribution {
	return &AccessWeightDistribution{
		weights: make(map[PeerID]uint64),
	}
}

func (c *AccessWeightDistribution) SetWeight(peerID PeerID, weight uint64) {
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

func (c *AccessWeightDistribution) Weight(peerID PeerID) uint64 {
	return c.weights[peerID]
}

func (c *AccessWeightDistribution) TotalWeight() uint64 {
	return c.totalWeight
}

func (c *AccessWeightDistribution) LargestWeight() uint64 {
	return c.largestWeight
}

func (c *AccessWeightDistribution) rescanForLargestWeight() {
	c.largestWeight = 0
	for _, weight := range c.weights {
		if weight > c.largestWeight {
			c.largestWeight = weight
		}
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
