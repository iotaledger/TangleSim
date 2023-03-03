package main

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/iotaledger/multivers-simulation/singlenodeattacks"

	"github.com/iotaledger/multivers-simulation/adversary"
	"github.com/iotaledger/multivers-simulation/simulation"

	"github.com/iotaledger/hive.go/types"

	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/logger"
	"github.com/iotaledger/multivers-simulation/multiverse"
	"github.com/iotaledger/multivers-simulation/network"
)

var (
	log          = logger.New("Simulation")
	MetricsMgr   *simulation.MetricsManager
	simulationWg = sync.WaitGroup{}
	// simulation variables
	shutdownSignal = make(chan types.Empty)
)

func main() {
	log.Info("Starting simulation ... [DONE]")
	defer log.Info("Shutting down simulation ... [DONE]")
	simulation.ParseFlags()

	nodeFactories := map[network.AdversaryType]network.NodeFactory{
		network.HonestNode:     network.NodeClosure(multiverse.NewNode),
		network.ShiftOpinion:   network.NodeClosure(adversary.NewShiftingOpinionNode),
		network.TheSameOpinion: network.NodeClosure(adversary.NewSameOpinionNode),
		network.NoGossip:       network.NodeClosure(adversary.NewNoGossipNode),
		network.Blowball:       network.NodeClosure(singlenodeattacks.NewBlowballNode),
	}
	testNetwork := network.New(
		network.Nodes(config.NodesCount, nodeFactories, network.ZIPFDistribution(
			config.ZipfParameter)),
		network.Delay(time.Duration(config.SlowdownFactor)*time.Duration(config.MinDelay)*time.Millisecond,
			time.Duration(config.SlowdownFactor)*time.Duration(config.MaxDelay)*time.Millisecond),
		network.PacketLoss(config.PacketLoss, config.PacketLoss),
		network.Topology(network.WattsStrogatz(config.NeighbourCountWS, config.RandomnessWS)),
		network.AdversaryPeeringAll(config.AdversaryPeeringAll),
		network.AdversarySpeedup(config.AdversarySpeedup),
	)
	MetricsMgr = simulation.NewMetricsManager()
	MetricsMgr.Setup(testNetwork)

	monitorNetworkState()
	MetricsMgr.StartMetricsCollection()

	// start a go routine for each node to start issuing messages
	startIssuingMessages(testNetwork)
	// start a go routine for each node to start processing messages received from neighbours and scheduling.
	startProcessingMessages(testNetwork)
	defer testNetwork.Shutdown()

	// To simulate the confirmation time w/o any double spending, the colored msgs are not to be sent
	SimulateAdversarialBehaviour(testNetwork)

	select {
	case <-shutdownSignal:
		shutdownSimulation()
		log.Info("Shutting down simulation (consensus reached) ... [DONE]")
	case <-time.After(time.Duration(config.SlowdownFactor) * config.SimulationDuration):
		shutdownSimulation()
		log.Info("Shutting down simulation (simulation timed out) ... [DONE]")
	}
}

func startProcessingMessages(n *network.Network) {
	for _, peer := range n.Peers {
		// The Blowball attacker does not need to process the message
		// TODO: Also disable `processMessages` for other attackers which do not require it.
		// todo not sure if processing message should be disabled, as node needs to have complete tangle to walk
		if !(config.SimulationMode == "Blowball" &&
			network.IsAttacker(int(peer.ID))) {
			go processMessages(peer)
		}
	}
}

func processMessages(peer *network.Peer) {
	simulationWg.Add(1)
	defer simulationWg.Done()
	pace := time.Duration((float64(time.Second) * float64(config.SlowdownFactor)) / float64(config.SchedulingRate))
	ticker := time.NewTicker(pace)
	for {
		select {
		case <-peer.ShutdownProcessing:
			return
		case networkMessage := <-peer.Socket:
			peer.Node.HandleNetworkMessage(networkMessage) // this includes payloads from the node itself so block are created here
		case <-ticker.C:

			// Trigger the scheduler to pop messages and gossip them
			peer.Node.(multiverse.NodeInterface).Tangle().Scheduler.IncrementAccessMana(float64(config.SchedulingRate))
			peer.Node.(multiverse.NodeInterface).Tangle().Scheduler.ScheduleMessage()
		}
	}
}

func SimulateAdversarialBehaviour(testNetwork *network.Network) {
	switch config.SimulationMode {
	case "Accidental":
		for i, node := range network.GetAccidentalIssuers(testNetwork) {
			color := multiverse.ColorFromInt(i + 1)
			go sendMessage(node, color)
			log.Infof("Peer %d sent double spend msg: %v", node.ID, color)
		}
	// todo adversary should be renamed to doublespend
	case "Adversary":
		time.Sleep(time.Duration(config.DoubleSpendDelay*config.SlowdownFactor) * time.Second)
		// Here we simulate the double spending
		MetricsMgr.SetDSIssuanceTime()
		for _, group := range testNetwork.AdversaryGroups {
			color := multiverse.ColorFromStr(group.InitColor)

			for _, nodeID := range group.NodeIDs {
				peer := testNetwork.Peer(nodeID)
				// honest node does not implement adversary behavior interface
				if group.AdversaryType != network.HonestNode {
					node := adversary.CastAdversary(peer.Node)
					node.AssignColor(color)
				}
				go sendMessage(peer, color)
				log.Infof("Peer %d sent double spend msg: %v", peer.ID, color)
			}
		}
	case "Blowball":
		ticker := time.NewTicker(time.Duration(config.SlowdownFactor*config.BlowballDelay) * time.Second)
		alreadySentCounter := 0
		for {
			if alreadySentCounter == config.BlowballMaxSent {
				ticker.Stop()
				break
			}
			select {
			case <-ticker.C:
				for _, group := range testNetwork.AdversaryGroups {
					for _, nodeID := range group.NodeIDs {
						peer := testNetwork.Peer(nodeID)
						go sendMessage(peer, multiverse.UndefinedColor)
						alreadySentCounter++
					}
				}
			}
		}

	}
}

