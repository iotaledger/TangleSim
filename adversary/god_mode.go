package adversary

import (
	"github.com/iotaledger/multivers-simulation/multiverse"
	"github.com/iotaledger/multivers-simulation/network"
	"time"
)

// adversary will be issuing after each honest node
// adv will have some % of mana
// his reaction can be delayed

type GodMode struct {
	node           multiverse.NodeInterface
	adversaryDelay time.Duration
	seenMessages   []*multiverse.Message
}

func NewGodMode(node *NodeInterface) {

}

func (g *GodMode) ListenToAllNodes(net *network.Network) {
	// attach to MessageStored event of storage component of each node

}
