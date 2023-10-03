package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/iotaledger/multivers-simulation/adversary"
	"github.com/iotaledger/multivers-simulation/simulation"
	"github.com/iotaledger/multivers-simulation/singlenodeattacks"

	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/hive.go/types"
	"github.com/iotaledger/hive.go/typeutils"

	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/logger"
	"github.com/iotaledger/multivers-simulation/multiverse"
	"github.com/iotaledger/multivers-simulation/network"
)

var (
	log = logger.New("Simulation")

	// csv
	awHeader = []string{"Message ID", "Issuance Time (unix)", "Confirmation Time (ns)", "Weight", "# of Confirmed Messages",
		"# of Issued Messages", "ns since start"}
	wwHeader = []string{"Witness Weight", "Time (ns)"}
	dsHeader = []string{"UndefinedColor", "Blue", "Red", "Green", "ns since start", "ns since issuance"}
	mmHeader = []string{"Number of Requested Messages", "ns since start"}
	tpHeader = []string{"UndefinedColor (Tip Pool Size)", "Blue (Tip Pool Size)", "Red (Tip Pool Size)", "Green (Tip Pool Size)",
		"UndefinedColor (Processed)", "Blue (Processed)", "Red (Processed)", "Green (Processed)", "# of Issued Messages", "ns since start"}

	ccHeader = []string{"Blue (Confirmed)", "Red (Confirmed)", "Green (Confirmed)",
		"Blue (Adversary Confirmed)", "Red (Adversary Confirmed)", "Green (Adversary Confirmed)",
		"Blue (Confirmed Accumulated Weight)", "Red (Confirmed Accumulated Weight)", "Green (Confirmed Accumulated Weight)",
		"Blue (Confirmed Adversary Weight)", "Red (Confirmed Adversary Weight)", "Green (Confirmed Adversary Weight)",
		"Blue (Like)", "Red (Like)", "Green (Like)",
		"Blue (Like Accumulated Weight)", "Red (Like Accumulated Weight)", "Green (Like Accumulated Weight)",
		"Blue (Adversary Like Accumulated Weight)", "Red (Adversary Like Accumulated Weight)", "Green (Adversary Like Accumulated Weight)",
		"Unconfirmed Blue", "Unconfirmed Red", "Unconfirmed Green",
		"Unconfirmed Blue Accumulated Weight", "Unconfirmed Red Accumulated Weight", "Unconfirmed Green Accumulated Weight",
		"Flips (Winning color changed)", "Honest nodes Flips", "ns since start", "ns since issuance"}
	adHeader = []string{"AdversaryGroupID", "Strategy", "AdversaryCount", "q", "ns since issuance"}
	ndHeader = []string{"Node ID", "Adversary", "Min Confirmed Accumulated Weight", "Unconfirmation Count"}

	csvMutex sync.Mutex

	// simulation variables
	globalMetricsTicker = time.NewTicker(time.Duration(config.Params.SlowdownFactor*config.Params.ConsensusMonitorTick) * time.Millisecond)
	simulationWg        = sync.WaitGroup{}
	shutdownSignal      = make(chan types.Empty)

	// global declarations
	dsIssuanceTime           time.Time
	mostLikedColor           multiverse.Color
	honestOnlyMostLikedColor multiverse.Color
	simulationStartTime      time.Time
	// MetricsMgr               *simulation.MetricsManager

	// counters
	colorCounters     = simulation.NewColorCounters()
	adversaryCounters = simulation.NewColorCounters()
	nodeCounters      = []simulation.AtomicCounters[string, int64]{}
	atomicCounters    = simulation.NewAtomicCounters[string, int64]()

	confirmedMessageCounter = make(map[network.PeerID]int64)

	storedMessageMap                 = make(map[multiverse.MessageID]int)
	storedMessages                   = make(map[multiverse.MessageID]*multiverse.Message)
	storedMessageMutex               sync.RWMutex
	disseminatedMessageCounter       = make([]int64, config.Params.NodesCount)
	undisseminatedMessageCounter     = make([]int64, config.Params.NodesCount)
	disseminatedMessageMutex         sync.RWMutex
	disseminatedMessages             = make(map[multiverse.MessageID]*multiverse.Message)
	disseminatedMessageMetadata      = make(map[multiverse.MessageID]*multiverse.MessageMetadata)
	confirmedMessageMutex            sync.RWMutex
	confirmedMessageMap              = make(map[multiverse.MessageID]int)
	firstConfirmedTimeMap            = make(map[multiverse.MessageID]time.Time)
	confirmedDelayInNetworkMap       = make(map[multiverse.MessageID]time.Duration)
	confirmedDelayInNetworkMutex     sync.Mutex
	fullyConfirmedMessageCounter     = make([]int64, config.Params.NodesCount)
	fullyConfirmedMessages           = make(map[multiverse.MessageID]*multiverse.Message)
	fullyConfirmedMessageMetadata    = make(map[multiverse.MessageID]*multiverse.MessageMetadata)
	partiallyConfirmedMessageCounter = make([]int64, config.Params.NodesCount)
	unconfirmedMessageCounter        = make([]int64, config.Params.NodesCount)
	shutdownGlobalMetrics            = make(chan struct{})

	localMetrics        = make(map[string]map[network.PeerID]float64)
	localResultsWriters = make(map[string]*csv.Writer)
	localMetricsMutex   sync.RWMutex
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

	// The simulation start time
	simulationStartTime = time.Now()
	testNetwork := network.New(
		network.Nodes(config.Params.NodesCount,
			nodeFactories,
			network.ZIPFDistribution(config.Params.ZipfParameter),
			network.MixedZIPFDistribution(config.Params.ZipfParameter)),
		network.Delay(time.Duration(config.Params.SlowdownFactor)*time.Duration(config.Params.MinDelay)*time.Millisecond,
			time.Duration(config.Params.SlowdownFactor)*time.Duration(config.Params.MaxDelay)*time.Millisecond),
		network.PacketLoss(config.Params.PacketLoss, config.Params.PacketLoss),
		network.Topology(network.WattsStrogatz(config.Params.NeighbourCountWS, config.Params.RandomnessWS)),
		network.AdversaryPeeringAll(config.Params.AdversaryPeeringAll),
		network.AdversarySpeedup(config.Params.AdversarySpeedup),
		network.GenesisTime(simulationStartTime),
	)
	// MetricsMgr = simulation.NewMetricsManager()
	// MetricsMgr.Setup(testNetwork)

	monitorNetworkState(testNetwork)
	// MetricsMgr.StartMetricsCollection()

	// The simulation start time
	simulationStartTime = time.Now()

	// Dump the configuration of this simulation
	dumpConfig(path.Join(config.Params.ResultDir, config.Params.ScriptStartTimeStr, "mb.config"))
	// Dump the network information
	dumpNetworkConfig(testNetwork)
	// Start monitoring global metrics
	monitorGlobalMetrics(testNetwork)

	// start a go routine for each node to start issuing messages
	startIssuingMessages(testNetwork)
	// start a go routine for each node to start processing messages received from nieghbours and scheduling.
	startProcessingMessages(testNetwork)

	// To simulate the confirmation time w/o any double spending, the colored msgs are not to be sent
	if config.Params.SimulationTarget == "DS" {
		SimulateDoubleSpent(testNetwork)
	}

	select {
	case <-shutdownSignal:
		shutdownSimulation(testNetwork)
		log.Info("Shutting down simulation (consensus reached) ... [DONE]")
	case <-time.After(time.Duration(config.Params.SlowdownFactor) * config.Params.SimulationDuration):
		fmt.Println(">>>>>>>>>>>>>.Simulation timed out")
		shutdownSimulation(testNetwork)
		log.Info("Shutting down simulation (simulation timed out) ... [DONE]")
	}
}

func startProcessingMessages(n *network.Network) {
	for _, peer := range n.Peers {
		// The Blowball attacker does not need to process the message
		// TODO: Also disable `processMessages` for other attackers which do not require it.
		// todo not sure if processing message should be disabled, as node needs to have complete tangle to walk
		if !(config.Params.SimulationMode == "Blowball" &&
			network.IsAttacker(int(peer.ID))) {
			go processMessages(peer)
		}
	}
}

