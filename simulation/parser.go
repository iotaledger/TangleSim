package simulation

import (
	"flag"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/labstack/gommon/log"
	"strconv"
	"strings"
)

// Parse the flags and update the configuration
func ParseFlags() {

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
		flag.Int("decelerationFactor", config.DecelerationFactor, "The factor to control the speed in the simulation")
	consensusMonitorTickPtr :=
		flag.Int("consensusMonitorTick", config.ConsensusMonitorTick, "The tick to monitor the consensus, in milliseconds")
	doubleSpendDelayPtr :=
		flag.Int("doubleSpendDelay", config.DoubleSpendDelay, "Delay for issuing double spend transactions. (Seconds)")
	relevantValidatorWeightPtr :=
		flag.Int("releventValidatorWeight", config.RelevantValidatorWeight, "The node whose weight * RelevantValidatorWeight <= largestWeight will not issue messages")
	payloadLoss :=
		flag.Float64("payloadLoss", config.PayloadLoss, "The payload loss percentage")
	minDelay :=
		flag.Int("minDelay", config.MinDelay, "The minimum network delay in ms")
	maxDelay :=
		flag.Int("maxDelay", config.MaxDelay, "The maximum network delay in ms")
	deltaURTS :=
		flag.Float64("deltaURTS", config.DeltaURTS, "in seconds, reference: https://iota.cafe/t/orphanage-with-restricted-urts/1199")
	simulationStopThreshold :=
		flag.Float64("simulationStopThreshold", config.SimulationStopThreshold, "Stop the simulation when >= SimulationStopThreshold * NodesCount have reached the same opinion")
	simulationTarget :=
		flag.String("simulationTarget", config.SimulationTarget, "The simulation target, CT: Confirmation Time, DS: Double Spending")
	resultDirPtr :=
		flag.String("resultDir", config.ResultDir, "Directory where the results will be stored")
	imif :=
		flag.String("IMIF", config.IMIF, "Inter Message Issuing Function for time delay between activity messages: poisson or uniform")
	randomnessWS :=
		flag.Float64("WattsStrogatzRandomness", config.RandomnessWS, "WattsStrogatz randomness parameter")
	neighbourCountWS :=
		flag.Int("WattsStrogatzNeighborCount", config.NeighbourCountWS, "Number of neighbors node is connected to in WattsStrogatz network topology")
	adversaryDelays :=
		flag.String("adversaryDelays", "", "Delays in ms of adversary nodes, eg '50 100 200'")
	adversaryTypes :=
		flag.String("adversaryType", "", "Defines attack strategy, one of the following: 'shift', 'same'")
	adversaryMana :=
		flag.String("adversaryMana", "", "Adversary nodes mana in %, e.g. '10 10' or 'random' if adversary nodes should be selected randomly")
	adversaryErrorThreshold :=
		flag.Float64("payloadLoss", config.AdversaryErrorThreshold, "The error threshold of q - percentage of mana held by adversary")

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
	config.RelevantValidatorWeight = *relevantValidatorWeightPtr
	config.DoubleSpendDelay = *doubleSpendDelayPtr
	config.PayloadLoss = *payloadLoss
	config.MinDelay = *minDelay
	config.MaxDelay = *maxDelay
	config.DeltaURTS = *deltaURTS
	config.SimulationStopThreshold = *simulationStopThreshold
	config.SimulationTarget = *simulationTarget
	config.ResultDir = *resultDirPtr
	config.IMIF = *imif
	config.RandomnessWS = *randomnessWS
	config.NeighbourCountWS = *neighbourCountWS
	parseAdversaryConfig(adversaryDelays, adversaryTypes, adversaryMana)
	config.AdversaryErrorThreshold = *adversaryErrorThreshold

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
	log.Info("RelevantValidatorWeight: ", config.RelevantValidatorWeight)
	log.Info("DoubleSpendDelay: ", config.DoubleSpendDelay)
	log.Info("PayloadLoss: ", config.PayloadLoss)
	log.Info("MinDelay: ", config.MinDelay)
	log.Info("MaxDelay: ", config.MaxDelay)
	log.Info("DeltaURTS:", config.DeltaURTS)
	log.Info("SimulationStopThreshold:", config.SimulationStopThreshold)
	log.Info("SimulationTarget:", config.SimulationTarget)
	log.Info("ResultDir:", config.ResultDir)
	log.Info("IMIF: ", config.IMIF)
	log.Info("WattsStrogatzRandomness: ", config.RandomnessWS)
	log.Info("WattsStrogatzNeighborCount: ", config.NeighbourCountWS)
	log.Info("AdversaryDelays: ", config.AdversaryDelays)
	log.Info("AdversaryTypes: ", config.AdversaryTypes)
	log.Info("AdversaryMana: ", config.AdversaryMana)
	log.Info("AdversaryErrorThreshold: ", config.AdversaryErrorThreshold)
}

func parseAdversaryConfig(adversaryDelays *string, adversaryTypes *string, adversaryMana *string) {
	if *adversaryDelays != "" {
		config.AdversaryDelays = parseStrToInt(*adversaryDelays)
	}
	if *adversaryTypes != "" {
		config.AdversaryTypes = parseStrToInt(*adversaryTypes)
	}
	if *adversaryMana != "" {
		config.AdversaryMana = parseStrToFloat64(*adversaryMana)
	}

	// no adversary if simulation target is not DS
	if config.SimulationTarget != "DS" {
		config.AdversaryTypes = []int{}
	}

	// maximum 3 adversary types allowed for one simulation, one per color
	if len(config.AdversaryTypes) > 3 {
		config.AdversaryTypes = []int{}
	}

	// make sure mana and delays are only defined when adversary type is provided and have the same length
	if len(config.AdversaryDelays) != 0 && len(config.AdversaryDelays) != len(config.AdversaryTypes) {
		config.AdversaryDelays = []int{}
	}
	if len(config.AdversaryMana) != 0 && len(config.AdversaryMana) != len(config.AdversaryTypes) {
		config.AdversaryMana = []float64{}
	}
}

func parseStrToInt(strList string) []int {
	split := strings.Split(strList, " ")
	parsed := make([]int, len(split))
	for i, elem := range split {
		num, _ := strconv.Atoi(elem)
		parsed[i] = num
	}
	return parsed
}

func parseStrToFloat64(strList string) []float64 {
	split := strings.Split(strList, " ")
	parsed := make([]float64, len(split))
	for i, elem := range split {
		num, _ := strconv.ParseFloat(elem, 64)
		parsed[i] = num
	}
	return parsed
}
