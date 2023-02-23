package simulation

import (
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/multiverse"
	"github.com/iotaledger/multivers-simulation/network"
)

type Simulator struct {
	// general metrics
	GlobalCounters *AtomicCounters[string, int64]
	PeerCounters   *MapCounters[network.PeerID, int64]
	// double spend metrics
	ColorCounters     *MapCounters[multiverse.Color, int64]
	AdversaryCounters *MapCounters[multiverse.Color, int64]

	// internal variables calculated from config
	RGBColors  []multiverse.Color
	uRGBColors []multiverse.Color

	adversaryNodesCount int
	honestNodesCount    int
	highestWeightPeerID int
	allPeerIDs          []network.PeerID // all peers in the network
	watchedPeerIDs      []network.PeerID // peers with collected more specific metrics
}

func NewSimulator() *Simulator {
	return &Simulator{
		GlobalCounters:    NewAtomicCounters[string, int64](),
		PeerCounters:      NewCounters[network.PeerID, int64](),
		ColorCounters:     NewCounters[multiverse.Color, int64](),
		AdversaryCounters: NewCounters[multiverse.Color, int64](),

		allPeerIDs:     make([]network.PeerID, 0),
		watchedPeerIDs: make([]network.PeerID, 0),
	}
}

func (s *Simulator) Setup(testNetwork *network.Network) {
	s.SetupInternalVariables(testNetwork)
	s.SetupMetrics()
	s.SetupMetricsCollection(testNetwork)
}

func (s *Simulator) SetupInternalVariables(n *network.Network) {
	s.RGBColors = []multiverse.Color{multiverse.Red, multiverse.Green, multiverse.Blue}
	s.uRGBColors = []multiverse.Color{multiverse.UndefinedColor, multiverse.Red, multiverse.Green, multiverse.Blue}
	s.adversaryNodesCount = len(network.AdversaryNodeIDToGroupIDMap) // todo can we define it with config info only?
	s.honestNodesCount = config.NodesCount - s.adversaryNodesCount
	s.highestWeightPeerID = 0 // todo make sure all simulation modes has 0 index as the highest weight peer
	for _, peer := range n.Peers {
		s.allPeerIDs = append(s.allPeerIDs, peer.ID)
	}
	// peers with collected more specific metrics, can be set in config
	for _, monitoredID := range config.MonitoredPeers {
		s.watchedPeerIDs = append(s.watchedPeerIDs, network.PeerID(monitoredID))
	}
}
