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

	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/hive.go/typeutils"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/logger"
	"github.com/iotaledger/multivers-simulation/multiverse"
	"github.com/iotaledger/multivers-simulation/network"
)

var (
	log      = logger.New("Simulation")
	awHeader = []string{"Message ID", "Issuance Time (unix)", "Confirmation Time (ns)", "Weight", "# of Confirmed Messages"}
	dsHeader = []string{"Red", "Blue"}
	csvMutex sync.Mutex
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
}

func main() {
	log.Info("Starting simulation ... [DONE]")
	defer log.Info("Shutting down simulation ... [DONE]")

	parseFlags()
	testNetwork := network.New(
		network.Nodes(config.NodesCount, multiverse.NewNode, network.ZIPFDistribution(config.ZipfParameter, float64(config.NodesTotalWeight))),
		network.Delay(time.Duration(config.DecelerationFactor)*30*time.Microsecond, time.Duration(config.DecelerationFactor)*250*time.Microsecond),
		network.PacketLoss(0, 0.05),
		network.Topology(network.WattsStrogatz(4, 1)),
	)
	testNetwork.Start()
	defer testNetwork.Shutdown()

	resultsWriters := monitorNetworkState(testNetwork)
	defer flushWriters(resultsWriters)
	secureNetwork(testNetwork, config.DecelerationFactor)

	time.Sleep(2 * time.Second)

	sendMessage(testNetwork.Peers[0], multiverse.Blue)
	sendMessage(testNetwork.Peers[0], multiverse.Red)

	time.Sleep(30 * time.Second)
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

	confirmedMessageCounter = int64(0)

	opinionMutex sync.Mutex

	relevantValidators int
)

func dumpConfig(fileName string) {
	type Configuration struct {
		NodesCount, NodesTotalWeight, TipsCount, TPS, ConsensusMonitorTick, ReleventValidatorWeight int
		ZipfParameter, MessageWeightThreshold, WeakTipsRatio, DecelerationFactor                    float64
		TSA                                                                                         string
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

	// The simulation start time
	simulationStartTime := time.Now().UTC().Format(time.RFC3339)

	// Dump the configuration of this simulation
	dumpConfig(fmt.Sprint("aw-", simulationStartTime, ".config"))

	// Dump the double spending result

	// Define the file name of the aw results
	fileName := fmt.Sprint("ds-", simulationStartTime, ".result")
	file, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	dsResultsWriter := csv.NewWriter(file)
	resultsWriters = append(resultsWriters, dsResultsWriter)
	if err = dsResultsWriter.Write(dsHeader); err != nil {
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
			events.NewClosure(func(message *multiverse.Message, messageMetadata *multiverse.MessageMetadata, weight uint64) {
				atomic.AddInt64(&confirmedMessageCounter, 1)

				record := []string{
					strconv.FormatInt(int64(message.ID), 10),
					strconv.FormatInt(message.IssuanceTime.Unix(), 10),
					strconv.FormatInt(int64(messageMetadata.ConfirmationTime().Sub(message.IssuanceTime)), 10),
					strconv.FormatUint(weight, 10),
					strconv.FormatInt(confirmedMessageCounter, 10),
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
	}

	go func() {
		for range time.Tick(time.Duration(config.ConsensusMonitorTick) * time.Millisecond) {
			log.Infof("Network Status: %d TPS :: Consensus[ %d Undefined / %d Blue / %d Red / %d Green ] :: %d Nodes :: %d Validators",
				atomic.LoadUint64(&tpsCounter),
				opinions[multiverse.UndefinedColor],
				opinions[multiverse.Blue],
				opinions[multiverse.Red],
				opinions[multiverse.Green],
				config.NodesCount,
				relevantValidators,
			)

			record := []string{
				strconv.FormatInt(int64(opinions[multiverse.Blue]), 10),
				strconv.FormatInt(int64(opinions[multiverse.Red]), 10),
			}

			if err := dsResultsWriter.Write(record); err != nil {
				log.Fatal("error writing record to csv:", err)
			}

			if err := dsResultsWriter.Error(); err != nil {
				log.Fatal(err)
			}

			atomic.StoreUint64(&tpsCounter, 0)
		}
	}()

	return
}

func secureNetwork(testNetwork *network.Network, decelerationFactor float64) {
	largestWeight := float64(testNetwork.WeightDistribution.LargestWeight())

	for _, peer := range testNetwork.Peers {
		weightOfPeer := float64(testNetwork.WeightDistribution.Weight(peer.ID))

		if float64(config.ReleventValidatorWeight)*weightOfPeer <= largestWeight {
			continue
		}

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
