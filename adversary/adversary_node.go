package adversary

import (
	"github.com/iotaledger/multivers-simulation/logger"
	"github.com/iotaledger/multivers-simulation/multiverse"
	"github.com/iotaledger/multivers-simulation/network"
	"reflect"
)

var log = logger.New("Adversary")

type NodeInterface interface {
	AssignColor(color multiverse.Color)
}

func CastAdversary(node network.Node) NodeInterface {
	s := reflect.ValueOf(node)
	switch s.Interface().(type) {
	case *ShiftingOpinionNode:
		return node.(*ShiftingOpinionNode)
	case *SameOpinionNode:
		return node.(*SameOpinionNode)
	case *NoGossipNode:
		return node.(*NoGossipNode)
	}
	return nil
}
