package network

import (
	"github.com/iotaledger/hive.go/crypto"
	"github.com/iotaledger/hive.go/datastructure/set"
	"github.com/iotaledger/multivers-simulation/config"
	"time"
)

// region AdversaryType ////////////////////////////////////////////////////////////////////////////////////////////////

type AdversaryType int

const (
	HonestNode AdversaryType = iota
	ShiftOpinion
	TheSameOpinion
)

func ToAdversaryType(adv int) AdversaryType {
	switch adv {
	case int(ShiftOpinion):
		return ShiftOpinion
	case int(TheSameOpinion):
		return TheSameOpinion
	default:
		return HonestNode
	}
}

func AdversaryTypeToString(adv AdversaryType) string {
	switch adv {
	case HonestNode:
		return "Honest"
	case ShiftOpinion:
		return "ShiftingOpinion"
	case TheSameOpinion:
		return "TheSameOpinion"
	}
	return ""
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region AdversaryGroup ////////////////////////////////////////////////////////////////////////////////////////////////

var AdversaryNodeIDToGroupIDMap = make(map[int]int)

func IsAdversary(nodeID int) bool {
	_, ok := AdversaryNodeIDToGroupIDMap[nodeID]
	return ok
}

type AdversaryGroup struct {
	NodeIDs       []int
	GroupMana     float64
	TargetMana    float64
	Delay         time.Duration
	AdversaryType AdversaryType
	InitColor     string
	NodeCount     int
}

func (g *AdversaryGroup) AddNodeID(id, groupId int) {
	g.NodeIDs = append(g.NodeIDs, id)
	AdversaryNodeIDToGroupIDMap[id] = groupId
}

type AdversaryGroups []*AdversaryGroup

func NewAdversaryGroups() (groups AdversaryGroups) {
	groups = make(AdversaryGroups, 0, len(config.AdversaryTypes))
	for i, configAdvType := range config.AdversaryTypes {
		// -1 indicates that there is no target mana provided, adversary will be selected randomly
		targetMana := float64(-1)
		delay := config.MinDelay
		color := ""
		nCount := 1

		if len(config.AdversaryMana) > 0 {
			targetMana = config.AdversaryMana[i]
		}

		if len(config.AdversaryDelays) > 0 {
			delay = config.AdversaryDelays[i]
		}

		if len(config.AdversaryNodeCounts) > 0 {
			nCount = config.AdversaryNodeCounts[i]
		}

		color = config.AdversaryInitColors[i]
		group := &AdversaryGroup{
			NodeIDs:       make([]int, 0, nCount),
			TargetMana:    targetMana,
			Delay:         time.Millisecond * time.Duration(delay),
			AdversaryType: ToAdversaryType(configAdvType),
			InitColor:     color,
			NodeCount:     nCount,
		}
		log.Infof("GroupID %d group %v\n", i, group)
		groups = append(groups, group)
	}

	return
}

// CalculateWeightTotalConfig returns how many nodes will be used for weight distribution and their total weight
// after excluding all adversary nodes that will not be selected randomly
func (g *AdversaryGroups) CalculateWeightTotalConfig() (int, float64) {
	totalAdvNotRandom := 0
	totalAdvManaPercentage := float64(0)

	for _, group := range *g {
		if group.TargetMana != -1 {
			totalAdvNotRandom += group.NodeCount
			totalAdvManaPercentage += group.TargetMana
		}
	}
	log.Info("config.NodesCount ", config.NodesCount, "totalAdvNotRandom ", totalAdvNotRandom)
	totalCount := config.NodesCount - totalAdvNotRandom
	totalWeight := float64(config.NodesTotalWeight) * (1 - totalAdvManaPercentage)
	return totalCount, totalWeight
}

// UpdateAdversaryNodes assigns adversary nodes in AdversaryGroups to correct nodeIDs and updates their mana
func (g *AdversaryGroups) UpdateAdversaryNodes(weightDistribution []uint64) []uint64 {
	// weight distribution with adversary weights appended at the ned
	newWeights := make([]uint64, len(weightDistribution))
	copy(newWeights, weightDistribution)
	totalRandom, randomManaGroups := g.updateNotRandomMana(weightDistribution)

	// Adversary nodes are taking indexes from the end, excluded randomly chosen nodes
	advIndex := len(weightDistribution)

	g.updateNotRandomAdvIDs(advIndex, newWeights)

	g.updateRandomManaAdversary(weightDistribution, totalRandom, randomManaGroups)

	return newWeights
}

func (g *AdversaryGroups) updateRandomManaAdversary(weightDistribution []uint64, totalRandom int, randomManaGroups []int) {
	randomIndexes := randomWeightIndex(weightDistribution, totalRandom)
	currentNodeId := 0
	for _, groupID := range randomManaGroups {
		groupMana := uint64(0)
		for i := 0; i < (*g)[groupID].NodeCount; i++ {
			weightIndex := randomIndexes[currentNodeId]
			(*g)[groupID].AddNodeID(weightIndex, groupID)
			groupMana += weightDistribution[weightIndex]
			currentNodeId++
		}
		(*g)[groupID].GroupMana = float64(groupMana)
	}
}

func (g *AdversaryGroups) updateNotRandomAdvIDs(advIndex int, newWeights []uint64) {
	for groupIndex, group := range *g {
		for i := 0; i < group.NodeCount; i++ {
			group.AddNodeID(advIndex, groupIndex)
			advIndex++
			// append adversary weight at the end of weight distribution
			nodeWeight := uint64(group.TargetMana / float64(group.NodeCount))
			newWeights = append(newWeights, nodeWeight)
		}
	}
}

func (g *AdversaryGroups) updateNotRandomMana(weightDistribution []uint64) (int, []int) {
	totalRandom := 0
	var randomManaGroups []int

	maxZipf, minZipf := weightDistribution[0], weightDistribution[len(weightDistribution)-1]

	for groupIndex, group := range *g {
		switch group.TargetMana {
		case 100: // node has max
			group.GroupMana = float64(maxZipf)
		case 0:
			group.GroupMana = float64(minZipf)
		case -1:
			randomManaGroups = append(randomManaGroups, groupIndex)
			totalRandom += group.NodeCount
		default:
			group.GroupMana = group.TargetMana
		}
	}
	return totalRandom, randomManaGroups
}

func (g *AdversaryGroups) ApplyNetworkDelayForAdversaryNodes(network *Network) {
	for _, adversaryGroup := range *g {
		for _, nodeID := range adversaryGroup.NodeIDs {
			peer := network.Peer(nodeID)
			for _, neighbor := range peer.Neighbors {
				neighbor.SetDelay(adversaryGroup.Delay)
			}
		}
	}
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

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
