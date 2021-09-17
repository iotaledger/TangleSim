package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/iotaledger/multivers-simulation/adversary"
	"github.com/iotaledger/multivers-simulation/simulation"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/iotaledger/hive.go/types"

	"github.com/iotaledger/hive.go/events"
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
	dsHeader = []string{"UndefinedColor", "Blue", "Red", "Green", "ns since start", "ns since issuance"}
	tpHeader = []string{"UndefinedColor (Tip Pool Size)", "Blue (Tip Pool Size)", "Red (Tip Pool Size)", "Green (Tip Pool Size)",
		"UndefinedColor (Processed)", "Blue (Processed)", "Red (Processed)", "Green (Processed)", "# of Issued Messages", "ns since start"}
	ccHeader = []string{"Blue (Confirmed)", "Red (Confirmed)", "Green (Confirmed)",
		"Blue (Confirmed Accumulated Weight)", "Red (Confirmed Accumulated Weight)", "Green (Confirmed Accumulated Weight)",
		"Blue (Like)", "Red (Like)", "Green (Like)",
		"Blue (Like Accumulated Weight)", "Red (Like Accumulated Weight)", "Green (Like Accumulated Weight)",
		"Blue (Adversary Liked AW)", "Red (Adversary Like AW)", "Green (Adversary Like AW)",
		"Unconfirmed Blue", "Unconfirmed Red", "Unconfirmed Green",
		"Unconfirmed Blue Accumulated Weight", "Unconfirmed Red Accumulated Weight", "Unconfirmed Green Accumulated Weight",
		"Flips (Winning color changed)", "Honest nodes Flips", "ns since start", "ns since issuance"}
	adHeader = []string{"AdversaryGroupID", "Strategy", "AdversaryCount", "q", "ns since issuance"}

	csvMutex sync.Mutex

	// simulation variables
	dumpingTicker         = time.NewTicker(time.Duration(config.DecelerationFactor*config.ConsensusMonitorTick) * time.Millisecond)
	simulationWg          = sync.WaitGroup{}
	maxSimulationDuration = time.Minute
	shutdownSignal        = make(chan types.Empty)

	// global declarations
	dsIssuanceTime           time.Time
	mostLikedColor           multiverse.Color
	honestOnlyMostLikedColor multiverse.Color
	simulationStartTime      time.Time

	// counters
	colorCounters     = simulation.NewColorCounters()
	adversaryCounters = simulation.NewColorCounters()
	atomicCounters    = simulation.NewAtomicCounters()

	confirmedMessageCounter = make(map[network.PeerID]int64)
	confirmedMessageMutex   sync.RWMutex
)

func main() {
	log.Info("Starting simulation ... [DONE]")
	defer log.Info("Shutting down simulation ... [DONE]")
	simulation.ParseFlags()

	nodeFactories := map[network.AdversaryType]network.NodeFactory{
		network.HonestNode:     network.NodeClosure(multiverse.NewNode),
		network.ShiftOpinion:   network.NodeClosure(adversary.NewShiftingOpinionNode),
		network.TheSameOpinion: network.NodeClosure(adversary.NewSameOpinionNode),
	}
	testNetwork := network.New(
		network.Nodes(config.NodesCount, nodeFactories, network.ZIPFDistribution(
			config.ZipfParameter, float64(config.NodesTotalWeight))),
		network.Delay(time.Duration(config.DecelerationFactor)*time.Duration(config.MinDelay)*time.Millisecond,
			time.Duration(config.DecelerationFactor)*time.Duration(config.MaxDelay)*time.Millisecond),
		network.PacketLoss(0, config.PayloadLoss),
		network.Topology(network.WattsStrogatz(config.NeighbourCountWS*2, config.RandomnessWS)),
	)
	testNetwork.Start()
	defer testNetwork.Shutdown()

	resultsWriters := monitorNetworkState(testNetwork)
	defer flushWriters(resultsWriters)
	secureNetwork(testNetwork)

	// To simulate the confirmation time w/o any double spendings, the colored msgs are not to be sent

	// Here we simulate the double spending
	if config.SimulationTarget == "DS" {
		time.Sleep(time.Duration(config.DoubleSpendDelay*config.DecelerationFactor) * time.Second)
		dsIssuanceTime = time.Now()
		for groupIndex, group := range testNetwork.AdversaryGroups {
			color := multiverse.ColorFromInt(groupIndex + 1)
			// honest node does not implement adversary behavior interface
			if group.AdversaryType != network.HonestNode {
				for _, nodeID := range group.NodeIDs {
					node := adversary.CastAdversary(testNetwork.Peer(nodeID).Node)
					node.AssignColor(color)
				}
			}
			// only one node from the group needs to issue message
			sendMessage(testNetwork.Peer(group.NodeIDs[0]), color)
		}
	}

	select {
	case <-shutdownSignal:
		shutdownSimulation()
		log.Info("Shutting down simulation (consensus reached) ... [DONE]")
	case <-time.After(time.Duration(config.DecelerationFactor) * maxSimulationDuration):
		shutdownSimulation()
		log.Info("Shutting down simulation (simulation timed out) ... [DONE]")
	}
}

