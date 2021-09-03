package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
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
	log      = logger.New("Simulation")
	awHeader = []string{"Message ID", "Issuance Time (unix)", "Confirmation Time (ns)", "Weight", "# of Confirmed Messages", "# of Issued Messages"}
	dsHeader = []string{"UndefinedColor", "Blue", "Red", "Green"}
	tpHeader = []string{"UndefinedColor (Tip Pool Size)", "Blue (Tip Pool Size)", "Red (Tip Pool Size)", "Green (Tip Pool Size)",
		"UndefinedColor (Processed)", "Blue (Processed)", "Red (Processed)", "Green (Processed)", "# of Issued Messages"}
	csvMutex              sync.Mutex
	shutdownSignal        = make(chan types.Empty)
	maxSimulationDuration = time.Minute
	simulationTarget      = "CT" // "DS" (CT: confirmation time anslysis, DS: double spending analysis)
)

// Parse the flags and update the configuration
func parseFlags() {

	// Define the configuration flags
	nodesCountPtr :=
		flag.Int("nodesCount", config.NodesCount, "The number of nodes")
	nodesTotalWeightPtr :=
		flag.Int("nodesTotalWeight", config.NodesTotalWeight, "The total weight of nodes")
	zipfParameterPtr :=
		flag.Float64("zipfParameter", config.ZipfParameter, "The zipf's parameter")
	messageWeightThresholdPtr :=
		flag.Float64("messageWeightThreshold", config.MessageWeightThreshold, "The messageWeightThreshold of confirmed messages")
	tipsCountPtr :=
		flag.Int("tipsCount", config.TipsCount, "The tips count for a message")
	weakTipsRatioPtr :=
		flag.Float64("weakTipsRatio", config.WeakTipsRatio, "The ratio of weak tips")
	tsaPtr :=
		flag.String("tsa", config.TSA, "The tip selection algorithm")
	tpsPtr :=
		flag.Int("tps", config.TPS, "the tips per seconds")
	decelerationFactorPtr :=
		flag.Float64("decelerationFactor", config.DecelerationFactor, "The factor to control the speed in the simulation")
	consensusMonitorTickPtr :=
		flag.Int("consensusMonitorTick", config.ConsensusMonitorTick, "The tick to monitor the consensus, in milliseconds")
	releventValidatorWeightPtr :=
		flag.Int("releventValidatorWeight", config.ReleventValidatorWeight, "The node whose weight * ReleventValidatorWeight <= largestWeight will not issue messages")
	payloadLoss := flag.Float64("payloadLoss", config.PayloadLoss, "The payload loss percentage")
	minDelay := flag.Int("minDelay", config.MinDelay, "The minimum network delay in ms")
	maxDelay := flag.Int("maxDelay", config.MaxDelay, "The maximum network delay in ms")
	deltaURTS := flag.Float64("DeltaURTS", config.DeltaURTS, "in seconds, reference: https://iota.cafe/t/orphanage-with-restricted-urts/1199")
	simulationStopThreshold := flag.Float64("SimulationStopThreshold", config.SimulationStopThreshold, "Stop the simulation when > SimulationStopThreshold * NodesCount have reached the same opinion")
	simulationTarget := flag.String("SimulationTarget", config.SimulationTarget, "The simulation target, CT: Confirmation Time, DS: Double Spending")

	// Parse the flags
	flag.Parse()

	// Update the configuration parameters
	config.NodesCount = *nodesCountPtr
	config.NodesTotalWeight = *nodesTotalWeightPtr
	config.ZipfParameter = *zipfParameterPtr
	config.MessageWeightThreshold = *messageWeightThresholdPtr
	config.TipsCount = *tipsCountPtr
	config.WeakTipsRatio = *weakTipsRatioPtr
	config.TSA = *tsaPtr
	config.TPS = *tpsPtr
	config.DecelerationFactor = *decelerationFactorPtr
	config.ConsensusMonitorTick = *consensusMonitorTickPtr
	config.ReleventValidatorWeight = *releventValidatorWeightPtr
	config.PayloadLoss = *payloadLoss
	config.MinDelay = *minDelay
	config.MaxDelay = *maxDelay
	config.DeltaURTS = *deltaURTS
	config.SimulationStopThreshold = *simulationStopThreshold
	config.SimulationTarget = *simulationTarget

	log.Info("Current configuration:")
	log.Info("NodesCount: ", config.NodesCount)
	log.Info("NodesTotalWeight: ", config.NodesTotalWeight)
	log.Info("ZipfParameter: ", config.ZipfParameter)
	log.Info("MessageWeightThreshold: ", config.MessageWeightThreshold)
	log.Info("TipsCount: ", config.TipsCount)
	log.Info("WeakTipsRatio: ", config.WeakTipsRatio)
	log.Info("TSA: ", config.TSA)
	log.Info("TPS: ", config.TPS)
	log.Info("DecelerationFactor: ", config.DecelerationFactor)
	log.Info("ConsensusMonitorTick: ", config.ConsensusMonitorTick)
	log.Info("ReleventValidatorWeight: ", config.ReleventValidatorWeight)
	log.Info("PayloadLoss: ", config.PayloadLoss)
	log.Info("MinDelay: ", config.MinDelay)
	log.Info("MaxDelay: ", config.MaxDelay)
	log.Info("DeltaURTS:", config.DeltaURTS)
	log.Info("SimulationStopThreshold:", config.SimulationStopThreshold)
	log.Info("SimulationTarget:", config.SimulationTarget)
}

