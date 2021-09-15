package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/iotaledger/multivers-simulation/simulation"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
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
	awHeader = []string{"Message ID", "Issuance Time (unix)", "Confirmation Time (ns)", "Weight", "# of Confirmed Messages", "# of Issued Messages", "ns since start"}
	dsHeader = []string{"UndefinedColor", "Blue", "Red", "Green", "ns since start", "ns since issuance"}
	tpHeader = []string{"UndefinedColor (Tip Pool Size)", "Blue (Tip Pool Size)", "Red (Tip Pool Size)", "Green (Tip Pool Size)",
		"UndefinedColor (Processed)", "Blue (Processed)", "Red (Processed)", "Green (Processed)", "# of Issued Messages", "ns since start"}
	ccHeader = []string{"Blue (Confirmed)", "Red (Confirmed)", "Green (Confirmed)", "Blue (Like)", "Red (Like)", "Green (Like)", "Unconfirmed Blue", "Unconfirmed Red", "Unconfirmed Green", "Flips (Winning color changed)", "ns since start", "ns since issuance"}

	csvMutex              sync.Mutex
	shutdownSignal        = make(chan types.Empty)
	maxSimulationDuration = time.Minute

	tpsCounter = uint64(0)

	opinions = make(map[multiverse.Color]int)

	opinionsWeights = make(map[multiverse.Color]int64)

	tipPoolSizes = make(map[multiverse.Color]int)

	processedMessageCounts = make(map[multiverse.Color]uint64)

	issuedMessageCounter = int64(0)

	confirmedNodesCounter   = make(map[multiverse.Color]int)
	colorUnconfirmedCounter = make(map[multiverse.Color]int)

	confirmedMessageCounter = make(map[network.PeerID]int64)
	confirmedMessageMutex   sync.RWMutex

	opinionMutex sync.RWMutex

	confirmationMutex sync.RWMutex

	opinionWeightMutex sync.RWMutex

	processedMessageMutex sync.RWMutex

	relevantValidators int

	dsIssuanceTime time.Time

	mostLikedColor multiverse.Color

	flipsCounter int
)

