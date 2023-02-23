package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/iotaledger/multivers-simulation/singlenodeattacks"

	"github.com/iotaledger/multivers-simulation/adversary"
	"github.com/iotaledger/multivers-simulation/simulation"

	"github.com/iotaledger/hive.go/types"

	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/hive.go/typeutils"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/logger"
	"github.com/iotaledger/multivers-simulation/multiverse"
	"github.com/iotaledger/multivers-simulation/network"
)

var (
	log       = logger.New("Simulation")
	Simulator *simulation.Simulator
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
	dumpingTicker         = time.NewTicker(time.Duration(config.SlowdownFactor*config.ConsensusMonitorTick) * time.Millisecond)
	simulationWg          = sync.WaitGroup{}
	maxSimulationDuration = time.Minute
	shutdownSignal        = make(chan types.Empty)

	// global declarations
	dsIssuanceTime           time.Time
	mostLikedColor           multiverse.Color
	honestOnlyMostLikedColor multiverse.Color
	simulationStartTime      time.Time

	// counters

	confirmedMessageCounter = make(map[network.PeerID]int64)
	confirmedMessageMutex   sync.RWMutex

	// simulation start time string in the result file name
	simulationStartTimeStr string
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

	Simulator = simulation.NewSimulator()
	Simulator.Setup(testNetwork)
	resultsWriters := monitorNetworkState(testNetwork)
	defer flushWriters(resultsWriters)

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
	case <-time.After(time.Duration(config.SlowdownFactor) * maxSimulationDuration):
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
	pace := time.Duration((float64(time.Second) * float64(config.SlowdownFactor)) / float64(config.SchedulingRate))
	ticker := time.NewTicker(pace)
	for {
		select {
		case networkMessage := <-peer.Socket:
			peer.Node.HandleNetworkMessage(networkMessage)
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
		dsIssuanceTime = time.Now()
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
		Simulator.GlobalCounters.Add("relevantValidators", 1)

		// peer.AdversarySpeedup=1 for honest nodes and can have different values from adversary nodes
		band := peer.AdversarySpeedup * weightOfPeer * float64(config.IssuingRate) / nodeTotalWeight
		// fmt.Println(peer.AdversarySpeedup, weightOfPeer, config.IssuingRate, nodeTotalWeight)
		//fmt.Printf("speedup %f band %f\n", peer.AdversarySpeedup, band)
		go issueMessages(peer, band)
	}
}

func issueMessages(peer *network.Peer, band float64) {
	pace := time.Duration(float64(time.Second) * float64(config.SlowdownFactor) / band)

	log.Debug("Starting security worker for Peer ID: ", peer.ID, " Pace: ", pace)
	if pace == time.Duration(0) {
		log.Warn("Peer ID: ", peer.ID, " has 0 pace!")
		return
	}
	ticker := time.NewTicker(pace)

	for range ticker.C {
		if config.IMIF == "poisson" {
			pace = time.Duration(float64(time.Second) * float64(config.SlowdownFactor) * rand.ExpFloat64() / band)
			if pace > 0 {
				ticker.Reset(pace)
			}
		}
		sendMessage(peer)
	}
}

func sendMessage(peer *network.Peer, optionalColor ...multiverse.Color) {
	Simulator.GlobalCounters.Add("tps", 1)

	if len(optionalColor) >= 1 {
		peer.Node.(multiverse.NodeInterface).IssuePayload(optionalColor[0])
	}

	peer.Node.(multiverse.NodeInterface).IssuePayload(multiverse.UndefinedColor)
}

func shutdownSimulation() {
	dumpingTicker.Stop()
	dumpFinalRecorder()
	simulationWg.Wait()
}

func dumpFinalRecorder() {
	fileName := fmt.Sprint("nd-", simulationStartTimeStr, ".csv")
	file, err := os.Create(path.Join(config.ResultDir, fileName))
	if err != nil {
		panic(err)
	}

	writer := csv.NewWriter(file)
	if err := writer.Write(ndHeader); err != nil {
		panic(err)
	}

	for i := 0; i < config.NodesCount; i++ {
		record := []string{
			strconv.FormatInt(int64(i), 10),
			strconv.FormatBool(network.IsAdversary(int(i))),
			strconv.FormatInt(Simulator.PeerCounters.Get("minConfirmedAccumulatedWeight", network.PeerID(i)), 10),
			strconv.FormatInt(Simulator.PeerCounters.Get("unconfirmationCount", network.PeerID(i)), 10),
		}
		writeLine(writer, record)

		// Flush the writers, or the data will be truncated for high node count
		writer.Flush()
	}
}

func flushWriters(writers []*csv.Writer) {
	for _, writer := range writers {
		writer.Flush()
		err := writer.Error()
		if err != nil {
			log.Error(err)
		}
	}
}

func dumpConfig(fileName string) {
	type Configuration struct {
		NodesCount, NodesTotalWeight, ParentsCount, SchedulingRate, IssuingRate, ConsensusMonitorTick, RelevantValidatorWeight, MinDelay, MaxDelay, SlowdownFactor, DoubleSpendDelay, NeighbourCountWS int
		ZipfParameter, WeakTipsRatio, PacketLoss, DeltaURTS, SimulationStopThreshold, RandomnessWS                                                                                                     float64
		ConfirmationThreshold, TSA, ResultDir, IMIF, SimulationTarget, SimulationMode, BurnPolicyNames                                                                                                 string
		AdversaryDelays, AdversaryTypes, AdversaryNodeCounts                                                                                                                                           []int
		AdversarySpeedup, AdversaryMana                                                                                                                                                                []float64
		AdversaryInitColor, AccidentalMana                                                                                                                                                             []string
		AdversaryPeeringAll                                                                                                                                                                            bool
	}
	data := Configuration{
		NodesCount:              config.NodesCount,
		NodesTotalWeight:        config.NodesTotalWeight,
		ZipfParameter:           config.ZipfParameter,
		ConfirmationThreshold:   fmt.Sprintf("%.2f-%v", config.ConfirmationThreshold, config.ConfirmationThresholdAbsolute),
		ParentsCount:            config.ParentsCount,
		WeakTipsRatio:           config.WeakTipsRatio,
		TSA:                     config.TSA,
		SchedulingRate:          config.SchedulingRate,
		IssuingRate:             config.IssuingRate,
		SlowdownFactor:          config.SlowdownFactor,
		ConsensusMonitorTick:    config.ConsensusMonitorTick,
		RelevantValidatorWeight: config.RelevantValidatorWeight,
		DoubleSpendDelay:        config.DoubleSpendDelay,
		PacketLoss:              config.PacketLoss,
		MinDelay:                config.MinDelay,
		MaxDelay:                config.MaxDelay,
		DeltaURTS:               config.DeltaURTS,
		SimulationStopThreshold: config.SimulationStopThreshold,
		ResultDir:               config.ResultDir,
		IMIF:                    config.IMIF,
		RandomnessWS:            config.RandomnessWS,
		NeighbourCountWS:        config.NeighbourCountWS,
		AdversaryTypes:          config.AdversaryTypes,
		AdversaryDelays:         config.AdversaryDelays,
		AdversaryMana:           config.AdversaryMana,
		AdversaryNodeCounts:     config.AdversaryNodeCounts,
		AdversaryInitColor:      config.AdversaryInitColors,
		SimulationMode:          config.SimulationMode,
		AccidentalMana:          config.AccidentalMana,
		AdversaryPeeringAll:     config.AdversaryPeeringAll,
		AdversarySpeedup:        config.AdversarySpeedup,
		BurnPolicyNames:         config.BurnPolicyNames,
	}

	bytes, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		log.Error(err)
	}
	if _, err = os.Stat(config.ResultDir); os.IsNotExist(err) {
		err = os.Mkdir(config.ResultDir, 0700)
		if err != nil {
			log.Error(err)
		}
	}
	if ioutil.WriteFile(path.Join(config.ResultDir, fileName), bytes, 0644) != nil {
		log.Error(err)
	}
}

