package network

import (
	"github.com/iotaledger/hive.go/core/generics/constraints"
	"github.com/iotaledger/multivers-simulation/config"
)

type SingleAttacker struct {
	nodeID               int
	TargetManaPercentage int
	AttackerType         AdversaryType
	weight               float64
}

func (a SingleAttacker) CalculateWeightTotalConfig() (newNodesCount int, newTotalWeight float64) {
	newTotalWeight = float64(config.NodesTotalWeight) - a.weight
	newNodesCount = config.NodesCount - 1
	return
}
func insert[V constraints.Numeric](array []V, element V, i int) []V {
	array = append(array[:i+1], array[i:]...)
	array[i] = element
	return array
}

func (a SingleAttacker) UpdateAttackerWeight(weights []uint64) []uint64 {
	return insert(weights, uint64(a.weight), a.nodeID)
}

func NewSingleAttacker() *SingleAttacker {
	return &SingleAttacker{
		weight:               float64(config.BlowballMana) * float64(config.NodesTotalWeight) / 100,
		nodeID:               config.BlowballNodeID,
		TargetManaPercentage: config.BlowballMana,
		AttackerType:         Blowball,
	}
}

// todo use generics
//type (
//	Vanilla  uint
//	Blowball uint
//)
//type AttackerType interface {
//	Blowball | Vanilla
//}
//
//func getNode[T AttackerType](node singlenodeattacks.AttackerNode) T {
//	return node.(T)
//}

func IsAttacker(nodeID int) bool {
	return nodeID == config.BlowballNodeID
}
