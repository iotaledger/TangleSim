package adversary

import (
	"github.com/iotaledger/multivers-simulation/multiverse"
	"github.com/iotaledger/multivers-simulation/network"
)

type Node struct {
	Peer    *network.Peer
	Tangle  *multiverse.Tangle
	AdvType network.AdversaryType
}