func dumpNetwork(net *network.Network, fileName string) {
	nwHeader := []string{"Peer ID", "Neighbor ID", "Network Delay (ns)", "Packet Loss (%)", "Weight"}

	file, err := os.Create(path.Join(config.ResultDir, fileName))
	if err != nil {
		panic(err)
	}
	writer := csv.NewWriter(file)
	if err := writer.Write(nwHeader); err != nil {
		panic(err)
	}

	for _, peer := range net.Peers {
		for neighbor, connection := range peer.Neighbors {
			record := []string{
				strconv.FormatInt(int64(peer.ID), 10),
				strconv.FormatInt(int64(neighbor), 10),
				strconv.FormatInt(connection.NetworkDelay().Nanoseconds(), 10),
				strconv.FormatInt(int64(connection.PacketLoss()*100), 10),
				strconv.FormatInt(int64(net.WeightDistribution.Weight(peer.ID)), 10),
			}
			writeLine(writer, record)
		}
		// Flush the writers, or the data will be truncated for high node count
		writer.Flush()
	}
}

func monitorNetworkState(testNetwork *network.Network) (resultsWriters []*csv.Writer) {
	adversaryNodesCount := len(network.AdversaryNodeIDToGroupIDMap)
	honestNodesCount := config.NodesCount - adversaryNodesCount

	mostLikedColor = multiverse.UndefinedColor
	honestOnlyMostLikedColor = multiverse.UndefinedColor

	// The simulation start time
	simulationStartTime = time.Now()
	simulationStartTimeStr = simulationStartTime.UTC().Format(time.RFC3339)

	// Dump the configuration of this simulation
	dumpConfig(fmt.Sprint("aw-", simulationStartTimeStr, ".config"))

	// Dump the network information
	dumpNetwork(testNetwork, fmt.Sprint("nw-", simulationStartTimeStr, ".csv"))

	// Dump the info about adversary nodes
	adResultsWriter := createWriter(fmt.Sprintf("ad-%s.csv", simulationStartTimeStr), adHeader, &resultsWriters)
	dumpResultsAD(adResultsWriter, testNetwork)

	// Dump the double spending result
	dsResultsWriter := createWriter(fmt.Sprintf("ds-%s.csv", simulationStartTimeStr), dsHeader, &resultsWriters)

	// Dump the tip pool and processed message (throughput) results
	tpResultsWriter := createWriter(fmt.Sprintf("tp-%s.csv", simulationStartTimeStr), tpHeader, &resultsWriters)

	// Dump the requested missing message result
	mmResultsWriter := createWriter(fmt.Sprintf("mm-%s.csv", simulationStartTimeStr), mmHeader, &resultsWriters)

	tpAllHeader := make([]string, 0, config.NodesCount+1)

	for i := 0; i < config.NodesCount; i++ {
		header := []string{fmt.Sprintf("Node %d", i)}
		// fmt.Sprintf("Blue (Tip Pool Size) %d", i),
		// fmt.Sprintf("Red (Tip Pool Size) %d", i),
		// fmt.Sprintf("Green (Tip Pool Size) %d", i),
		// fmt.Sprintf("UndefinedColor (Processed) %d", i),
		// fmt.Sprintf("Blue (Processed) %d", i),
		// fmt.Sprintf("Red (Processed) %d", i),
		// fmt.Sprintf("Green (Processed) %d", i),
		// fmt.Sprintf("# of Issued Messages %d", i)}
		tpAllHeader = append(tpAllHeader, header...)
	}
	header := []string{fmt.Sprintf("ns since start")}
	tpAllHeader = append(tpAllHeader, header...)

	// Dump the tip pool and processed message (throughput) results
	tpAllResultsWriter := createWriter(fmt.Sprintf("all-tp-%s.csv", simulationStartTimeStr), tpAllHeader, &resultsWriters)

	// Dump the info about how many nodes have confirmed and liked a certain color
	ccResultsWriter := createWriter(fmt.Sprintf("cc-%s.csv", simulationStartTimeStr), ccHeader, &resultsWriters)

	// Define the file name of the ww results
	wwResultsWriter := createWriter(fmt.Sprintf("ww-%s.csv", simulationStartTimeStr), wwHeader, &resultsWriters)

	// Dump the Witness Weight
	wwPeer := testNetwork.Peers[config.MonitoredWitnessWeightPeer]
	previousWitnessWeight := uint64(config.NodesTotalWeight)
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

	for _, id := range config.MonitoredAWPeers {
		awPeer := testNetwork.Peers[id]
		if typeutils.IsInterfaceNil(awPeer) {
			panic(fmt.Sprintf("unknowm peer with id %d", id))
		}
		// Define the file name of the aw results
		awResultsWriter := createWriter(fmt.Sprintf("aw%d-%s.csv", id, simulationStartTimeStr), awHeader, &resultsWriters)

		awPeer.Node.(multiverse.NodeInterface).Tangle().ApprovalManager.Events.MessageConfirmed.Attach(
			events.NewClosure(func(message *multiverse.Message, messageMetadata *multiverse.MessageMetadata, weight uint64, messageIDCounter int64) {
				confirmedMessageMutex.Lock()
				confirmedMessageCounter[awPeer.ID]++
				confirmedMessageMutex.Unlock()

				confirmedMessageMutex.RLock()
				record := []string{
					strconv.FormatInt(int64(message.ID), 10),
					strconv.FormatInt(message.IssuanceTime.Unix(), 10),
					strconv.FormatInt(int64(messageMetadata.ConfirmationTime().Sub(message.IssuanceTime)), 10),
					strconv.FormatUint(weight, 10),
					strconv.FormatInt(confirmedMessageCounter[awPeer.ID], 10),
					strconv.FormatInt(messageIDCounter, 10),
					strconv.FormatInt(time.Since(simulationStartTime).Nanoseconds(), 10),
				}
				confirmedMessageMutex.RUnlock()

				csvMutex.Lock()
				if err := awResultsWriter.Write(record); err != nil {
					log.Fatal("error writing record to csv:", err)
				}

				if err := awResultsWriter.Error(); err != nil {
					log.Fatal(err)
				}
				csvMutex.Unlock()
			}))
	}

	go func() {
		for range dumpingTicker.C {
			dumpRecords(dsResultsWriter, tpResultsWriter, ccResultsWriter, adResultsWriter, tpAllResultsWriter, mmResultsWriter, honestNodesCount, adversaryNodesCount)
		}
	}()

	return
}