func main() {
	log.Info("Starting simulation ... [DONE]")
	defer log.Info("Shutting down simulation ... [DONE]")
	simulation.ParseFlags()
	testNetwork := network.New(
		network.Nodes(config.NodesCount, network.NodeClosure(multiverse.NewNode), network.ZIPFDistribution(
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
		attackers := testNetwork.RandomPeers(3)
		dsIssuanceTime = time.Now()
		sendMessage(attackers[0], multiverse.Blue)
		sendMessage(attackers[1], multiverse.Red)
	}

	// Here three peers are randomly selected with defined Colors
	// attackers := testNetwork.RandomPeers(3)
	// sendMessage(attackers[0], multiverse.Red)
	// sendMessage(attackers[1], multiverse.Blue)
	// sendMessage(attackers[2], multiverse.Green)

	select {
	case <-shutdownSignal:
		log.Info("Shutting down simulation (consensus reached) ... [DONE]")
	case <-time.After(time.Duration(config.DecelerationFactor) * maxSimulationDuration):
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

func dumpConfig(fileName string) {
	type Configuration struct {
		NodesCount, NodesTotalWeight, TipsCount, TPS, ConsensusMonitorTick, ReleventValidatorWeight, MinDelay, MaxDelay, DecelerationFactor, DoubleSpendDelay, NeighbourCountWS int
		ZipfParameter, MessageWeightThreshold, WeakTipsRatio, PayloadLoss, DeltaURTS, SimulationStopThreshold, RandomnessWS                                                     float64
		TSA, ResultDir, IMIF, SimulationTarget                                                                                                                                  string
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
		ReleventValidatorWeight: config.RelevantValidatorWeight,
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
	opinions[multiverse.UndefinedColor] = config.NodesCount
	opinions[multiverse.Blue] = 0
	opinions[multiverse.Red] = 0
	opinions[multiverse.Green] = 0
	opinionsWeights[multiverse.UndefinedColor] = 0
	opinionsWeights[multiverse.Blue] = 0
	opinionsWeights[multiverse.Red] = 0
	opinionsWeights[multiverse.Green] = 0

	mostLikedColor = multiverse.UndefinedColor

	// The simulation start time
	simulationStartTime := time.Now()
	simulationStartTimeStr := simulationStartTime.UTC().Format(time.RFC3339)

	// Dump the configuration of this simulation
	dumpConfig(fmt.Sprint("aw-", simulationStartTimeStr, ".config"))

	// Dump the double spending result
	dsResultsWriter := createWriter(fmt.Sprintf("ds-%s.csv", simulationStartTimeStr))

	// Dump the tip pool and processed message (throughput) results
	tpResultsWriter := createWriter(fmt.Sprintf("tp-%s.csv", simulationStartTimeStr))

	// Dump the info about how many nodes have confirmed and liked a certain color
	ccResultsWriter := createWriter(fmt.Sprintf("cc-%s.csv", simulationStartTimeStr))

	// Check the result writers
	resultsWriters = append(resultsWriters, dsResultsWriter)
	resultsWriters = append(resultsWriters, tpResultsWriter)
	resultsWriters = append(resultsWriters, ccResultsWriter)

	// Write the headers
	if err := dsResultsWriter.Write(dsHeader); err != nil {
		panic(err)
	}
	if err := tpResultsWriter.Write(tpHeader); err != nil {
		panic(err)
	}
	if err := ccResultsWriter.Write(ccHeader); err != nil {
		panic(err)
	}

	for _, id := range config.MonitoredAWPeers {
		awPeer := testNetwork.Peers[id]
		if typeutils.IsInterfaceNil(awPeer) {
			panic(fmt.Sprintf("unknowm peer with id %d", id))
		}
		// Define the file name of the aw results
		awResultsWriter := createWriter(fmt.Sprintf("aw%d-%s.csv", id, simulationStartTimeStr))
		if err := awResultsWriter.Write(awHeader); err != nil {
			panic(err)
		}
		resultsWriters = append(resultsWriters, awResultsWriter)
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
		peer.Node.(multiverse.NodeInterface).Tangle().OpinionManager.Events().OpinionChanged.Attach(events.NewClosure(func(oldOpinion multiverse.Color, newOpinion multiverse.Color) {
			opinionMutex.Lock()
			defer opinionMutex.Unlock()

			opinions[oldOpinion]--
			opinions[newOpinion]++
			if mostLikedColorChanged() {
				flipsCounter++
			}

		}))
		peer.Node.(multiverse.NodeInterface).Tangle().OpinionManager.Events().ColorConfirmed.Attach(events.NewClosure(func(confirmedColor multiverse.Color) {
			confirmationMutex.Lock()
			defer confirmationMutex.Unlock()

			confirmedNodesCounter[confirmedColor]++
		}))

		peer.Node.(multiverse.NodeInterface).Tangle().OpinionManager.Events().ColorUnconfirmed.Attach(events.NewClosure(func(unconfirmedColor multiverse.Color) {
			confirmationMutex.Lock()
			defer confirmationMutex.Unlock()

			colorUnconfirmedCounter[unconfirmedColor]++
			confirmedNodesCounter[unconfirmedColor]--
		}))
	}

	// Here we only monitor the opinion weight of node w/ the highest weight
	dsPeer := testNetwork.Peers[0]
	dsPeer.Node.(multiverse.NodeInterface).Tangle().OpinionManager.Events().ApprovalWeightUpdated.Attach(events.NewClosure(func(opinion multiverse.Color, deltaWeight int64) {
		opinionWeightMutex.Lock()
		defer opinionWeightMutex.Unlock()

		opinionsWeights[opinion] += deltaWeight
	}))

	// Here we only monitor the tip pool size of node w/ the highest weight
	peer := testNetwork.Peers[0]
	peer.Node.(multiverse.NodeInterface).Tangle().TipManager.Events.MessageProcessed.Attach(events.NewClosure(
		func(opinion multiverse.Color, tipPoolSize int, processedMessages uint64, issuedMessages int64) {
			processedMessageMutex.Lock()
			defer processedMessageMutex.Unlock()

			tipPoolSizes[opinion] = tipPoolSize
			processedMessageCounts[opinion] = processedMessages
			atomic.StoreInt64(&issuedMessageCounter, issuedMessages)
		}))

	go func() {
		for range time.Tick(time.Duration(config.DecelerationFactor*config.ConsensusMonitorTick) * time.Millisecond) {
			log.Infof("Opinions[ %3d Undefined / %3d Blue / %3d Red / %3d Green ]",
				opinions[multiverse.UndefinedColor],
				opinions[multiverse.Blue],
				opinions[multiverse.Red],
				opinions[multiverse.Green],
			)
			log.Infof("Network Status: %3d TPS :: Consensus[ %3d Undefined / %3d Blue / %3d Red / %3d Green ] :: %d Nodes :: %d Validators",
				atomic.LoadUint64(&tpsCounter),
				confirmedNodesCounter[multiverse.UndefinedColor],
				confirmedNodesCounter[multiverse.Blue],
				confirmedNodesCounter[multiverse.Red],
				confirmedNodesCounter[multiverse.Green],
				config.NodesCount,
				relevantValidators,
			)
			opinionWeightMutex.RLock()
			processedMessageMutex.RLock()
			confirmationMutex.RLock()
			opinionMutex.RLock()

			sinceIssuance := "0"
			if !dsIssuanceTime.IsZero() {
				sinceIssuance = strconv.FormatInt(time.Since(dsIssuanceTime).Nanoseconds(), 10)

			}

			// Dump the double spending results
			record := []string{
				strconv.FormatInt(opinionsWeights[multiverse.UndefinedColor], 10),
				strconv.FormatInt(opinionsWeights[multiverse.Blue], 10),
				strconv.FormatInt(opinionsWeights[multiverse.Red], 10),
				strconv.FormatInt(opinionsWeights[multiverse.Green], 10),
				strconv.FormatInt(time.Since(simulationStartTime).Nanoseconds(), 10),
				sinceIssuance,
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
				strconv.FormatInt(atomic.LoadInt64(&issuedMessageCounter), 10),
				strconv.FormatInt(time.Since(simulationStartTime).Nanoseconds(), 10),
			}

			if err := tpResultsWriter.Write(record); err != nil {
				log.Fatal("error writing record to csv:", err)
			}

			if err := tpResultsWriter.Error(); err != nil {
				log.Fatal(err)
			}

			// Dump the opinion and confirmation counters
			record = []string{
				strconv.FormatInt(int64(confirmedNodesCounter[multiverse.Blue]), 10),
				strconv.FormatInt(int64(confirmedNodesCounter[multiverse.Red]), 10),
				strconv.FormatInt(int64(confirmedNodesCounter[multiverse.Green]), 10),
				strconv.FormatInt(int64(opinions[multiverse.Blue]), 10),
				strconv.FormatInt(int64(opinions[multiverse.Red]), 10),
				strconv.FormatInt(int64(opinions[multiverse.Green]), 10),
				strconv.FormatInt(int64(colorUnconfirmedCounter[multiverse.Blue]), 10),
				strconv.FormatInt(int64(colorUnconfirmedCounter[multiverse.Red]), 10),
				strconv.FormatInt(int64(colorUnconfirmedCounter[multiverse.Green]), 10),
				strconv.FormatInt(int64(flipsCounter), 10),
				strconv.FormatInt(time.Since(simulationStartTime).Nanoseconds(), 10),
				sinceIssuance,
			}

			if err := ccResultsWriter.Write(record); err != nil {
				log.Fatal("error writing record to csv:", err)
			}

			if err := ccResultsWriter.Error(); err != nil {
				log.Fatal(err)
			}

			if Max(Max(confirmedNodesCounter[multiverse.Blue], confirmedNodesCounter[multiverse.Red]), confirmedNodesCounter[multiverse.Green]) >= int(config.SimulationStopThreshold*float64(config.NodesCount)) {
				shutdownSignal <- types.Void
			}
			atomic.StoreUint64(&tpsCounter, 0)
			opinionWeightMutex.RUnlock()
			processedMessageMutex.RUnlock()
			confirmationMutex.RUnlock()
			opinionMutex.RUnlock()
		}
	}()

	return
}

func createWriter(fileName string) *csv.Writer {
	file, err := os.Create(path.Join(config.ResultDir, fileName))
	if err != nil {
		panic(err)
	}
	tpResultsWriter := csv.NewWriter(file)
	return tpResultsWriter
}

func secureNetwork(testNetwork *network.Network) {
	// In the simulation we let all nodes can send messages.
	// largestWeight := float64(testNetwork.WeightDistribution.LargestWeight())

	for _, peer := range testNetwork.Peers {
		weightOfPeer := float64(testNetwork.WeightDistribution.Weight(peer.ID))

		// if float64(config.RelevantValidatorWeight)*weightOfPeer <= largestWeight {
		// 	continue
		// }

		relevantValidators++

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
	atomic.AddUint64(&tpsCounter, 1)

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

func mostLikedColorChanged() bool {
	currentMostLikedColor := multiverse.UndefinedColor
	if opinions[multiverse.Green] > 0 {
		currentMostLikedColor = multiverse.Green
	}
	if opinions[multiverse.Blue] > opinions[multiverse.Green] {
		currentMostLikedColor = multiverse.Blue
	}
	if opinions[multiverse.Red] > opinions[multiverse.Blue] && opinions[multiverse.Red] > opinions[multiverse.Green] {
		currentMostLikedColor = multiverse.Red
	}
	// color selected
	if mostLikedColor != currentMostLikedColor {
		// color selected for the first time, it not counts
		if mostLikedColor == multiverse.UndefinedColor {
			mostLikedColor = currentMostLikedColor
			return false
		}
		mostLikedColor = currentMostLikedColor
		return true
	}
	return false
}