func main() {
	log.Info("Starting simulation ... [DONE]")
	defer log.Info("Shutting down simulation ... [DONE]")

	parseFlags()
	testNetwork := network.New(
		network.Nodes(config.NodesCount, multiverse.NewNode, network.ZIPFDistribution(
			config.ZipfParameter, float64(config.NodesTotalWeight))),
		network.Delay(time.Duration(config.DecelerationFactor)*time.Duration(config.MinDelay)*time.Millisecond,
			time.Duration(config.DecelerationFactor)*time.Duration(config.MaxDelay)*time.Millisecond),
		network.PacketLoss(0, config.PayloadLoss),
		network.Topology(network.WattsStrogatz(4, 1)),
	)
	testNetwork.Start()
	defer testNetwork.Shutdown()

	resultsWriters := monitorNetworkState(testNetwork)
	defer flushWriters(resultsWriters)
	secureNetwork(testNetwork, config.DecelerationFactor)

	// To simulate the confirmation time w/o any double spendings, the colored msgs are not to be sent

	// Here we simulate the double spending
	if config.SimulationTarget == "DS" {
		time.Sleep(2 * time.Second)
		sendMessage(testNetwork.Peers[0], multiverse.Blue)
		sendMessage(testNetwork.Peers[0], multiverse.Red)
	}

	// Here three peers are randomly selected with defined Colors
	// attackers := testNetwork.RandomPeers(3)
	// sendMessage(attackers[0], multiverse.Red)
	// sendMessage(attackers[1], multiverse.Blue)
	// sendMessage(attackers[2], multiverse.Green)

	select {
	case <-shutdownSignal:
		log.Info("Shutting down simulation (consensus reached) ... [DONE]")
	case <-time.After(maxSimulationDuration):
		log.Info("Shutting down simulation (simulation timed out) ... [DONE]")
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

var (
	tpsCounter = uint64(0)

	opinions = make(map[multiverse.Color]int)

	opinionsWeights = make(map[multiverse.Color]int64)

	tipPoolSizes = make(map[multiverse.Color]int)

	processedMessageCounts = make(map[multiverse.Color]uint64)

	issuedMessageCounter = int64(0)

	confirmedCounter = make(map[multiverse.Color]int)

	confirmedMessageCounter = int64(0)

	opinionMutex sync.Mutex

	confirmationMutex sync.Mutex

	opinionWeightMutex sync.Mutex

	processedMessageMutex sync.Mutex

	relevantValidators int
)

func dumpConfig(fileName string) {
	type Configuration struct {
		NodesCount, NodesTotalWeight, TipsCount, TPS, ConsensusMonitorTick, ReleventValidatorWeight, MinDelay, MaxDelay int
		ZipfParameter, MessageWeightThreshold, WeakTipsRatio, DecelerationFactor, PayloadLoss, DeltaURTS                float64
		TSA                                                                                                             string
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
		ReleventValidatorWeight: config.ReleventValidatorWeight,
		PayloadLoss:             config.PayloadLoss,
		MinDelay:                config.MinDelay,
		MaxDelay:                config.MaxDelay,
		DeltaURTS:               config.DeltaURTS,
	}

	file, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		log.Error(err)
	}
	if ioutil.WriteFile(fileName, file, 0644) != nil {
		log.Error(err)
	}
}

func monitorNetworkState(testNetwork *network.Network) (resultsWriters []*csv.Writer) {
	opinions[multiverse.UndefinedColor] = config.NodesCount
	opinions[multiverse.Blue] = 0
	opinions[multiverse.Red] = 0
	opinions[multiverse.Green] = 0
	opinionsWeights[multiverse.UndefinedColor] = 0
	opinionsWeights[multiverse.Blue] = 0
	opinionsWeights[multiverse.Red] = 0
	opinionsWeights[multiverse.Green] = 0

	// The simulation start time
	simulationStartTime := time.Now().UTC().Format(time.RFC3339)

	// Dump the configuration of this simulation
	dumpConfig(fmt.Sprint("aw-", simulationStartTime, ".config"))

	// Dump the double spending result
	// Define the file name of the aw results
	dsFileName := fmt.Sprint("ds-", simulationStartTime, ".result")
	file, err := os.Create(dsFileName)
	if err != nil {
		panic(err)
	}
	dsResultsWriter := csv.NewWriter(file)

	// Dump the tip pool and processed message (throughput) results
	// Define the file name of the throughput results
	tpFileName := fmt.Sprint("tp-", simulationStartTime, ".result")
	file, err = os.Create(tpFileName)
	if err != nil {
		panic(err)
	}
	tpResultsWriter := csv.NewWriter(file)

	// Check the result writers
	resultsWriters = append(resultsWriters, dsResultsWriter)
	resultsWriters = append(resultsWriters, tpResultsWriter)

	// Write the headers
	if err = dsResultsWriter.Write(dsHeader); err != nil {
		panic(err)
	}
	if err = tpResultsWriter.Write(tpHeader); err != nil {
		panic(err)
	}

	for _, id := range config.MonitoredAWPeers {
		awPeer := testNetwork.Peers[id]
		if typeutils.IsInterfaceNil(awPeer) {
			panic(fmt.Sprintf("unknowm peer with id %d", id))
		}
		// Define the file name of the aw results
		fileName := fmt.Sprint("aw", id, "-", simulationStartTime, ".result")

		file, err := os.Create(fileName)
		if err != nil {
			panic(err)
		}
		awResultsWriter := csv.NewWriter(file)
		if err = awResultsWriter.Write(awHeader); err != nil {
			panic(err)
		}
		resultsWriters = append(resultsWriters, awResultsWriter)
		awPeer.Node.(*multiverse.Node).Tangle.ApprovalManager.Events.MessageConfirmed.Attach(
			events.NewClosure(func(message *multiverse.Message, messageMetadata *multiverse.MessageMetadata, weight uint64, messageIDCounter int64) {
				atomic.AddInt64(&confirmedMessageCounter, 1)

				record := []string{
					strconv.FormatInt(int64(message.ID), 10),
					strconv.FormatInt(message.IssuanceTime.Unix(), 10),
					strconv.FormatInt(int64(messageMetadata.ConfirmationTime().Sub(message.IssuanceTime)), 10),
					strconv.FormatUint(weight, 10),
					strconv.FormatInt(confirmedMessageCounter, 10),
					strconv.FormatInt(messageIDCounter, 10),
				}

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
		peer.Node.(*multiverse.Node).Tangle.OpinionManager.Events.OpinionChanged.Attach(events.NewClosure(func(oldOpinion multiverse.Color, newOpinion multiverse.Color) {
			opinionMutex.Lock()
			defer opinionMutex.Unlock()

			opinions[oldOpinion]--
			opinions[newOpinion]++
		}))
		peer.Node.(*multiverse.Node).Tangle.OpinionManager.Events.ColorConfirmed.Attach(events.NewClosure(func(confirmedColor multiverse.Color) {
			confirmationMutex.Lock()
			defer confirmationMutex.Unlock()

			confirmedCounter[confirmedColor]++
		}))

		peer.Node.(*multiverse.Node).Tangle.OpinionManager.Events.ColorUnconfirmed.Attach(events.NewClosure(func(unconfirmedColor multiverse.Color) {
			confirmationMutex.Lock()
			defer confirmationMutex.Unlock()

			confirmedCounter[unconfirmedColor]--
		}))
	}

	// Here we only monitor the opinion weight of node w/ the highest weight
	dsPeer := testNetwork.Peers[0]
	dsPeer.Node.(*multiverse.Node).Tangle.OpinionManager.Events.ApprovalWeightUpdated.Attach(events.NewClosure(func(opinion multiverse.Color, delta_weight int64) {
		opinionWeightMutex.Lock()
		defer opinionWeightMutex.Unlock()

		opinionsWeights[opinion] += delta_weight
	}))

	// Here we only monitor the tip pool size of node w/ the highest weight
	peer := testNetwork.Peers[0]
	peer.Node.(*multiverse.Node).Tangle.TipManager.Events.MessageProcessed.Attach(events.NewClosure(
		func(opinion multiverse.Color, tipPoolSize int, processedMessages uint64, issuedMessages int64) {
			processedMessageMutex.Lock()
			defer processedMessageMutex.Unlock()

			tipPoolSizes[opinion] = tipPoolSize
			processedMessageCounts[opinion] = processedMessages
			issuedMessageCounter = issuedMessages
		}))

	go func() {
		for range time.Tick(time.Duration(config.ConsensusMonitorTick) * time.Millisecond) {
			log.Infof("Network Status: %d TPS :: Consensus[ %d Undefined / %d Blue / %d Red / %d Green ] :: %d Nodes :: %d Validators",
				atomic.LoadUint64(&tpsCounter),
				confirmedCounter[multiverse.UndefinedColor],
				confirmedCounter[multiverse.Blue],
				confirmedCounter[multiverse.Red],
				confirmedCounter[multiverse.Green],
				config.NodesCount,
				relevantValidators,
			)

			// Dump the double spending results
			record := []string{
				strconv.FormatInt(int64(opinionsWeights[multiverse.UndefinedColor]), 10),
				strconv.FormatInt(int64(opinionsWeights[multiverse.Blue]), 10),
				strconv.FormatInt(int64(opinionsWeights[multiverse.Red]), 10),
				strconv.FormatInt(int64(opinionsWeights[multiverse.Green]), 10),
			}

			if err := dsResultsWriter.Write(record); err != nil {
				log.Fatal("error writing record to csv:", err)
			}

			if err := dsResultsWriter.Error(); err != nil {
				log.Fatal(err)
			}

			// Dump the tip pool sizes
			record = []string{
				strconv.FormatInt(int64(tipPoolSizes[multiverse.UndefinedColor]), 10),
				strconv.FormatInt(int64(tipPoolSizes[multiverse.Blue]), 10),
				strconv.FormatInt(int64(tipPoolSizes[multiverse.Red]), 10),
				strconv.FormatInt(int64(tipPoolSizes[multiverse.Green]), 10),
				strconv.FormatInt(int64(processedMessageCounts[multiverse.UndefinedColor]), 10),
				strconv.FormatInt(int64(processedMessageCounts[multiverse.Blue]), 10),
				strconv.FormatInt(int64(processedMessageCounts[multiverse.Red]), 10),
				strconv.FormatInt(int64(processedMessageCounts[multiverse.Green]), 10),
				strconv.FormatInt(int64(issuedMessageCounter), 10),
			}

			if err := tpResultsWriter.Write(record); err != nil {
				log.Fatal("error writing record to csv:", err)
			}

			if err := tpResultsWriter.Error(); err != nil {
				log.Fatal(err)
			}

			if Max(Max(confirmedCounter[multiverse.Blue], confirmedCounter[multiverse.Red]), confirmedCounter[multiverse.Green]) >= int(config.SimulationStopThreshold*float64(config.NodesCount)) {
				shutdownSignal <- types.Void
			}
			atomic.StoreUint64(&tpsCounter, 0)
		}
	}()

	return
}

func secureNetwork(testNetwork *network.Network, decelerationFactor float64) {
	// In the simulation we let all nodes can send messages.
	// largestWeight := float64(testNetwork.WeightDistribution.LargestWeight())

	for _, peer := range testNetwork.Peers {
		weightOfPeer := float64(testNetwork.WeightDistribution.Weight(peer.ID))

		// if float64(config.ReleventValidatorWeight)*weightOfPeer <= largestWeight {
		// 	continue
		// }

		relevantValidators++

		// Weight: 100, 20, 1
		// TPS: 1000
		// Sleep time: 121/100000, 121/20000, 121/1000
		// Issuing message count per second: 100000/121 + 20000/121 + 1000/121 = 1000

		// Each peer should send messages according to their mana: Fix TPS for example 1000;
		// A node with a x% of mana will issue 1000*x% messages per second
		issuingPeriod := float64(config.NodesTotalWeight) / float64(config.TPS) / weightOfPeer
		log.Debug(peer.ID, " issuing period is ", issuingPeriod)
		pace := time.Duration(issuingPeriod * decelerationFactor * float64(time.Second))
		log.Debug(peer.ID, " peer sent a meesage at ", pace, ". weight of peer is ", weightOfPeer)
		go startSecurityWorker(peer, pace)
	}
}

func startSecurityWorker(peer *network.Peer, pace time.Duration) {
	for range time.Tick(pace) {
		sendMessage(peer)
	}
}

func sendMessage(peer *network.Peer, optionalColor ...multiverse.Color) {
	atomic.AddUint64(&tpsCounter, 1)

	if len(optionalColor) >= 1 {
		peer.Node.(*multiverse.Node).IssuePayload(optionalColor[0])
	}

	peer.Node.(*multiverse.Node).IssuePayload(multiverse.UndefinedColor)
}

// Max returns the larger of x or y.
func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}