func dumpRecords(dsResultsWriter *csv.Writer, tpResultsWriter *csv.Writer, ccResultsWriter *csv.Writer, adResultsWriter *csv.Writer, tpAllResultsWriter *csv.Writer, mmResultsWriter *csv.Writer, honestNodesCount int, adversaryNodesCount int) {
	simulationWg.Add(1)
	simulationWg.Done()

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

	//r, g, b := getLikesPerRGB(colorCounters, "confirmedNodes")
	//aR, aG, aB := getLikesPerRGB(adversaryCounters, "confirmedNodes")
	//hR, hG, hB := r-aR, g-aG, b-aB
	//if Max(Max(hB, hR), hG) >= int64(config.SimulationStopThreshold*float64(honestNodesCount)) {
	//	shutdownSignal <- types.Void
	//}
	Simulator.GlobalCounters.Set("tps", 0)
}

func dumpResultDS(dsResultsWriter *csv.Writer, sinceIssuance string) {
	// Dump the double spending results
	record := []string{
		strconv.FormatInt(Simulator.ColorCounters.Get("opinionsWeights", multiverse.UndefinedColor), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("opinionsWeights", multiverse.Blue), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("opinionsWeights", multiverse.Red), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("opinionsWeights", multiverse.Green), 10),
		strconv.FormatInt(time.Since(simulationStartTime).Nanoseconds(), 10),
		sinceIssuance,
	}

	writeLine(dsResultsWriter, record)

	// Flush the writers, or the data will sometimes be truncated if the buffer is full
	dsResultsWriter.Flush()
}