func processMessages(peer *network.Peer) {
	// simulationWg.Add(1)
	// defer simulationWg.Done()

	pace := time.Duration((float64(time.Second) * float64(config.Params.SlowdownFactor)) / float64(config.Params.SchedulingRate))
	ticker := time.NewTicker(pace)
	defer ticker.Stop()

	validatorPace := time.Duration((float64(time.Second) * float64(config.Params.SlowdownFactor)) / float64(config.Params.ValidatorBPS))
	validatorTicker := time.NewTicker(validatorPace)
	defer validatorTicker.Stop()

	for {
		select {
		case <-peer.ShutdownProcessing:
			log.Warn("Shutting down processing for peer", peer.ID)
			return
		case networkMessage := <-peer.Socket:
			peer.Node.HandleNetworkMessage(networkMessage) // this includes payloads from the node itself so block are created here
		case <-ticker.C:

			// Trigger the scheduler to pop messages and gossip them
			peer.Node.(multiverse.NodeInterface).Tangle().Scheduler.IncrementAccessMana(float64(config.Params.SchedulingRate))
			peer.Node.(multiverse.NodeInterface).Tangle().Scheduler.ScheduleMessage()
			monitorLocalMetrics(peer)
		case <-validatorTicker.C:
			if int(peer.ID) <= config.Params.ValidatorCount {
				if message, ok := peer.Node.(multiverse.NodeInterface).Tangle().MessageFactory.CreateMessage(true, multiverse.UndefinedColor); ok {
					peer.Node.(multiverse.NodeInterface).Tangle().ProcessMessage(message)
				}
			}
		}
	}
}