func shutdownSimulation() {
	dumpingTicker.Stop()
	simulationWg.Wait()
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
		NodesCount, NodesTotalWeight, TipsCount, TPS, ConsensusMonitorTick, RelevantValidatorWeight, MinDelay, MaxDelay, DecelerationFactor, DoubleSpendDelay, NeighbourCountWS int
		ZipfParameter, MessageWeightThreshold, WeakTipsRatio, PayloadLoss, DeltaURTS, SimulationStopThreshold, RandomnessWS, AdversaryErrorThreshold                            float64
		TSA, ResultDir, IMIF, SimulationTarget                                                                                                                                  string
		AdversaryDelays, AdversaryTypes                                                                                                                                         []int
		AdversaryMana                                                                                                                                                           []float64
	}
	data := Configuration{
		NodesCount:              config.NodesCount,
		NodesTotalWeight:        config.NodesTotalWeight,
		ZipfParameter:           config.ZipfParameter,
		MessageWeightThreshold:  config.MessageWeightThreshold,
		TipsCount:               config.TipsCount,
		WeakTipsRatio:           config.WeakTipsRatio,
		TSA:                     config.TSA,
		TPS:                     config.TPS,
		DecelerationFactor:      config.DecelerationFactor,
		ConsensusMonitorTick:    config.ConsensusMonitorTick,
		RelevantValidatorWeight: config.RelevantValidatorWeight,
		DoubleSpendDelay:        config.DoubleSpendDelay,
		PayloadLoss:             config.PayloadLoss,
		MinDelay:                config.MinDelay,
		MaxDelay:                config.MaxDelay,
		DeltaURTS:               config.DeltaURTS,
		SimulationStopThreshold: config.SimulationStopThreshold,
		SimulationTarget:        config.SimulationTarget,
		ResultDir:               config.ResultDir,
		IMIF:                    config.IMIF,
		RandomnessWS:            config.RandomnessWS,
		NeighbourCountWS:        config.NeighbourCountWS,
		AdversaryTypes:          config.AdversaryTypes,
		AdversaryDelays:         config.AdversaryDelays,
		AdversaryMana:           config.AdversaryMana,
		AdversaryErrorThreshold: config.AdversaryErrorThreshold,
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

func monitorNetworkState(testNetwork *network.Network) (resultsWriters []*csv.Writer) {
	adversaryNodesCount := len(network.AdversaryNodeIDToGroupIDMap)
	honestNodesCount := config.NodesCount - adversaryNodesCount

	allColors := []multiverse.Color{multiverse.UndefinedColor, multiverse.Red, multiverse.Green, multiverse.Blue}

	colorCounters.CreateCounter("opinions", allColors, []int64{int64(config.NodesCount), 0, 0, 0})
	colorCounters.CreateCounter("confirmedNodes", allColors, []int64{0, 0, 0, 0})
	colorCounters.CreateCounter("opinionsWeights", allColors, []int64{0, 0, 0, 0})
	colorCounters.CreateCounter("likeAccumulatedWeight", allColors, []int64{0, 0, 0, 0})
	colorCounters.CreateCounter("processedMessages", allColors, []int64{0, 0, 0, 0})
	colorCounters.CreateCounter("tipPoolSizes", allColors, []int64{0, 0, 0, 0})
	colorCounters.CreateCounter("colorUnconfirmed", allColors[1:], []int64{0, 0, 0})
	colorCounters.CreateCounter("confirmedAccumulatedWeight", allColors[1:], []int64{0, 0, 0})
	colorCounters.CreateCounter("unconfirmedAccumulatedWeight", allColors[1:], []int64{0, 0, 0})

	adversaryCounters.CreateCounter("likeAccumulatedWeight", allColors[1:], []int64{0, 0, 0})
	adversaryCounters.CreateCounter("opinions", allColors, []int64{int64(config.NodesCount), 0, 0, 0})

	atomicCounters.CreateAtomicCounter("flips", 0)
	atomicCounters.CreateAtomicCounter("honestFlips", 0)
	atomicCounters.CreateAtomicCounter("tps", 0)
	atomicCounters.CreateAtomicCounter("relevantValidators", 0)
	atomicCounters.CreateAtomicCounter("issuedMessages", 0)

	mostLikedColor = multiverse.UndefinedColor

	// The simulation start time
	simulationStartTime = time.Now()
	simulationStartTimeStr := simulationStartTime.UTC().Format(time.RFC3339)

	// Dump the configuration of this simulation
	dumpConfig(fmt.Sprint("aw-", simulationStartTimeStr, ".config"))

	// Dump the info about adversary nodes
	adResultsWriter := createWriter(fmt.Sprintf("ad-%s.csv", simulationStartTimeStr), adHeader, &resultsWriters)
	dumpResultsAD(adResultsWriter, testNetwork)

	// Dump the double spending result
	dsResultsWriter := createWriter(fmt.Sprintf("ds-%s.csv", simulationStartTimeStr), dsHeader, &resultsWriters)

	// Dump the tip pool and processed message (throughput) results
	tpResultsWriter := createWriter(fmt.Sprintf("tp-%s.csv", simulationStartTimeStr), tpHeader, &resultsWriters)

	// Dump the info about how many nodes have confirmed and liked a certain color
	ccResultsWriter := createWriter(fmt.Sprintf("cc-%s.csv", simulationStartTimeStr), ccHeader, &resultsWriters)

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
				confirmedMessageCounter[awPeer.ID] += 1
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

	for _, peer := range testNetwork.Peers {
		peer.Node.(multiverse.NodeInterface).Tangle().OpinionManager.Events().OpinionChanged.Attach(events.NewClosure(func(oldOpinion multiverse.Color, newOpinion multiverse.Color, weight int64, peerID network.PeerID) {
			colorCounters.Add("opinions", -1, oldOpinion)
			colorCounters.Add("opinions", 1, newOpinion)

			colorCounters.Add("likeAccumulatedWeight", -weight, oldOpinion)
			colorCounters.Add("likeAccumulatedWeight", weight, newOpinion)

			r, g, b := getLikesPerRGB(colorCounters, "opinions")
			if mostLikedColorChanged(r, g, b, mostLikedColor) {
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
			if mostLikedColorChanged(r-ar, g-ag, b-ab, honestOnlyMostLikedColor) {
				atomicCounters.Add("honestFlips", 1)
			}
		}))
		peer.Node.(multiverse.NodeInterface).Tangle().OpinionManager.Events().ColorConfirmed.Attach(events.NewClosure(func(confirmedColor multiverse.Color, weight int64) {
			colorCounters.Add("confirmedNodes", 1, confirmedColor)
			colorCounters.Add("confirmedAccumulatedWeight", weight, confirmedColor)
		}))

		peer.Node.(multiverse.NodeInterface).Tangle().OpinionManager.Events().ColorUnconfirmed.Attach(events.NewClosure(func(unconfirmedColor multiverse.Color, weight int64) {
			colorCounters.Add("colorUnconfirmed", 1, unconfirmedColor)
			colorCounters.Add("confirmedNodes", -1, unconfirmedColor)

			colorCounters.Add("unconfirmedAccumulatedWeight", weight, unconfirmedColor)
			colorCounters.Add("confirmedAccumulatedWeight", -weight, unconfirmedColor)
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

	go func() {
		for range dumpingTicker.C {
			dumpRecords(dsResultsWriter, tpResultsWriter, ccResultsWriter, adResultsWriter, honestNodesCount, adversaryNodesCount)
		}
	}()

	return
}

func dumpRecords(dsResultsWriter *csv.Writer, tpResultsWriter *csv.Writer, ccResultsWriter *csv.Writer, adResultsWriter *csv.Writer, honestNodesCount int, adversaryNodesCount int) {
	simulationWg.Add(1)
	simulationWg.Done()

	log.Infof("New opinions counter[ %3d Undefined / %3d Blue / %3d Red / %3d Green ]",
		colorCounters.Get("opinions", multiverse.UndefinedColor),
		colorCounters.Get("opinions", multiverse.Blue),
		colorCounters.Get("opinions", multiverse.Red),
		colorCounters.Get("opinions", multiverse.Green),
	)
	log.Infof("Network Status: %3d TPS :: Consensus[ %3d Undefined / %3d Blue / %3d Red / %3d Green ] :: %d  Honest Nodes :: %d Adversary Nodes :: %d Validators",
		atomicCounters.Get("tps"),
		colorCounters.Get("confirmedNodes", multiverse.UndefinedColor),
		colorCounters.Get("confirmedNodes", multiverse.Blue),
		colorCounters.Get("confirmedNodes", multiverse.Red),
		colorCounters.Get("confirmedNodes", multiverse.Green),
		honestNodesCount,
		adversaryNodesCount,
		atomicCounters.Get("relevantValidators"),
	)

	sinceIssuance := "0"
	if !dsIssuanceTime.IsZero() {
		sinceIssuance = strconv.FormatInt(time.Since(dsIssuanceTime).Nanoseconds(), 10)

	}

	dumpResultDS(dsResultsWriter, sinceIssuance)
	dumpResultsTP(tpResultsWriter)
	dumpResultsCC(ccResultsWriter, sinceIssuance)

	// determines whether consensus has been reached and simulation is over
	if Max(Max(colorCounters.GetInt("confirmedNodes", multiverse.Blue), colorCounters.GetInt("confirmedNodes", multiverse.Red)), colorCounters.GetInt("confirmedNodes", multiverse.Green)) >= int(config.SimulationStopThreshold*float64(honestNodesCount)) {
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
}

func dumpResultsCC(ccResultsWriter *csv.Writer, sinceIssuance string) {
	// Dump the opinion and confirmation counters
	record := []string{
		strconv.FormatInt(colorCounters.Get("confirmedNodes", multiverse.Blue), 10),
		strconv.FormatInt(colorCounters.Get("confirmedNodes", multiverse.Red), 10),
		strconv.FormatInt(colorCounters.Get("confirmedNodes", multiverse.Green), 10),
		strconv.FormatInt(colorCounters.Get("confirmedAccumulatedWeight", multiverse.Blue), 10),
		strconv.FormatInt(colorCounters.Get("confirmedAccumulatedWeight", multiverse.Red), 10),
		strconv.FormatInt(colorCounters.Get("confirmedAccumulatedWeight", multiverse.Green), 10),
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

func secureNetwork(testNetwork *network.Network) {
	// In the simulation we let all nodes can send messages.
	// largestWeight := float64(testNetwork.WeightDistribution.LargestWeight())

	for _, peer := range testNetwork.Peers {
		weightOfPeer := float64(testNetwork.WeightDistribution.Weight(peer.ID))

		// if float64(config.RelevantValidatorWeight)*weightOfPeer <= largestWeight {
		// 	continue
		// }

		atomicCounters.Add("relevantValidators", 1)

		// Each peer should send messages according to their mana: Fix TPS for example 1000;
		// A node with a x% of mana will issue 1000*x% messages per second

		// Weight: 100, 20, 1
		// TPS: 1000
		// Band widths summed up: 100000/121 + 20000/121 + 1000/121 = 1000

		band := weightOfPeer * float64(config.TPS) / float64(config.NodesTotalWeight)

		go startSecurityWorker(peer, band)
	}
}

func startSecurityWorker(peer *network.Peer, band float64) {
	pace := time.Duration(float64(time.Second) * float64(config.DecelerationFactor) / band)

	log.Debug("Peer ID: ", peer.ID, " Pace: ", pace)
	if pace == time.Duration(0) {
		log.Warn("Peer ID: ", peer.ID, " has 0 pace!")
		return
	}
	ticker := time.NewTicker(pace)

	for {
		select {
		case <-ticker.C:
			if config.IMIF == "poisson" {
				pace = time.Duration(float64(time.Second) * float64(config.DecelerationFactor) * rand.ExpFloat64() / band)
				if pace > 0 {
					ticker.Reset(pace)
				}
			}
			sendMessage(peer)
		}
	}
}

func sendMessage(peer *network.Peer, optionalColor ...multiverse.Color) {
	atomicCounters.Add("tps", 1)

	if len(optionalColor) >= 1 {
		peer.Node.(multiverse.NodeInterface).IssuePayload(optionalColor[0])
	}

	peer.Node.(multiverse.NodeInterface).IssuePayload(multiverse.UndefinedColor)
}

// Max returns the larger of x or y.
func Max(x, y int) int {
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

func mostLikedColorChanged(r, g, b int64, mostLikedColorVar multiverse.Color) bool {

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
	if mostLikedColorVar != currentMostLikedColor {
		// color selected for the first time, it not counts
		if mostLikedColorVar == multiverse.UndefinedColor {
			mostLikedColorVar = currentMostLikedColor
			return false
		}
		mostLikedColorVar = currentMostLikedColor
		return true
	}
	return false
}