func dumpResultsTP(tpResultsWriter *csv.Writer) {
	// Dump the tip pool sizes
	record := []string{
		strconv.FormatInt(Simulator.ColorCounters.Get("tipPoolSizes", multiverse.UndefinedColor), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("tipPoolSizes", multiverse.Blue), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("tipPoolSizes", multiverse.Red), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("tipPoolSizes", multiverse.Green), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("processedMessages", multiverse.UndefinedColor), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("processedMessages", multiverse.Blue), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("processedMessages", multiverse.Red), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("processedMessages", multiverse.Green), 10),
		strconv.FormatInt(Simulator.GlobalCounters.Get("issuedMessages"), 10),
		strconv.FormatInt(time.Since(simulationStartTime).Nanoseconds(), 10),
	}

	writeLine(tpResultsWriter, record)

	// Flush the writers, or the data will be truncated sometimes if the buffer is full
	tpResultsWriter.Flush()
}

func dumpResultsTPAll(tpAllResultsWriter *csv.Writer) {
	record := make([]string, config.NodesCount+1)
	i := 0
	for peerID := 0; peerID < config.NodesCount; peerID++ {
		tipCounterName := fmt.Sprint("tipPoolSizes-", peerID)
		// processedCounterName := fmt.Sprint("processedMessages-", peerID)
		// issuedCounterName := fmt.Sprint("issuedMessages-", peerID)
		record[i+0] = strconv.FormatInt(Simulator.ColorCounters.Get(tipCounterName, multiverse.UndefinedColor), 10)
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
		strconv.FormatInt(Simulator.ColorCounters.Get("requestedMissingMessages", multiverse.UndefinedColor), 10),
		strconv.FormatInt(time.Since(simulationStartTime).Nanoseconds(), 10),
	}

	writeLine(mmResultsWriter, record)

	// Flush the mm writer, or the data will be truncated sometimes if the buffer is full
	mmResultsWriter.Flush()
}

