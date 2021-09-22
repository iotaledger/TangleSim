package simulation

import (
	"flag"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/logger"
	"strconv"
	"strings"
)

var log = logger.New("Simulation")

// Parse the flags and update the configuration
func ParseFlags() {

	// Define the configuration flags
	nodesCountPtr :=
		flag.Int("nodesCount", config.NodesCount, "The number of nodes")
	nodesTotalWeightPtr :=
		flag.Int("nodesTotalWeight", config.NodesTotalWeight, "The total weight of nodes")
	zipfParameterPtr :=
		flag.Float64("zipfParameter", config.ZipfParameter, "The zipf's parameter")
	weightThresholdPtr :=
		flag.Float64("weightThreshold", config.WeightThreshold, "The weightThreshold of confirmed messages/color")
	weightThresholdAbsolutePtr :=
		flag.Bool("weightThresholdAbsolute", config.WeightThresholdAbsolute, "If set to false, the weight is counted by subtracting AW of the two largest conflicting branches.")
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
		flag.String("adversaryType", "", "Defines group attack strategy, one of the following: 0 - honest node behavior, 1 - shifts opinion, 2 - keeps the same opinion. SimulationTarget must be 'DS'")
	adversaryNodeCounts :=
		flag.String("adversaryNodeCounts", "", "Defines number of adversary nodes in the group. Leave empty for default value: 1. SimulationTarget must be 'DS'")
	adversaryInitColors :=
		flag.String("adversaryInitColors", "", "Defines initial color for adversary group, one of following: 'R', 'G', 'B'. Mandatory for each group. SimulationTarget must be 'DS'")
	adversaryMana :=
		flag.String("adversaryMana", "", "Adversary nodes mana in %, e.g. '10 10' Special values: -1 nodes should be selected randomly from weight distribution, SimulationTarget must be 'DS'")
	simulationMode :=
		flag.String("simulationMode", "Accidental", "Mode for the DS simulations one of: 'Accidental' - accidental double spends sent by max, min or random weight node from Zipf distrib, 'Adversary' - need to use adversary groups (parameters starting with 'Adversary...')")
	accidentalMana :=
		flag.String("accidentalMana", "", "Defines node which will be used: min, max or random")

	// Parse the flags
	flag.Parse()

	// Update the configuration parameters
	config.NodesCount = *nodesCountPtr
	config.NodesTotalWeight = *nodesTotalWeightPtr
	config.ZipfParameter = *zipfParameterPtr
	config.WeightThreshold = *weightThresholdPtr
	config.WeightThresholdAbsolute = *weightThresholdAbsolutePtr
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
	config.SimulationMode = *simulationMode
	parseAccidentalConfig(accidentalMana)
	parseAdversaryConfig(adversaryDelays, adversaryTypes, adversaryMana, adversaryNodeCounts, adversaryInitColors)
	log.Info("Current configuration:")
	log.Info("NodesCount: ", config.NodesCount)
	log.Info("NodesTotalWeight: ", config.NodesTotalWeight)
	log.Info("ZipfParameter: ", config.ZipfParameter)
	log.Info("WeightThreshold: ", config.WeightThreshold)
	log.Info("WeightThresholdAbsolute: ", config.WeightThresholdAbsolute)
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
}

func parseAdversaryConfig(adversaryDelays, adversaryTypes, adversaryMana, adversaryNodeCounts, adversaryInitColors *string) {
	if config.SimulationMode != "Adversary" {
		config.AdversaryTypes = []int{}
		config.AdversaryNodeCounts = []int{}
		config.AdversaryMana = []float64{}
		config.AdversaryDelays = []int{}
		config.AdversaryInitColors = []string{}
		return
	}

	if *adversaryDelays != "" {
		config.AdversaryDelays = parseStrToInt(*adversaryDelays)
	}
	if *adversaryTypes != "" {
		config.AdversaryTypes = parseStrToInt(*adversaryTypes)
	}
	if *adversaryMana != "" {
		config.AdversaryMana = parseStrToFloat64(*adversaryMana)
	}
	if *adversaryMana != "" {
		config.AdversaryNodeCounts = parseStrToInt(*adversaryNodeCounts)
	}
	if *adversaryMana != "" {
		config.AdversaryInitColors = parseStr(*adversaryInitColors)
	}

	// no adversary if simulation target is not DS
	if config.SimulationTarget != "DS" {
		config.AdversaryTypes = []int{}
	}

	// no adversary if colors are not provided
	if len(config.AdversaryNodeCounts) != len(config.AdversaryTypes) {
		config.AdversaryTypes = []int{}
	}

	// make sure mana, nodeCounts and delays are only defined when adversary type is provided and have the same length
	if len(config.AdversaryDelays) != 0 && len(config.AdversaryDelays) != len(config.AdversaryTypes) {
		config.AdversaryDelays = []int{}
	}
	if len(config.AdversaryMana) != 0 && len(config.AdversaryMana) != len(config.AdversaryTypes) {
		config.AdversaryMana = []float64{}
	}
	if len(config.AdversaryNodeCounts) != 0 && len(config.AdversaryNodeCounts) != len(config.AdversaryTypes) {
		config.AdversaryNodeCounts = []int{}
	}
}

func parseAccidentalConfig(accidentalMana *string) {
	if config.SimulationMode != "Accidental" {
		config.AdversaryInitColors = []string{}
		return
	}
	if *accidentalMana != "" {
		config.AdversaryInitColors = parseStr(*accidentalMana)
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

func parseStr(strList string) []string {
	split := strings.Split(strList, " ")
	return split
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