func SimulateAdversarialBehaviour(testNetwork *network.Network) {
	switch config.Params.SimulationMode {
	case "Accidental":
		for i, node := range network.GetAccidentalIssuers(testNetwork) {
			color := multiverse.ColorFromInt(i + 1)
			go sendMessage(node, color)
			log.Infof("Peer %d sent double spend msg: %v", node.ID, color)
		}
	// todo adversary should be renamed to doublespend
	case "Adversary":
		time.Sleep(time.Duration(config.Params.DoubleSpendDelay*config.Params.SlowdownFactor) * time.Second)
		// Here we simulate the double spending
		// MetricsMgr.SetDSIssuanceTime()
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
		ticker := time.NewTicker(time.Duration(config.Params.SlowdownFactor*config.Params.BlowballDelay) * time.Second)
		alreadySentCounter := 0
		for {
			if alreadySentCounter == config.Params.BlowballMaxSent {
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
		// MetricsMgr.GlobalCounters.Add("relevantValidators", 1)

		// peer.AdversarySpeedup=1 for honest nodes and can have different values from adversary nodes
		// band := peer.AdversarySpeedup * weightOfPeer * float64(config.Params.IssuingRate) / nodeTotalWeight
		band := peer.AdversarySpeedup * float64(testNetwork.BandwidthDistribution.Bandwidth(peer.ID))
		// log.Debugf("startIssuingMessages... Peer ID: %d, Bandwidth: %f", peer.ID, band)
		// fmt.Println(peer.AdversarySpeedup, weightOfPeer, config.Params.IssuingRate, nodeTotalWeight)
		//fmt.Printf("speedup %f band %f\n", peer.AdversarySpeedup, band)
		go issueMessages(peer, band)
	}
}

func issueMessages(peer *network.Peer, band float64) {
	// simulationWg.Add(1)
	// defer simulationWg.Done()

	pace := time.Duration(float64(time.Second) * float64(config.Params.SlowdownFactor) / band)

	if pace == time.Duration(0) {
		log.Warn("Peer ID: ", peer.ID, " has 0 pace!")
		return
	}
	ticker := time.NewTicker(pace)
	congestionTicker := time.NewTicker(time.Duration(config.Params.SlowdownFactor) * config.Params.SimulationDuration / time.Duration(len(config.Params.CongestionPeriods)))
	defer ticker.Stop()
	defer congestionTicker.Stop()

	band *= config.Params.CongestionPeriods[0]
	i := 0
	for {
		select {
		case <-peer.ShutdownIssuing:
			log.Warn("Peer ID: ", peer.ID, " has been shutdown!")
			return
		case <-ticker.C:
			if config.Params.IMIF == "poisson" {
				pace = time.Duration(float64(time.Second) * float64(config.Params.SlowdownFactor) * rand.ExpFloat64() / band)
				if pace > 0 {
					ticker.Reset(pace)
				}
			}

			// TODO: for attackers, they don't use the rate setter but will issue as many as blocks to fill up the network traffic
			//       and they will use higher-frequency ticker to issue more blocks
			if peer.Node.(multiverse.NodeInterface).Tangle().Scheduler.RateSetter() {
				sendMessage(peer)
			}

		case <-congestionTicker.C:
			if i < len(config.Params.CongestionPeriods)-1 {
				band *= config.Params.CongestionPeriods[i+1] / config.Params.CongestionPeriods[i]
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

func shutdownSimulation(net *network.Network) {
	net.Shutdown()
	close(shutdownGlobalMetrics)
	dumpAcceptanceLatencyAmongNodes()
	dumpFinalData(net)
	simulationWg.Wait()
	//dumpAllMessageMetaData(net.Peers[0].Node.(multiverse.NodeInterface).Tangle().Storage)
}

func monitorLocalMetrics(peer *network.Peer) {
	localMetricsMutex.Lock()
	defer localMetricsMutex.Unlock()

	if len(localMetrics) != 0 {
		localMetrics["Ready Lengths"][peer.ID] = float64(peer.Node.(multiverse.NodeInterface).Tangle().Scheduler.ReadyLen())
		localMetrics["Non Ready Lengths"][peer.ID] = float64(peer.Node.(multiverse.NodeInterface).Tangle().Scheduler.NonReadyLen())
		localMetrics["Own Mana"][peer.ID] = float64(peer.Node.(multiverse.NodeInterface).Tangle().Scheduler.GetNodeAccessMana(peer.ID))
		localMetrics["Tips"][peer.ID] = float64(peer.Node.(multiverse.NodeInterface).Tangle().TipManager.TipSet(0).Size())
		localMetrics["Price"][peer.ID] = float64(peer.Node.(multiverse.NodeInterface).Tangle().Scheduler.GetMaxManaBurn())
		currentSlotIndex := peer.Node.(multiverse.NodeInterface).Tangle().Storage.SlotIndex(time.Now())
		localMetrics["RMC"][peer.ID] = float64(peer.Node.(multiverse.NodeInterface).Tangle().Storage.RMC(currentSlotIndex))
		localMetrics["Time since ATT"][peer.ID] = float64(time.Since(peer.Node.(multiverse.NodeInterface).Tangle().Storage.ATT).Seconds())
		if peer.ID == 0 {
			for i := 0; i < config.Params.NodesCount; i++ {
				localMetrics["Mana at Node 0"][network.PeerID(i)] = float64(peer.Node.(multiverse.NodeInterface).Tangle().Scheduler.GetNodeAccessMana(network.PeerID(i)))
				localMetrics["Issuer Queue Lengths at Node 0"][network.PeerID(i)] = float64(peer.Node.(multiverse.NodeInterface).Tangle().Scheduler.IssuerQueueLen(network.PeerID(i)))
				localMetrics["Deficits at Node 0"][network.PeerID(i)] = float64(peer.Node.(multiverse.NodeInterface).Tangle().Scheduler.Deficit(network.PeerID(i)))
			}
		}
	} else {
		localMetrics["Ready Lengths"] = make(map[network.PeerID]float64)
		localMetrics["Non Ready Lengths"] = make(map[network.PeerID]float64)
		localMetrics["Own Mana"] = make(map[network.PeerID]float64)
		localMetrics["Tips"] = make(map[network.PeerID]float64)
		localMetrics["Price"] = make(map[network.PeerID]float64)
		localMetrics["RMC"] = make(map[network.PeerID]float64)
		localMetrics["Mana at Node 0"] = make(map[network.PeerID]float64)
		localMetrics["Issuer Queue Lengths at Node 0"] = make(map[network.PeerID]float64)
		localMetrics["Deficits at Node 0"] = make(map[network.PeerID]float64)
		localMetrics["Time since ATT"] = make(map[network.PeerID]float64)
	}
}

func dumpLocalMetrics() {
	simulationWg.Add(1)
	defer simulationWg.Done()
	timeSinceStart := time.Since(simulationStartTime).Nanoseconds()
	timeStr := strconv.FormatInt(timeSinceStart, 10)

	localMetricsMutex.RLock()
	defer localMetricsMutex.RUnlock()
	for name := range localMetrics {
		if _, exists := localResultsWriters[name]; !exists { // create the file and results writer if it doesn't already exist
			lmHeader := make([]string, 0, config.Params.NodesCount+1)
			for i := 0; i < config.Params.NodesCount; i++ {
				header := []string{fmt.Sprintf("Node %d", i)}
				lmHeader = append(lmHeader, header...)
			}
			header := []string{"ns since start"}
			lmHeader = append(lmHeader, header...)

			file, err := createFile(path.Join(config.Params.GeneralOutputDir, strings.Join([]string{name, ".csv"}, "")))
			if err != nil {
				panic(err)
			}
			localResultsWriters[name] = csv.NewWriter(file)
			if err := localResultsWriters[name].Write(lmHeader); err != nil {
				panic(err)
			}
		}
		record := make([]string, config.Params.NodesCount+1)
		for id := 0; id < config.Params.NodesCount; id++ {
			record[id] = strconv.FormatFloat(localMetrics[name][network.PeerID(id)], 'f', 6, 64)
		}
		record[config.Params.NodesCount] = timeStr
		if err := localResultsWriters[name].Write(record); err != nil {
			panic(err)
		}
		localResultsWriters[name].Flush()
	}
}

func dumpGlobalMetrics(dissemResultsWriter, undissemResultsWriter, confirmationResultsWriter, partialConfirmationResultsWriter, unconfirmationResultsWriter *csv.Writer) {
	simulationWg.Add(1)
	defer simulationWg.Done()
	timeSinceStart := time.Since(simulationStartTime).Nanoseconds()
	timeStr := strconv.FormatInt(timeSinceStart, 10)
	log.Debug("Simulation Completion: ", int(100*float64(timeSinceStart)/(float64(config.Params.SlowdownFactor)*float64(config.Params.SimulationDuration))), "%")
	disseminatedMessageMutex.RLock()
	record := make([]string, config.Params.NodesCount+1)
	for id := 0; id < config.Params.NodesCount; id++ {
		record[id] = strconv.FormatInt(disseminatedMessageCounter[id], 10)
	}
	disseminatedMessageMutex.RUnlock()
	//log.Debug("Disseminated Messages: ", record)
	record[config.Params.NodesCount] = timeStr
	if err := dissemResultsWriter.Write(record); err != nil {
		panic(err)
	}
	disseminatedMessageMutex.RLock()
	record = make([]string, config.Params.NodesCount+1)
	for id := 0; id < config.Params.NodesCount; id++ {
		record[id] = strconv.FormatInt(undisseminatedMessageCounter[id], 10)
	}
	disseminatedMessageMutex.RUnlock()
	//log.Debug("Disseminated Messages: ", record)
	record[config.Params.NodesCount] = timeStr
	if err := undissemResultsWriter.Write(record); err != nil {
		panic(err)
	}

	confirmedMessageMutex.RLock()
	record = make([]string, config.Params.NodesCount+1)
	for id := 0; id < config.Params.NodesCount; id++ {
		record[id] = strconv.FormatInt(fullyConfirmedMessageCounter[id], 10)
	}
	confirmedMessageMutex.RUnlock()
	//log.Debug("Confirmed Messages: ", record)
	record[config.Params.NodesCount] = timeStr
	if err := confirmationResultsWriter.Write(record); err != nil {
		panic(err)
	}
	confirmedMessageMutex.RLock()
	record = make([]string, config.Params.NodesCount+1)
	for id := 0; id < config.Params.NodesCount; id++ {
		record[id] = strconv.FormatInt(partiallyConfirmedMessageCounter[id], 10)
	}
	confirmedMessageMutex.RUnlock()
	//log.Debug("Partially Confirmed Messages: ", record)
	record[config.Params.NodesCount] = timeStr
	if err := partialConfirmationResultsWriter.Write(record); err != nil {
		panic(err)
	}
	confirmedMessageMutex.RLock()
	record = make([]string, config.Params.NodesCount+1)
	for id := 0; id < config.Params.NodesCount; id++ {
		record[id] = strconv.FormatInt(unconfirmedMessageCounter[id], 10)
	}
	confirmedMessageMutex.RUnlock()
	//log.Debug("Unconfirmed Messages: ", record)
	record[config.Params.NodesCount] = timeStr
	if err := unconfirmationResultsWriter.Write(record); err != nil {
		panic(err)
	}

	// Flush the results writer to avoid truncation.
	dissemResultsWriter.Flush()
	undissemResultsWriter.Flush()
	confirmationResultsWriter.Flush()
	partialConfirmationResultsWriter.Flush()
	unconfirmationResultsWriter.Flush()
}

func monitorGlobalMetrics(net *network.Network) {
	// check for global network events such as dissemination and confirmation.
	for id := 0; id < config.Params.NodesCount; id++ {
		mbPeer := net.Peers[id]
		if typeutils.IsInterfaceNil(mbPeer) {
			panic(fmt.Sprintf("unknowm peer with id %d", id))
		}

		mbPeer.Node.(multiverse.NodeInterface).Tangle().Storage.Events.MessageStored.Attach(
			events.NewClosure(func(messageID multiverse.MessageID, message *multiverse.Message, messageMetadata *multiverse.MessageMetadata) {
				storedMessageMutex.Lock()
				if numNodes, exists := storedMessageMap[messageID]; exists {
					if numNodes > config.Params.NodesCount {
						panic("message stored more than once per node")
					}
					storedMessageMap[messageID] = numNodes + 1
					storedMessages[messageID] = message
				} else {
					storedMessageMap[messageID] = 1
					confirmedMessageMutex.Lock()
					unconfirmedMessageCounter[message.Issuer] += 1
					confirmedMessageMutex.Unlock()
					disseminatedMessageMutex.Lock()
					undisseminatedMessageCounter[message.Issuer] += 1
					disseminatedMessageMutex.Unlock()
				}
				// a message is disseminated if it has been stored by all nodes.
				if storedMessageMap[messageID] == config.Params.NodesCount {
					disseminatedMessageMutex.Lock()
					disseminatedMessageCounter[message.Issuer] += 1
					undisseminatedMessageCounter[message.Issuer] -= 1
					disseminatedMessages[messageID] = message
					//log.Debug("Mana Burn value: ", message.ManaBurnValue)
					disseminatedMessageMetadata[messageID] = messageMetadata
					disseminatedMessageMutex.Unlock()
				}
				storedMessageMutex.Unlock()

			}))
		mbPeer.Node.(multiverse.NodeInterface).Tangle().ApprovalManager.Events.MessageConfirmed.Attach(
			events.NewClosure(func(message *multiverse.Message, messageMetadata *multiverse.MessageMetadata, weight uint64, messageIDCounter int64) {
				confirmedMessageMutex.Lock()
				defer confirmedMessageMutex.Unlock()
				if numNodes, exists := confirmedMessageMap[message.ID]; exists {
					if numNodes > config.Params.NodesCount {
						panic("message confirmed more than once per node")
					}
					confirmedMessageMap[message.ID] = numNodes + 1
				} else {
					confirmedMessageMap[message.ID] = 1
					partiallyConfirmedMessageCounter[message.Issuer] += 1
					unconfirmedMessageCounter[message.Issuer] -= 1
				}
				// a message is disseminated if it has been confirmed by all nodes.
				if confirmedMessageMap[message.ID] == config.Params.NodesCount {
					partiallyConfirmedMessageCounter[message.Issuer] -= 1
					fullyConfirmedMessageCounter[message.Issuer] += 1
					fullyConfirmedMessages[message.ID] = message
					fullyConfirmedMessageMetadata[message.ID] = messageMetadata
				}

				// The accepted time difference between the node which first accepted it and the last node which accepted it lastly
				confirmedDelayInNetworkMutex.Lock()
				defer confirmedDelayInNetworkMutex.Unlock()
				// TODO (confirmation): monitor PreAcceptedTime, AcceptedTime, PreConfirmationTime, ConfirmationTime
				if firstAcceptedTime, exists := firstConfirmedTimeMap[message.ID]; exists {
					if confirmedMessageMap[message.ID] == config.Params.NodesCount {
						confirmedDelayInNetworkMap[message.ID] = messageMetadata.ConfirmationTime().Sub(firstAcceptedTime)
						delete(firstConfirmedTimeMap, message.ID)
					}
				} else {
					firstConfirmedTimeMap[message.ID] = messageMetadata.ConfirmationTime()
				}

			}))
	}
	// define header with time of dump and each node ID
	gmHeader := make([]string, 0, config.Params.NodesCount+1)
	for i := 0; i < config.Params.NodesCount; i++ {
		header := []string{fmt.Sprintf("Node %d", i)}
		gmHeader = append(gmHeader, header...)
	}
	header := []string{"ns since start"}
	gmHeader = append(gmHeader, header...)
	// dissemination results
	file, err := createFile(path.Join(config.Params.SchedulerOutputDir, "disseminatedMessages.csv"))
	if err != nil {
		panic(err)
	}
	dissemResultsWriter := csv.NewWriter(file)
	if err := dissemResultsWriter.Write(gmHeader); err != nil {
		panic(err)
	}
	file, err = createFile(path.Join(config.Params.SchedulerOutputDir, "undisseminatedMessages.csv"))
	if err != nil {
		panic(err)
	}
	undissemResultsWriter := csv.NewWriter(file)
	if err := undissemResultsWriter.Write(gmHeader); err != nil {
		panic(err)
	}

	// confirmination results
	file, err = createFile(path.Join(config.Params.SchedulerOutputDir, "fullyConfirmedMessages.csv"))
	if err != nil {
		panic(err)
	}
	confirmationResultsWriter := csv.NewWriter(file)
	if err := confirmationResultsWriter.Write(gmHeader); err != nil {
		panic(err)
	}
	file, err = createFile(path.Join(config.Params.SchedulerOutputDir, "partiallyConfirmedMessages.csv"))
	if err != nil {
		panic(err)
	}
	partialConfirmationResultsWriter := csv.NewWriter(file)
	if err := partialConfirmationResultsWriter.Write(gmHeader); err != nil {
		panic(err)
	}
	file, err = createFile(path.Join(config.Params.SchedulerOutputDir, "unconfirmedMessages.csv"))
	if err != nil {
		panic(err)
	}
	unconfirmationResultsWriter := csv.NewWriter(file)
	if err := unconfirmationResultsWriter.Write(gmHeader); err != nil {
		panic(err)
	}

	go func() {
		for {
			select {
			case <-globalMetricsTicker.C:
				dumpLocalMetrics()
				dumpGlobalMetrics(dissemResultsWriter,
					undissemResultsWriter,
					confirmationResultsWriter,
					partialConfirmationResultsWriter,
					unconfirmationResultsWriter)
			case <-shutdownGlobalMetrics:
				log.Warn("Shutting down global metrics")
				return
			}
		}
	}()
}

func dumpAcceptanceLatencyAmongNodes() {
	// accepted time latency in network
	file, err := createFile(path.Join(config.Params.GeneralOutputDir, "acceptanceTimeLatencyAmongNodes.csv"))
	if err != nil {
		panic(err)
	}

	header := []string{
		"blockID",
		"Accepted Time Diff",
	}

	writer := csv.NewWriter(file)
	if err := writer.Write(header); err != nil {
		panic(err)
	}
	writer.Flush()

	record := make([]string, len(header))

	confirmedDelayInNetworkMutex.Lock()
	defer confirmedDelayInNetworkMutex.Unlock()
	fmt.Println(len(confirmedDelayInNetworkMap))
	// Extract blockIDs from confirmedDelayInNetworkMap into a slice of integers
	var blockIDs []int
	for blkID := range confirmedDelayInNetworkMap {
		blockIDs = append(blockIDs, int(blkID))
	}

	// Sort the blockIDs in ascending order
	sort.Ints(blockIDs)

	// Iterate over sorted blockIDs and write data to CSV file
	for _, blkID := range blockIDs {
		timeDiffs := confirmedDelayInNetworkMap[multiverse.MessageID(blkID)]
		record[0] = strconv.FormatInt(int64(blkID), 10)
		record[1] = strconv.FormatInt(timeDiffs.Nanoseconds(), 10)
		if err := writer.Write(record); err != nil {
			panic(err)
		}
		writer.Flush()
	}
}

func dumpFinalData(net *network.Network) {
	file, err := createFile(path.Join(config.Params.GeneralOutputDir, "Traffic.csv"))
	if err != nil {
		panic(err)
	}
	header := []string{
		"Slot ID",
		"Blocks Count",
	}
	writer := csv.NewWriter(file)
	if err := writer.Write(header); err != nil {
		panic(err)
	}
	writer.Flush()
	record := make([]string, len(header))
	mbPeer := net.Peers[0]
	traffic := mbPeer.Node.(multiverse.NodeInterface).Tangle().Storage.MessagesCountPerSlot()

	// Extract slotIDs from traffic map into a slice of integers
	var slotIDs []int
	for slotID := range traffic {
		slotIDs = append(slotIDs, int(slotID))
	}

	// Sort the slotIDs in ascending order
	sort.Ints(slotIDs)

	// Iterate over sorted slotIDs and write data to CSV file
	for _, slotID := range slotIDs {
		blockCount := traffic[multiverse.SlotIndex(slotID)]
		record[0] = strconv.FormatInt(int64(slotID), 10)
		record[1] = strconv.FormatInt(int64(blockCount), 10)
		if err := writer.Write(record); err != nil {
			panic(err)
		}
		writer.Flush()
	}

	file, err = createFile(path.Join(config.Params.GeneralOutputDir, "BlockInformation.csv"))
	if err != nil {
		panic(err)
	}

	// Message ID,Issuance Time (unix),Confirmation Time (ns),Weight,# of Confirmed Messages,# of Issued Messages,ns since start
	header = []string{
		"Issuer Burn Policy",
		"Message ID",
		"Issuance Time Since Start (ns)",
		"Confirmation Time (ns)",
	}

	writer = csv.NewWriter(file)
	if err := writer.Write(header); err != nil {
		panic(err)
	}
	writer.Flush()
	record = make([]string, len(header))
	for messageID := range disseminatedMessages {
		message := disseminatedMessages[messageID]
		messageMetadata := disseminatedMessageMetadata[messageID]
		record[0] = strconv.FormatInt(int64(config.Params.BurnPolicies[int(message.Issuer)]), 10)
		record[1] = strconv.FormatInt(int64(message.ID), 10)
		record[2] = strconv.FormatInt(message.IssuanceTime.Sub(simulationStartTime).Nanoseconds(), 10)
		// TODO (confirmation): monitor PreAcceptedTime, AcceptedTime, PreConfirmationTime, ConfirmationTime
		t := int64(messageMetadata.ConfirmationTime().Sub(message.IssuanceTime))
		if t < 0 {
			t = 0
		}
		record[3] = strconv.FormatInt(t, 10)
		if err := writer.Write(record); err != nil {
			panic(err)
		}
		delete(storedMessages, messageID)
		writer.Flush()
	}

	for messageID := range storedMessages {
		message := storedMessages[messageID]
		record[0] = strconv.FormatInt(int64(config.Params.BurnPolicies[int(message.Issuer)]), 10)
		record[1] = strconv.FormatInt(int64(message.ID), 10)
		record[2] = strconv.FormatInt(message.IssuanceTime.Sub(simulationStartTime).Nanoseconds(), 10)
		record[3] = strconv.FormatInt(0, 10)
		if err := writer.Write(record); err != nil {
			panic(err)
		}
		writer.Flush()
	}

	file, err = createFile(path.Join(config.Params.SchedulerOutputDir, "DisseminationLatency.csv"))
	if err != nil {
		panic(err)
	}
	header = []string{
		"Issuer ID",
		"Dissemination Time",
		"Dissemination Latency",
	}
	writer = csv.NewWriter(file)
	if err := writer.Write(header); err != nil {
		panic(err)
	}
	writer.Flush()
	record = make([]string, len(header))
	for messageID := range disseminatedMessages {
		message := disseminatedMessages[messageID]
		messageMetadata := disseminatedMessageMetadata[messageID]
		record[0] = strconv.FormatInt(int64(message.Issuer), 10)
		record[1] = strconv.FormatInt(int64(messageMetadata.ArrivalTime().Sub(simulationStartTime).Nanoseconds()), 10)
		record[2] = strconv.FormatInt(int64(messageMetadata.ArrivalTime().Sub(message.IssuanceTime).Nanoseconds()), 10)
		if err := writer.Write(record); err != nil {
			panic(err)
		}
		writer.Flush()
	}
	file, err = createFile(path.Join(config.Params.GeneralOutputDir, "ConfirmationLatency.csv"))
	if err != nil {
		panic(err)
	}
	header = []string{
		"Issuer ID",
		"Confirmation Time",
		"Confirmation Latency",
	}
	writer = csv.NewWriter(file)
	if err := writer.Write(header); err != nil {
		panic(err)
	}
	writer.Flush()
	for messageID := range fullyConfirmedMessages {
		message := fullyConfirmedMessages[messageID]
		messageMetadata := fullyConfirmedMessageMetadata[messageID]
		record[0] = strconv.FormatInt(int64(message.Issuer), 10)
		// TODO (confirmation): monitor PreAcceptedTime, AcceptedTime, PreConfirmationTime, ConfirmationTime
		record[1] = strconv.FormatInt(int64(messageMetadata.ConfirmationTime().Sub(simulationStartTime).Nanoseconds()), 10)
		record[2] = strconv.FormatInt(int64(messageMetadata.ConfirmationTime().Sub(message.IssuanceTime).Nanoseconds()), 10)
		if err := writer.Write(record); err != nil {
			panic(err)
		}
		writer.Flush()
	}
	file, err = createFile(path.Join(config.Params.GeneralOutputDir, "localMetrics.csv"))
	if err != nil {
		panic(err)
	}
	writer = csv.NewWriter(file)
	for name := range localMetrics {
		if err := writer.Write([]string{name}); err != nil {
			panic(err)
		}
	}
	writer.Flush()
}

func dumpFinalRecorder() {
	fileName := fmt.Sprint("nd-", config.Params.ScriptStartTimeStr, ".csv")
	file, err := createFile(path.Join(config.Params.GeneralOutputDir, fileName))
	if err != nil {
		panic(err)
	}

	writer := csv.NewWriter(file)
	if err := writer.Write(ndHeader); err != nil {
		panic(err)
	}

	for i := 0; i < config.Params.NodesCount; i++ {
		record := []string{
			strconv.FormatInt(int64(i), 10),
			strconv.FormatBool(network.IsAdversary(int(i))),
			strconv.FormatInt(int64(nodeCounters[i].Get("minConfirmedAccumulatedWeight")), 10),
			strconv.FormatInt(int64(nodeCounters[i].Get("unconfirmationCount")), 10),
		}
		writeLine(writer, record)

		// Flush the writers, or the data will be truncated for high node count
		writer.Flush()
	}
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

func dumpConfig(filePath string) {
	bytes, err := json.MarshalIndent(config.Params, "", " ")
	if err != nil {
		log.Error(err)
	}
	if err := ioutil.WriteFile(filePath, bytes, 0644); err != nil {
		log.Error(err)
	}
}

func dumpNetworkConfig(net *network.Network) {
	file, err := createFile(path.Join(config.Params.SchedulerOutputDir, "networkConfig.csv"))
	if err != nil {
		panic(err)
	}
	ncHeader := []string{"Peer ID", "Neighbor ID", "Network Delay (ns)", "Packet Loss (%)"}
	ncWriter := csv.NewWriter(file)
	if err := ncWriter.Write(ncHeader); err != nil {
		panic(err)
	}
	file, err = createFile(path.Join(config.Params.SchedulerOutputDir, "weights.csv"))
	if err != nil {
		panic(err)
	}
	wHeader := []string{"Peer ID", "Weight"}
	wWriter := csv.NewWriter(file)
	if err := wWriter.Write(wHeader); err != nil {
		panic(err)
	}
	for _, peer := range net.Peers {
		for neighbor, connection := range peer.Neighbors {
			record := []string{
				strconv.FormatInt(int64(peer.ID), 10),
				strconv.FormatInt(int64(neighbor), 10),
				strconv.FormatInt(connection.NetworkDelay().Nanoseconds(), 10),
				strconv.FormatInt(int64(connection.PacketLoss()*100), 10),
			}
			writeLine(ncWriter, record)
		}
		writeLine(wWriter, []string{
			strconv.FormatInt(int64(peer.ID), 10),
			strconv.FormatInt(int64(net.WeightDistribution.Weight(peer.ID)), 10),
		})
		// Flush the writers, or the data will be truncated for high node count
		flushWriters([]*csv.Writer{ncWriter, wWriter})
	}
}

func monitorNetworkState(testNetwork *network.Network) (resultsWriters []*csv.Writer) {
	adversaryNodesCount := len(network.AdversaryNodeIDToGroupIDMap)
	// honestNodesCount := config.Params.NodesCount - adversaryNodesCount

	allColors := []multiverse.Color{multiverse.UndefinedColor, multiverse.Red, multiverse.Green, multiverse.Blue}

	colorCounters.CreateCounter("opinions", allColors, []int64{int64(config.Params.NodesCount), 0, 0, 0})
	colorCounters.CreateCounter("confirmedNodes", allColors, []int64{0, 0, 0, 0})
	colorCounters.CreateCounter("opinionsWeights", allColors, []int64{0, 0, 0, 0})
	colorCounters.CreateCounter("likeAccumulatedWeight", allColors, []int64{0, 0, 0, 0})
	colorCounters.CreateCounter("processedMessages", allColors, []int64{0, 0, 0, 0})
	colorCounters.CreateCounter("requestedMissingMessages", allColors, []int64{0, 0, 0, 0})
	colorCounters.CreateCounter("tipPoolSizes", allColors, []int64{0, 0, 0, 0})
	for _, peer := range testNetwork.Peers {
		peerID := peer.ID
		tipCounterName := fmt.Sprint("tipPoolSizes-", peerID)
		processedCounterName := fmt.Sprint("processedMessages-", peerID)
		colorCounters.CreateCounter(tipCounterName, allColors, []int64{0, 0, 0, 0})
		colorCounters.CreateCounter(processedCounterName, allColors, []int64{0, 0, 0, 0})
	}
	colorCounters.CreateCounter("colorUnconfirmed", allColors[1:], []int64{0, 0, 0})
	colorCounters.CreateCounter("confirmedAccumulatedWeight", allColors[1:], []int64{0, 0, 0})
	colorCounters.CreateCounter("unconfirmedAccumulatedWeight", allColors[1:], []int64{0, 0, 0})

	adversaryCounters.CreateCounter("likeAccumulatedWeight", allColors[1:], []int64{0, 0, 0})
	adversaryCounters.CreateCounter("opinions", allColors, []int64{int64(adversaryNodesCount), 0, 0, 0})
	adversaryCounters.CreateCounter("confirmedNodes", allColors, []int64{0, 0, 0, 0})
	adversaryCounters.CreateCounter("confirmedAccumulatedWeight", allColors, []int64{0, 0, 0, 0})

	// Initialize the minConfirmedWeight to be the max value (i.e., the total weight)
	for i := 0; i < config.Params.NodesCount; i++ {
		nodeCounters = append(nodeCounters, *simulation.NewAtomicCounters[string, int64]())
		nodeCounters[i].CreateCounter("minConfirmedAccumulatedWeight", int64(config.Params.NodesTotalWeight))
		nodeCounters[i].CreateCounter("unconfirmationCount", 0)
	}

	atomicCounters.CreateCounter("flips", 0)
	atomicCounters.CreateCounter("honestFlips", 0)
	atomicCounters.CreateCounter("tps", 0)
	atomicCounters.CreateCounter("relevantValidators", 0)
	atomicCounters.CreateCounter("issuedMessages", 0)
	for _, peer := range testNetwork.Peers {
		peerID := peer.ID
		issuedCounterName := fmt.Sprint("issuedMessages-", peerID)
		atomicCounters.CreateCounter(issuedCounterName, 0)
	}

	mostLikedColor = multiverse.UndefinedColor
	honestOnlyMostLikedColor = multiverse.UndefinedColor

	// Dump the network information
	dumpNetworkConfig(testNetwork)

	// Dump the info about adversary nodes
	adResultsWriter := createWriter(fmt.Sprintf("ad-%s.csv", config.Params.ScriptStartTimeStr), adHeader, &resultsWriters)
	dumpResultsAD(adResultsWriter, testNetwork)

	// Dump the double spending result
	// dsResultsWriter := createWriter(fmt.Sprintf("ds-%s.csv", config.Params.ScriptStartTimeStr), dsHeader, &resultsWriters)

	// Dump the tip pool and processed message (throughput) results
	// tpResultsWriter := createWriter(fmt.Sprintf("tp-%s.csv", config.Params.ScriptStartTimeStr), tpHeader, &resultsWriters)

	// Dump the requested missing message result
	// mmResultsWriter := createWriter(fmt.Sprintf("mm-%s.csv", config.Params.ScriptStartTimeStr), mmHeader, &resultsWriters)

	tpAllHeader := make([]string, 0, config.Params.NodesCount+1)

	for i := 0; i < config.Params.NodesCount; i++ {
		header := []string{fmt.Sprintf("Node %d", i)}
		tpAllHeader = append(tpAllHeader, header...)
	}
	header := []string{fmt.Sprintf("ns since start")}
	tpAllHeader = append(tpAllHeader, header...)

	// Dump the tip pool and processed message (throughput) results
	// tpAllResultsWriter := createWriter(fmt.Sprintf("all-tp-%s.csv", config.Params.ScriptStartTimeStr), tpAllHeader, &resultsWriters)

	// Dump the info about how many nodes have confirmed and liked a certain color
	// ccResultsWriter := createWriter(fmt.Sprintf("cc-%s.csv", config.Params.ScriptStartTimeStr), ccHeader, &resultsWriters)

	// Define the file name of the ww results
	wwResultsWriter := createWriter(fmt.Sprintf("ww-%s.csv", config.Params.ScriptStartTimeStr), wwHeader, &resultsWriters)

	// Dump the Witness Weight
	wwPeer := testNetwork.Peers[config.Params.MonitoredWitnessWeightPeer]
	previousWitnessWeight := uint64(config.Params.NodesTotalWeight)
	wwPeer.Node.(multiverse.NodeInterface).Tangle().ApprovalManager.Events.MessageWitnessWeightUpdated.Attach(
		events.NewClosure(func(message *multiverse.Message, weight uint64) {
			if uint64(previousWitnessWeight) == weight {
				return
			}
			previousWitnessWeight = weight
			record := []string{
				strconv.FormatUint(weight, 10),
				strconv.FormatInt(time.Since(message.IssuanceTime).Nanoseconds(), 10),
			}
			csvMutex.Lock()
			if err := wwResultsWriter.Write(record); err != nil {
				log.Fatal("error writing record to csv:", err)
			}

			if err := wwResultsWriter.Error(); err != nil {
				log.Fatal(err)
			}
			csvMutex.Unlock()
		}))

	for _, id := range config.Params.MonitoredAWPeers {
		awPeer := testNetwork.Peers[id]
		if typeutils.IsInterfaceNil(awPeer) {
			panic(fmt.Sprintf("unknowm peer with id %d", id))
		}
		// Define the file name of the aw results
		// awResultsWriter := createWriter(fmt.Sprintf("aw%d-%s.csv", id, config.Params.ScriptStartTimeStr), awHeader, &resultsWriters)

		// awPeer.Node.(multiverse.NodeInterface).Tangle().ApprovalManager.Events.MessageConfirmed.Attach(
		// 	events.NewClosure(func(message *multiverse.Message, messageMetadata *multiverse.MessageMetadata, weight uint64, messageIDCounter int64) {
		// 		// if BurnPolicies[x] == 0: spammer node only
		// 		// if BurnPolicies[x] == 1: normal node only
		// 		if config.Params.BurnPolicies[int(message.Issuer)] == 0 {
		// 			confirmedMessageMutex.Lock()
		// 			confirmedMessageCounter[awPeer.ID]++
		// 			confirmedMessageMutex.Unlock()
		// 			confirmedMessageMutex.RLock()
		// 			record := []string{
		// 				strconv.FormatInt(int64(message.ID), 10),
		// 				strconv.FormatInt(message.IssuanceTime.Unix(), 10),
		// 				strconv.FormatInt(int64(messageMetadata.ConfirmationTime().Sub(message.IssuanceTime)), 10),
		// 				strconv.FormatUint(weight, 10),
		// 				strconv.FormatInt(confirmedMessageCounter[awPeer.ID], 10),
		// 				strconv.FormatInt(messageIDCounter, 10),
		// 				strconv.FormatInt(time.Since(simulationStartTime).Nanoseconds(), 10),
		// 			}
		// 			confirmedMessageMutex.RUnlock()

		// 			csvMutex.Lock()
		// 			if err := awResultsWriter.Write(record); err != nil {
		// 				log.Fatal("error writing record to csv:", err)
		// 			}

		// 			if err := awResultsWriter.Error(); err != nil {
		// 				log.Fatal(err)
		// 			}
		// 			awResultsWriter.Flush()
		// 			csvMutex.Unlock()
		// 		}
		// 	}))
	}

	for _, peer := range testNetwork.Peers {
		peerID := peer.ID

		peer.Node.(multiverse.NodeInterface).Tangle().OpinionManager.Events().OpinionChanged.Attach(events.NewClosure(func(oldOpinion multiverse.Color, newOpinion multiverse.Color, weight int64) {
			colorCounters.Add("opinions", -1, oldOpinion)
			colorCounters.Add("opinions", 1, newOpinion)

			colorCounters.Add("likeAccumulatedWeight", -weight, oldOpinion)
			colorCounters.Add("likeAccumulatedWeight", weight, newOpinion)

			r, g, b := getLikesPerRGB(colorCounters, "opinions")
			if mostLikedColorChanged(r, g, b, &mostLikedColor) {
				atomicCounters.Add("flips", 1)
			}
			if network.IsAdversary(int(peerID)) {
				adversaryCounters.Add("likeAccumulatedWeight", -weight, oldOpinion)
				adversaryCounters.Add("likeAccumulatedWeight", weight, newOpinion)
				adversaryCounters.Add("opinions", -1, oldOpinion)
				adversaryCounters.Add("opinions", 1, newOpinion)
			}

			ar, ag, ab := getLikesPerRGB(adversaryCounters, "opinions")
			// honest nodes likes status only, flips
			if mostLikedColorChanged(r-ar, g-ag, b-ab, &honestOnlyMostLikedColor) {
				atomicCounters.Add("honestFlips", 1)
			}
		}))
		peer.Node.(multiverse.NodeInterface).Tangle().OpinionManager.Events().ColorConfirmed.Attach(events.NewClosure(func(confirmedColor multiverse.Color, weight int64) {
			colorCounters.Add("confirmedNodes", 1, confirmedColor)
			colorCounters.Add("confirmedAccumulatedWeight", weight, confirmedColor)
			if network.IsAdversary(int(peerID)) {
				adversaryCounters.Add("confirmedNodes", 1, confirmedColor)
				adversaryCounters.Add("confirmedAccumulatedWeight", weight, confirmedColor)
			}
		}))

		peer.Node.(multiverse.NodeInterface).Tangle().OpinionManager.Events().ColorUnconfirmed.Attach(events.NewClosure(func(unconfirmedColor multiverse.Color, unconfirmedSupport int64, weight int64) {
			colorCounters.Add("colorUnconfirmed", 1, unconfirmedColor)
			colorCounters.Add("confirmedNodes", -1, unconfirmedColor)

			colorCounters.Add("unconfirmedAccumulatedWeight", weight, unconfirmedColor)
			colorCounters.Add("confirmedAccumulatedWeight", -weight, unconfirmedColor)

			// When the color is unconfirmed, the min confirmed accumulated weight should be reset
			nodeCounters[int(peerID)].Set("minConfirmedAccumulatedWeight", int64(config.Params.NodesTotalWeight))

			// Accumulate the unconfirmed count for each node
			nodeCounters[int(peerID)].Add("unconfirmationCount", 1)
		}))

		// We want to know how deep the support for our once confirmed color could fall
		peer.Node.(multiverse.NodeInterface).Tangle().OpinionManager.Events().MinConfirmedWeightUpdated.Attach(events.NewClosure(func(opinion multiverse.Color, confirmedWeight int64) {
			if nodeCounters[int(peerID)].Get("minConfirmedAccumulatedWeight") > confirmedWeight {
				nodeCounters[int(peerID)].Set("minConfirmedAccumulatedWeight", confirmedWeight)
			}
		}))
	}

	// Here we only monitor the opinion weight of node w/ the highest weight
	dsPeer := testNetwork.Peers[0]
	dsPeer.Node.(multiverse.NodeInterface).Tangle().OpinionManager.Events().ApprovalWeightUpdated.Attach(events.NewClosure(func(opinion multiverse.Color, deltaWeight int64) {
		colorCounters.Add("opinionsWeights", deltaWeight, opinion)
	}))

	// Here we only monitor the tip pool size of node w/ the highest weight
	peer := testNetwork.Peers[0]
	peer.Node.(multiverse.NodeInterface).Tangle().TipManager.Events.MessageProcessed.Attach(events.NewClosure(
		func(opinion multiverse.Color, tipPoolSize int, processedMessages uint64, issuedMessages int64) {
			colorCounters.Set("tipPoolSizes", int64(tipPoolSize), opinion)
			colorCounters.Set("processedMessages", int64(processedMessages), opinion)

			atomicCounters.Set("issuedMessages", issuedMessages)
		}))
	peer.Node.(multiverse.NodeInterface).Tangle().Requester.Events.Request.Attach(events.NewClosure(
		func(messageID multiverse.MessageID) {
			colorCounters.Add("requestedMissingMessages", int64(1), multiverse.UndefinedColor)
		}))

	for _, peer := range testNetwork.Peers {
		peerID := peer.ID
		tipCounterName := fmt.Sprint("tipPoolSizes-", peerID)
		processedCounterName := fmt.Sprint("processedMessages-", peerID)
		issuedCounterName := fmt.Sprint("issuedMessages-", peerID)
		peer.Node.(multiverse.NodeInterface).Tangle().TipManager.Events.MessageProcessed.Attach(events.NewClosure(
			func(opinion multiverse.Color, tipPoolSize int, processedMessages uint64, issuedMessages int64) {
				colorCounters.Set(tipCounterName, int64(tipPoolSize), opinion)
				colorCounters.Set(processedCounterName, int64(processedMessages), opinion)
				atomicCounters.Set(issuedCounterName, issuedMessages)
			}))
	}

	// TODO: reopen global metrics
	// go func() {
	// 	for {
	// 		select {
	// 		case <-globalMetricsTicker.C:
	// 			dumpRecords(dsResultsWriter, tpResultsWriter, ccResultsWriter, adResultsWriter, tpAllResultsWriter, mmResultsWriter, honestNodesCount, adversaryNodesCount)
	// 		case <-shutdownGlobalMetrics:
	// 			log.Warn("Shutting down global metrics")
	// 			return
	// 		}
	// 	}
	// }()

	return
}

func dumpRecords(dsResultsWriter *csv.Writer, tpResultsWriter *csv.Writer, ccResultsWriter *csv.Writer, adResultsWriter *csv.Writer, tpAllResultsWriter *csv.Writer, mmResultsWriter *csv.Writer, honestNodesCount int, adversaryNodesCount int) {
	simulationWg.Add(1)
	defer simulationWg.Done()

	sinceIssuance := "0"
	if !dsIssuanceTime.IsZero() {
		sinceIssuance = strconv.FormatInt(time.Since(dsIssuanceTime).Nanoseconds(), 10)
	}

	dumpResultDS(dsResultsWriter, sinceIssuance)
	dumpResultsTP(tpResultsWriter)
	dumpResultsTPAll(tpAllResultsWriter)
	dumpResultsCC(ccResultsWriter, sinceIssuance)
	dumpResultsMM(mmResultsWriter)

	// determines whether consensus has been reached and simulation is over

	r, g, b := getLikesPerRGB(colorCounters, "confirmedNodes")
	aR, aG, aB := getLikesPerRGB(adversaryCounters, "confirmedNodes")
	hR, hG, hB := r-aR, g-aG, b-aB
	if Max(Max(hB, hR), hG) >= int64(config.Params.SimulationStopThreshold*float64(honestNodesCount)) {
		shutdownSignal <- types.Void
	}
	atomicCounters.Set("tps", 0)
}

func dumpResultDS(dsResultsWriter *csv.Writer, sinceIssuance string) {
	// Dump the double spending results
	record := []string{
		strconv.FormatInt(colorCounters.Get("opinionsWeights", multiverse.UndefinedColor), 10),
		strconv.FormatInt(colorCounters.Get("opinionsWeights", multiverse.Blue), 10),
		strconv.FormatInt(colorCounters.Get("opinionsWeights", multiverse.Red), 10),
		strconv.FormatInt(colorCounters.Get("opinionsWeights", multiverse.Green), 10),
		strconv.FormatInt(time.Since(simulationStartTime).Nanoseconds(), 10),
		sinceIssuance,
	}

	writeLine(dsResultsWriter, record)

	// Flush the writers, or the data will be truncated sometimes if the buffer is full
	dsResultsWriter.Flush()
}

func dumpResultsTP(tpResultsWriter *csv.Writer) {
	// Dump the tip pool sizes
	record := []string{
		strconv.FormatInt(colorCounters.Get("tipPoolSizes", multiverse.UndefinedColor), 10),
		strconv.FormatInt(colorCounters.Get("tipPoolSizes", multiverse.Blue), 10),
		strconv.FormatInt(colorCounters.Get("tipPoolSizes", multiverse.Red), 10),
		strconv.FormatInt(colorCounters.Get("tipPoolSizes", multiverse.Green), 10),
		strconv.FormatInt(colorCounters.Get("processedMessages", multiverse.UndefinedColor), 10),
		strconv.FormatInt(colorCounters.Get("processedMessages", multiverse.Blue), 10),
		strconv.FormatInt(colorCounters.Get("processedMessages", multiverse.Red), 10),
		strconv.FormatInt(colorCounters.Get("processedMessages", multiverse.Green), 10),
		strconv.FormatInt(atomicCounters.Get("issuedMessages"), 10),
		strconv.FormatInt(time.Since(simulationStartTime).Nanoseconds(), 10),
	}

	writeLine(tpResultsWriter, record)

	// Flush the writers, or the data will be truncated sometimes if the buffer is full
	tpResultsWriter.Flush()
}

func dumpResultsTPAll(tpAllResultsWriter *csv.Writer) {
	record := make([]string, config.Params.NodesCount+1)
	i := 0
	for peerID := 0; peerID < config.Params.NodesCount; peerID++ {
		tipCounterName := fmt.Sprint("tipPoolSizes-", peerID)
		// processedCounterName := fmt.Sprint("processedMessages-", peerID)
		// issuedCounterName := fmt.Sprint("issuedMessages-", peerID)
		record[i+0] = strconv.FormatInt(colorCounters.Get(tipCounterName, multiverse.UndefinedColor), 10)
		// record[i+1] = strconv.FormatInt(colorCounters.Get(tipCounterName, multiverse.Blue), 10)
		// record[i+2] = strconv.FormatInt(colorCounters.Get(tipCounterName, multiverse.Red), 10)
		// record[i+3] = strconv.FormatInt(colorCounters.Get(tipCounterName, multiverse.Green), 10)
		// record[i+4] = strconv.FormatInt(colorCounters.Get(processedCounterName, multiverse.UndefinedColor), 10)
		// record[i+5] = strconv.FormatInt(colorCounters.Get(processedCounterName, multiverse.Blue), 10)
		// record[i+6] = strconv.FormatInt(colorCounters.Get(processedCounterName, multiverse.Red), 10)
		// record[i+7] = strconv.FormatInt(colorCounters.Get(processedCounterName, multiverse.Green), 10)
		// record[i+8] = strconv.FormatInt(atomicCounters.Get(issuedCounterName), 10)
		// record[i+9] = strconv.FormatInt(time.Since(simulationStartTime).Nanoseconds(), 10)
		i = i + 1
	}
	record[i] = strconv.FormatInt(time.Since(simulationStartTime).Nanoseconds(), 10)

	writeLine(tpAllResultsWriter, record)

	// Flush the writers, or the data will be truncated sometimes if the buffer is full
	tpAllResultsWriter.Flush()
}

func dumpResultsMM(mmResultsWriter *csv.Writer) {
	// Dump the opinion and confirmation counters
	record := []string{
		strconv.FormatInt(colorCounters.Get("requestedMissingMessages", multiverse.UndefinedColor), 10),
		strconv.FormatInt(time.Since(simulationStartTime).Nanoseconds(), 10),
	}

	writeLine(mmResultsWriter, record)

	// Flush the mm writer, or the data will be truncated sometimes if the buffer is full
	mmResultsWriter.Flush()
}

func dumpResultsCC(ccResultsWriter *csv.Writer, sinceIssuance string) {
	// Dump the opinion and confirmation counters
	record := []string{
		strconv.FormatInt(colorCounters.Get("confirmedNodes", multiverse.Blue), 10),
		strconv.FormatInt(colorCounters.Get("confirmedNodes", multiverse.Red), 10),
		strconv.FormatInt(colorCounters.Get("confirmedNodes", multiverse.Green), 10),
		strconv.FormatInt(adversaryCounters.Get("confirmedNodes", multiverse.Blue), 10),
		strconv.FormatInt(adversaryCounters.Get("confirmedNodes", multiverse.Red), 10),
		strconv.FormatInt(adversaryCounters.Get("confirmedNodes", multiverse.Green), 10),
		strconv.FormatInt(colorCounters.Get("confirmedAccumulatedWeight", multiverse.Blue), 10),
		strconv.FormatInt(colorCounters.Get("confirmedAccumulatedWeight", multiverse.Red), 10),
		strconv.FormatInt(colorCounters.Get("confirmedAccumulatedWeight", multiverse.Green), 10),
		strconv.FormatInt(adversaryCounters.Get("confirmedAccumulatedWeight", multiverse.Blue), 10),
		strconv.FormatInt(adversaryCounters.Get("confirmedAccumulatedWeight", multiverse.Red), 10),
		strconv.FormatInt(adversaryCounters.Get("confirmedAccumulatedWeight", multiverse.Green), 10),
		strconv.FormatInt(colorCounters.Get("opinions", multiverse.Blue), 10),
		strconv.FormatInt(colorCounters.Get("opinions", multiverse.Red), 10),
		strconv.FormatInt(colorCounters.Get("opinions", multiverse.Green), 10),
		strconv.FormatInt(colorCounters.Get("likeAccumulatedWeight", multiverse.Blue), 10),
		strconv.FormatInt(colorCounters.Get("likeAccumulatedWeight", multiverse.Red), 10),
		strconv.FormatInt(colorCounters.Get("likeAccumulatedWeight", multiverse.Green), 10),
		strconv.FormatInt(adversaryCounters.Get("likeAccumulatedWeight", multiverse.Blue), 10),
		strconv.FormatInt(adversaryCounters.Get("likeAccumulatedWeight", multiverse.Red), 10),
		strconv.FormatInt(adversaryCounters.Get("likeAccumulatedWeight", multiverse.Green), 10),
		strconv.FormatInt(colorCounters.Get("colorUnconfirmed", multiverse.Blue), 10),
		strconv.FormatInt(colorCounters.Get("colorUnconfirmed", multiverse.Red), 10),
		strconv.FormatInt(colorCounters.Get("colorUnconfirmed", multiverse.Green), 10),
		strconv.FormatInt(colorCounters.Get("unconfirmedAccumulatedWeight", multiverse.Blue), 10),
		strconv.FormatInt(colorCounters.Get("unconfirmedAccumulatedWeight", multiverse.Red), 10),
		strconv.FormatInt(colorCounters.Get("unconfirmedAccumulatedWeight", multiverse.Green), 10),
		strconv.FormatInt(atomicCounters.Get("flips"), 10),
		strconv.FormatInt(atomicCounters.Get("honestFlips"), 10),
		strconv.FormatInt(time.Since(simulationStartTime).Nanoseconds(), 10),
		sinceIssuance,
	}

	writeLine(ccResultsWriter, record)

	// Flush the cc writer, or the data will be truncated sometimes if the buffer is full
	ccResultsWriter.Flush()
}

func dumpResultsAD(adResultsWriter *csv.Writer, net *network.Network) {
	adHeader = []string{"AdversaryGroupID", "Strategy", "AdversaryCount", "q"}
	for groupID, group := range net.AdversaryGroups {
		record := []string{
			strconv.FormatInt(int64(groupID), 10),
			network.AdversaryTypeToString(group.AdversaryType),
			strconv.FormatInt(int64(len(group.NodeIDs)), 10),
			strconv.FormatFloat(float64(group.GroupMana)/float64(config.Params.NodesTotalWeight), 'f', 6, 64),
			strconv.FormatInt(time.Since(simulationStartTime).Nanoseconds(), 10),
		}
		writeLine(adResultsWriter, record)
	}
	// Flush the cc writer, or the data will be truncated sometimes if the buffer is full
	adResultsWriter.Flush()
}

func SimulateDoubleSpent(testNetwork *network.Network) {
	time.Sleep(time.Duration(config.Params.DoubleSpendDelay*config.Params.SlowdownFactor) * time.Second)
	// Here we simulate the double spending
	dsIssuanceTime = time.Now()

	switch config.Params.SimulationMode {
	case "Accidental":
		for i, node := range network.GetAccidentalIssuers(testNetwork) {
			color := multiverse.ColorFromInt(i + 1)
			go sendMessage(node, color)
			log.Infof("Peer %d sent double spend msg: %v", node.ID, color)
		}
	case "Adversary":
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
	}
}

func writeLine(writer *csv.Writer, record []string) {
	if err := writer.Write(record); err != nil {
		log.Fatal("error writing record to csv:", err)
	}

	if err := writer.Error(); err != nil {
		log.Fatal(err)
	}
}

func createWriter(fileName string, header []string, resultsWriters *[]*csv.Writer) *csv.Writer {
	file, err := createFile(path.Join(config.Params.GeneralOutputDir, fileName))
	if err != nil {
		panic(err)
	}
	resultsWriter := csv.NewWriter(file)

	// Check the result writers
	if resultsWriters != nil {
		*resultsWriters = append(*resultsWriters, resultsWriter)
	}
	// Write the headers
	if err := resultsWriter.Write(header); err != nil {
		panic(err)
	}
	return resultsWriter
}

// Max returns the larger of x or y.
func Max(x, y int64) int64 {
	if x < y {
		return y
	}
	return x
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

func getLikesPerRGB(counter *simulation.ColorCounters, flag string) (int64, int64, int64) {
	return counter.Get(flag, multiverse.Red), counter.Get(flag, multiverse.Green), counter.Get(flag, multiverse.Blue)
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

// createFile creates nested directories first if they're not existed.
func createFile(p string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(p), 0770); err != nil {
		return nil, err
	}
	return os.Create(p)
}
