package simulation

import (
	"github.com/iotaledger/multivers-simulation/network"
)

type Simulator struct {
	network *network.Network
	metrics *MetricsManager
}