func dumpResultsCC(ccResultsWriter *csv.Writer, sinceIssuance string) {
	// Dump the opinion and confirmation counters
	record := []string{
		strconv.FormatInt(Simulator.ColorCounters.Get("confirmedNodes", multiverse.Blue), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("confirmedNodes", multiverse.Red), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("confirmedNodes", multiverse.Green), 10),
		strconv.FormatInt(Simulator.AdversaryCounters.Get("confirmedNodes", multiverse.Blue), 10),
		strconv.FormatInt(Simulator.AdversaryCounters.Get("confirmedNodes", multiverse.Red), 10),
		strconv.FormatInt(Simulator.AdversaryCounters.Get("confirmedNodes", multiverse.Green), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("confirmedAccumulatedWeight", multiverse.Blue), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("confirmedAccumulatedWeight", multiverse.Red), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("confirmedAccumulatedWeight", multiverse.Green), 10),
		strconv.FormatInt(Simulator.AdversaryCounters.Get("confirmedAccumulatedWeight", multiverse.Blue), 10),
		strconv.FormatInt(Simulator.AdversaryCounters.Get("confirmedAccumulatedWeight", multiverse.Red), 10),
		strconv.FormatInt(Simulator.AdversaryCounters.Get("confirmedAccumulatedWeight", multiverse.Green), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("opinions", multiverse.Blue), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("opinions", multiverse.Red), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("opinions", multiverse.Green), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("likeAccumulatedWeight", multiverse.Blue), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("likeAccumulatedWeight", multiverse.Red), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("likeAccumulatedWeight", multiverse.Green), 10),
		strconv.FormatInt(Simulator.AdversaryCounters.Get("likeAccumulatedWeight", multiverse.Blue), 10),
		strconv.FormatInt(Simulator.AdversaryCounters.Get("likeAccumulatedWeight", multiverse.Red), 10),
		strconv.FormatInt(Simulator.AdversaryCounters.Get("likeAccumulatedWeight", multiverse.Green), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("colorUnconfirmed", multiverse.Blue), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("colorUnconfirmed", multiverse.Red), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("colorUnconfirmed", multiverse.Green), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("unconfirmedAccumulatedWeight", multiverse.Blue), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("unconfirmedAccumulatedWeight", multiverse.Red), 10),
		strconv.FormatInt(Simulator.ColorCounters.Get("unconfirmedAccumulatedWeight", multiverse.Green), 10),
		strconv.FormatInt(Simulator.GlobalCounters.Get("flips"), 10),
		strconv.FormatInt(Simulator.GlobalCounters.Get("honestFlips"), 10),
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
			strconv.FormatFloat(float64(group.GroupMana)/float64(config.NodesTotalWeight), 'f', 6, 64),
			strconv.FormatInt(time.Since(simulationStartTime).Nanoseconds(), 10),
		}
		writeLine(adResultsWriter, record)
	}
	// Flush the cc writer, or the data will be truncated sometimes if the buffer is full
	adResultsWriter.Flush()
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
	file, err := os.Create(path.Join(config.ResultDir, fileName))
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
