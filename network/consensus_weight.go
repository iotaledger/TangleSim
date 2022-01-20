package network

import (
	"math"
)

type WeightGenerator func(nodeCount int, nodeTotalWeight float64) []uint64

// region ZIPFDistribution /////////////////////////////////////////////////////////////////////////////////////////////

func ZIPFDistribution(s float64) WeightGenerator {
	return func(nodeCount int, totalWeight float64) (result []uint64) {
		rawTotalWeight := uint64(0)
		rawWeights := make([]uint64, nodeCount)
		for i := 0; i < nodeCount; i++ {
			weight := uint64(math.Pow(float64(i+1), -s) * totalWeight)
			rawWeights[i] = weight
			rawTotalWeight += weight
		}

		normalizedTotalWeight := uint64(0)
		result = make([]uint64, nodeCount)
		for i := 0; i < nodeCount; i++ {
			normalizedWeight := uint64((float64(rawWeights[i]) / float64(rawTotalWeight)) * totalWeight)

			result[i] = normalizedWeight
			normalizedTotalWeight += normalizedWeight
		}

		result[0] += uint64(totalWeight) - normalizedTotalWeight

		return
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region ConsensusWeightDistribution //////////////////////////////////////////////////////////////////////////////////

type ConsensusWeightDistribution struct {
	weights       map[PeerID]uint64
	totalWeight   uint64
	largestWeight uint64
}

func (c *ConsensusWeightDistribution) WeightsLength() int {
	return len(c.weights)
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