func startIssuingMessages(testNetwork *network.Network) {
	fmt.Println("totalWeight ", testNetwork.WeightDistribution.TotalWeight())
	if testNetwork.WeightDistribution.TotalWeight() == 0 {
		panic("total weight is 0")
	}
	nodeTotalWeight := float64(testNetwork.WeightDistribution.TotalWeight())

	for _, peer := range testNetwork.Peers {
		weightOfPeer := float64(testNetwork.WeightDistribution.Weight(peer.ID))
		log.Warn("Peer ID Weight: ", peer.ID, weightOfPeer, nodeTotalWeight)
		MetricsMgr.GlobalCounters.Add("relevantValidators", 1)

		// peer.AdversarySpeedup=1 for honest nodes and can have different values from adversary nodes
		band := peer.AdversarySpeedup * weightOfPeer * float64(config.IssuingRate) / nodeTotalWeight
		// fmt.Println(peer.AdversarySpeedup, weightOfPeer, config.IssuingRate, nodeTotalWeight)
		//fmt.Printf("speedup %f band %f\n", peer.AdversarySpeedup, band)
		go issueMessages(peer, band)
	}
}

func issueMessages(peer *network.Peer, band float64) {
	simulationWg.Add(1)
	defer simulationWg.Done()
	pace := time.Duration(float64(time.Second) * float64(config.SlowdownFactor) / band)

	if pace == time.Duration(0) {
		log.Warn("Peer ID: ", peer.ID, " has 0 pace!")
		return
	}
	ticker := time.NewTicker(pace)
	congestionTicker := time.NewTicker(time.Duration(config.SlowdownFactor) * config.SimulationDuration / time.Duration(len(config.CongestionPeriods)))
	band *= config.CongestionPeriods[0]
	i := 0
	for {
		select {
		case <-peer.ShutdownIssuing:
			return
		case <-ticker.C:
			if config.IMIF == "poisson" {
				pace = time.Duration(float64(time.Second) * float64(config.SlowdownFactor) * rand.ExpFloat64() / band)
				if pace > 0 {
					ticker.Reset(pace)
				}
			}
			sendMessage(peer)
		case <-congestionTicker.C:
			if i < len(config.CongestionPeriods)-1 {
				band *= config.CongestionPeriods[i+1] / config.CongestionPeriods[i]
				i++
			}
		}

	}
}

func sendMessage(peer *network.Peer, optionalColor ...multiverse.Color) {
	//MetricsMgr.GlobalCounters.Add("tps", 1)

	if len(optionalColor) >= 1 {
		peer.Node.(multiverse.NodeInterface).IssuePayload(optionalColor[0])
	}

	peer.Node.(multiverse.NodeInterface).IssuePayload(multiverse.UndefinedColor)
}

func shutdownSimulation() {
	simulationWg.Wait()
	MetricsMgr.Shutdown()

}

// todo add to metrics manager on shutdown if needed
func flushWriters(writers []*csv.Writer) {
	for _, writer := range writers {
		writer.Flush()
		err := writer.Error()
		if err != nil {
			log.Error(err)
		}
	}
}

func monitorNetworkState() (resultsWriters []*csv.Writer) {
	// todo add most liked color counters to metrics manager
	//mostLikedColor = multiverse.UndefinedColoric
	//honestOnlyMostLikedColor = multiverse.UndefinedColor

	return
}

// ArgMax returns the max value of the array.
func ArgMax(x []int64) int {
	maxLocation := 0
	currentMax := int64(x[0])
	for i, v := range x[1:] {
		if v > currentMax {
			currentMax = v
			maxLocation = i + 1
		}
	}
	return maxLocation
}

func mostLikedColorChanged(r, g, b int64, mostLikedColorVar *multiverse.Color) bool {

	currentMostLikedColor := multiverse.UndefinedColor
	if g > 0 {
		currentMostLikedColor = multiverse.Green
	}
	if b > g {
		currentMostLikedColor = multiverse.Blue
	}
	if r > b && r > g {
		currentMostLikedColor = multiverse.Red
	}
	// color selected
	if *mostLikedColorVar != currentMostLikedColor {
		// color selected for the first time, it not counts
		if *mostLikedColorVar == multiverse.UndefinedColor {
			*mostLikedColorVar = currentMostLikedColor
			return false
		}
		*mostLikedColorVar = currentMostLikedColor
		return true
	}
	return false
}
