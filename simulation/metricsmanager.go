package simulation

import (
	"encoding/csv"
	"fmt"
	"time"

	"github.com/iotaledger/hive.go/core/generics/constraints"
	"github.com/iotaledger/hive.go/types"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/multiverse"
	"github.com/iotaledger/multivers-simulation/network"
)

type Csv struct {
	filename string
	header   []string
}

type MetricsManager struct {
	network *network.Network

	// metrics
	GlobalCounters    *AtomicCounters[string, int64]
	PeerCounters      *MapCounters[network.PeerID, int64]
	ColorCounters     *MapCounters[multiverse.Color, int64]
	AdversaryCounters *MapCounters[multiverse.Color, int64]

	// internal variables for the metrics
	RGBColors           []multiverse.Color
	uRGBColors          []multiverse.Color
	adversaryNodesCount int
	honestNodesCount    int
	highestWeightPeerID int
	allPeerIDs          []network.PeerID // all peers in the network
	watchedPeerIDs      []network.PeerID // peers with collected more specific metrics
	simulationStartTime time.Time
	dsIssuanceTime      time.Time

	// csv writers
	writers           map[string]*csv.Writer
	collectFuncs      map[string]func() csvRows
	dumpingTicker     *time.Ticker
	onShutdownDumpers []func()

	shutdown chan types.Empty

	dumpOnEventUsed     bool
	dumpOnEventShutdown chan types.Empty
}

func NewMetricsManager() *MetricsManager {
	return &MetricsManager{
		GlobalCounters:    NewAtomicCounters[string, int64](),
		PeerCounters:      NewCounters[network.PeerID, int64](),
		ColorCounters:     NewCounters[multiverse.Color, int64](),
		AdversaryCounters: NewCounters[multiverse.Color, int64](),

		allPeerIDs:     make([]network.PeerID, 0),
		watchedPeerIDs: make([]network.PeerID, 0),

		writers:      make(map[string]*csv.Writer),
		collectFuncs: make(map[string]func() csvRows),

		shutdown:            make(chan types.Empty),
		dumpOnEventShutdown: make(chan types.Empty),
	}
}

func (s *MetricsManager) Setup(network *network.Network) {
	s.network = network
	s.SetupInternalVariables()
	DumpConfig(fmt.Sprint("aw-", formatTime(s.simulationStartTime), ".config"))
	s.SetupMetrics()
	s.SetupMetricsCollection()
	s.SetupWriters()
}

func (s *MetricsManager) SetupInternalVariables() {
	s.RGBColors = []multiverse.Color{multiverse.Red, multiverse.Green, multiverse.Blue}
	s.uRGBColors = []multiverse.Color{multiverse.UndefinedColor, multiverse.Red, multiverse.Green, multiverse.Blue}
	s.adversaryNodesCount = len(network.AdversaryNodeIDToGroupIDMap) // todo can we define it with config info only?
	s.honestNodesCount = config.Params.NodesCount - s.adversaryNodesCount
	s.highestWeightPeerID = 0 // todo make sure all simulation modes has 0 index as the highest weight peer
	for _, peer := range s.network.Peers {
		s.allPeerIDs = append(s.allPeerIDs, peer.ID)
	}
	// peers with collected more specific metrics, can be set in config
	for _, monitoredID := range config.Params.MonitoredAWPeers {
		s.watchedPeerIDs = append(s.watchedPeerIDs, network.PeerID(monitoredID))
	}
	s.simulationStartTime = time.Now()
}

func (s *MetricsManager) StartMetricsCollection() {
	s.dumpingTicker = time.NewTicker(time.Duration(config.Params.SlowdownFactor*config.Params.ConsensusMonitorTick) * time.Millisecond)
	go func() {
		for {
			select {
			case <-s.dumpingTicker.C:
				s.collectMetrics()
			case <-s.shutdown:
				for _, w := range s.writers {
					w.Flush()
				}

				// todo move final condition reaching detection to some more accurate place
				// determines whether consensus has been reached and simulation is over
				r, g, b := getLikesPerRGB(s.ColorCounters, "confirmedNodes")
				aR, aG, aB := getLikesPerRGB(s.AdversaryCounters, "confirmedNodes")
				hR, hG, hB := r-aR, g-aG, b-aB
				if max(max(hB, hR), hG) >= int64(config.Params.SimulationStopThreshold*float64(s.honestNodesCount)) {
					//shutdownSignal <- types.Void
				}
				s.GlobalCounters.Set("tps", 0)
				return
			}
		}

	}()
}

func (s *MetricsManager) Shutdown() {
	s.shutdown <- types.Void

	s.dumpOnShutdown()
	if s.dumpingTicker != nil {
		s.dumpingTicker.Stop()
	}
	s.dumpOnEventShutdown <- types.Void
}

func (s *MetricsManager) dumpOnShutdown() {
	for _, collector := range s.onShutdownDumpers {
		collector()
	}
}

func (s *MetricsManager) collectMetrics() {
	for key := range s.writers {
		s.collect(key)
	}
}

func (s *MetricsManager) collect(writerKey string) {
	writer := s.writers[writerKey]
	record := s.collectFuncs[writerKey]()
	for _, row := range record {
		if err := writer.Write(row); err != nil {
			log.Fatal("error writing record to csv:", err)
		}
	}

	if err := writer.Error(); err != nil {
		log.Fatal(err)
	}
	writer.Flush()
}

func (s *MetricsManager) SetDSIssuanceTime() {
	s.dsIssuanceTime = time.Now()
}

func allNodesHeader() []string {
	header := make([]string, 0, config.Params.NodesCount+1)
	for i := 0; i < config.Params.NodesCount; i++ {
		header = append(header, fmt.Sprintf("Node %d", i))
	}
	header = append(header, "ns since start")
	return header
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func getLikesPerRGB(counter *MapCounters[multiverse.Color, int64], flag string) (int64, int64, int64) {
	return counter.Get(flag, multiverse.Red), counter.Get(flag, multiverse.Green), counter.Get(flag, multiverse.Blue)
}

// max returns the largest of x or y.
func max[T constraints.Numeric](x, y T) T {
	if x < y {
		return y
	}
	return x
}
