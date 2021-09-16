package network

import (
	"github.com/iotaledger/hive.go/crypto"
	"github.com/iotaledger/hive.go/datastructure/queue"
	"github.com/iotaledger/hive.go/datastructure/set"
	"github.com/iotaledger/hive.go/types"
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

func ToType(adv int) AdversaryType {
	switch adv {
	case int(ShiftOpinion):
		return ShiftOpinion
	case int(TheSameOpinion):
		return TheSameOpinion
	default:
		return HonestNode
	}
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

		if len(config.AdversaryMana) > 0 {
			targetMana = config.AdversaryMana[i]
		}

		if len(config.AdversaryDelays) > 0 {
			delay = config.AdversaryDelays[i]
		}

		groups = append(groups, &AdversaryGroup{
			NodeIDs:       make([]int, 0),
			TargetMana:    targetMana,
			Delay:         time.Millisecond * time.Duration(delay),
			AdversaryType: ToType(configAdvType),
		})
	}
	return
}

func (g *AdversaryGroups) ChooseAdversaryNodes(ZIPFDistribution []uint64, totalWeight float64) {
	// are there any adversary groups provided in the configuration?
	if len(*g) == 0 {
		return
	}
	manaPercentageGroupIds := make([]int, 0)
	usedWeights := make(map[int]types.Empty)
	// check if adversary mana target is provided
	for groupIndex, group := range *g {
		switch group.TargetMana {
		case 100: // node has max
			idx := selectMaxWeightIndex(ZIPFDistribution, usedWeights)
			group.AddNodeID(idx, groupIndex)
			group.GroupMana = float64(ZIPFDistribution[idx])
			usedWeights[idx] = types.Void
		case 0:
			idx := selectMinWeightIndex(ZIPFDistribution, usedWeights)
			group.AddNodeID(idx, groupIndex)
			group.GroupMana = float64(ZIPFDistribution[idx])
			usedWeights[idx] = types.Void
		case -1:
			// all weights are chosen randomly
			for i, idx := range randomWeightIndex(ZIPFDistribution, len(*g)) {
				(*g)[i].GroupMana = float64(ZIPFDistribution[idx])
				(*g)[i].AddNodeID(idx, i)
			}
			return
		default:
			group.TargetMana = totalWeight * group.TargetMana / 100
			manaPercentageGroupIds = append(manaPercentageGroupIds, groupIndex)
		}
	}

	// We define this because the adversary nodes' weights might not exactly occupy totalWeight * q
	errorThreshold := totalWeight * config.AdversaryErrorThreshold
	groupIndexQueue := queue.New(len(manaPercentageGroupIds))
	// to choose indexes alternately and keep the balance between adversary groups
	for _, groupIndex := range manaPercentageGroupIds {
		groupIndexQueue.Offer(groupIndex)
	}
	for index, v := range ZIPFDistribution[1:] {
		for i := 0; i < groupIndexQueue.Size(); i++ {
			elem, _ := groupIndexQueue.Poll()
			groupIndex := elem.(int)
			if (*g)[groupIndex].GroupMana < (*g)[groupIndex].TargetMana && (*g)[groupIndex].GroupMana+float64(v) < (*g)[groupIndex].TargetMana+errorThreshold {
				(*g)[groupIndex].GroupMana += float64(v)
				(*g)[groupIndex].AddNodeID(index+1, groupIndex)
				groupIndexQueue.Offer(groupIndex)
				break
			}
			groupIndexQueue.Offer(groupIndex)
		}
	}
	for groupIndex, group := range *g {
		log.Infof("GroupId: %d, AdversaryType: %d, Node count: %d, q: %.5f, adNodeList: %v", groupIndex, group.AdversaryType, len(group.NodeIDs),
			group.GroupMana/totalWeight, group.NodeIDs)
	}
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

func selectMinWeightIndex(weights []uint64, usedWeights map[int]types.Empty) int {
	minLocation := -1
	currentMin := weights[0]
	for i, weight := range weights {
		// skip indexes that has been already used
		if _, ok := usedWeights[i]; !ok {
			if weight < currentMin || minLocation == -1 {
				currentMin = weight
				minLocation = i
			}
		}
	}
	return minLocation
}

func selectMaxWeightIndex(weights []uint64, usedWeights map[int]types.Empty) int {
	maxLocation := -1
	currentMax := weights[0]
	for i, weight := range weights {
		// skip indexes that has been already used
		if _, ok := usedWeights[i]; !ok {
			if weight > currentMax || maxLocation == -1 {
				currentMax = weight
				maxLocation = i
			}
		}
	}
	return maxLocation
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
