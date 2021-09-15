package network

import (
	"github.com/iotaledger/hive.go/crypto"
	"github.com/iotaledger/hive.go/datastructure/set"
	"github.com/iotaledger/multivers-simulation/config"
	"math"
)

type WeightGenerator func(nodeCount int, adversaryGroups AdversaryGroups) []uint64

// region ZIPFDistribution /////////////////////////////////////////////////////////////////////////////////////////////

func ZIPFDistribution(s float64, totalWeight float64) WeightGenerator {
	return func(nodeCount int, groups AdversaryGroups) (result []uint64) {
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

		// TODO connect to choose index
		return
	}
}

func chooseAdversaryNodes(ZIPFDistribution []uint64, groups AdversaryGroups, totalWeight float64, nodeCount int) {
	// are there any adversary groups provided in the configuration?
	if len(groups) == 0 {
		return
	}
	manaPercentageProvided := make(AdversaryGroups, 0)
	// check if adversary mana target provided
	for _, group := range groups {
		switch group.TargetMana {
		case 100:
			idx := selectMaxWeightIndex(ZIPFDistribution)
			group.NodeIDs = append(group.NodeIDs, idx)
		case 0:
			idx := selectMinWeightIndex(ZIPFDistribution)
			group.NodeIDs = append(group.NodeIDs, idx)
		case -1:
			// all weights are chosen randomly
			for i, idx := range randomWeightIndex(ZIPFDistribution, len(groups)) {
				groups[i].NodeIDs = append(groups[i].NodeIDs, idx)
			}
			return
		default:
			group.TargetMana = totalWeight * group.TargetMana / 100
			manaPercentageProvided = append(manaPercentageProvided, group)
		}
	}

	// We define this because the adversary nodes' weights might not exactly occupy totalWeight * q
	errorThreshold := totalWeight * config.AdversaryErrorThreshold

	for i, v := range ZIPFDistribution[1:] {
		for _, group := range manaPercentageProvided {
			if group.GroupMana < group.TargetMana && group.GroupMana+float64(v) < group.TargetMana+errorThreshold {
				group.GroupMana += float64(v)
				group.NodeIDs = append(group.NodeIDs, i+1)
			}
			log.Infof("Group: %d, Node count: %d, q: %.5f, adNodeList: %v", i, nodeCount,
				group.GroupMana/totalWeight, group.NodeIDs)
		}
	}

	// select randomly

}

func randomWeightIndex(weights []uint64, count int) (randomWeights []int) {
	selectedPeers := set.New()
	for len(randomWeights) < count {
		if randomIndex := crypto.Randomness.Intn(len(weights)); selectedPeers.Add(randomIndex) {
			randomWeights = append(randomWeights, randomIndex)
		}
	}
	return
}

func selectMinWeightIndex(x []uint64) int {
	minLocation := 0
	currentMin := x[0]
	for i, v := range x[1:] {
		if v < currentMin {
			currentMin = v
			minLocation = i + 1
		}
	}
	return minLocation
}

func selectMaxWeightIndex(x []uint64) int {
	maxLocation := 0
	currentMax := x[0]
	for i, v := range x[1:] {
		if v > currentMax {
			currentMax = v
			maxLocation = i + 1
		}
	}
	return maxLocation
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

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
