package simulation

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/logger"
)

var log = logger.New("Simulation")

// ParseFlags the flags and update the configuration
func ParseFlags() {

	// Define the configuration flags
	nodesCountPtr :=
		flag.Int("nodesCount", config.NodesCount, "The number of nodes")
	nodesTotalWeightPtr :=
		flag.Int("nodesTotalWeight", config.NodesTotalWeight, "The total weight of nodes")
	zipfParameterPtr :=
		flag.Float64("zipfParameter", config.ZipfParameter, "The zipf's parameter")
	confirmationThresholdPtr :=
		flag.Float64("confirmationThreshold", config.ConfirmationThreshold, "The confirmationThreshold of confirmed messages/color")
	confirmationThresholdAbsolutePtr :=
		flag.Bool("confirmationThresholdAbsolute", config.ConfirmationThresholdAbsolute, "If set to false, the weight is counted by subtracting AW of the two largest conflicting branches.")
	parentsCountPtr :=
		flag.Int("parentsCount", config.ParentsCount, "The parents count for a message")
	weakTipsRatioPtr :=
		flag.Float64("weakTipsRatio", config.WeakTipsRatio, "The ratio of weak tips")
	tsaPtr :=
		flag.String("tsa", config.TSA, "The tip selection algorithm")
	schedulingRate :=
		flag.Int("schedulingRate", config.SchedulingRate, "The scheduling rate of the scheduler in message per second.")
	issuingRatePtr :=
		flag.Int("issuingRate", config.IssuingRate, "the tips per seconds")
	slowdownFactorPtr :=
		flag.Int("slowdownFactor", config.SlowdownFactor, "The factor to control the speed in the simulation")
	consensusMonitorTickPtr :=
		flag.Int("consensusMonitorTick", config.MetricsMonitorTick, "The tick to monitor the consensus, in milliseconds")
	doubleSpendDelayPtr :=
		flag.Int("doubleSpendDelay", config.DoubleSpendDelay, "Delay for issuing double spend transactions. (Seconds)")
	relevantValidatorWeightPtr :=
		flag.Int("releventValidatorWeight", config.RelevantValidatorWeight, "The node whose weight * RelevantValidatorWeight <= largestWeight will not issue messages")
	packetLoss :=
		flag.Float64("packetLoss", config.PacketLoss, "The packet loss percentage")
	minDelay :=
		flag.Int("minDelay", config.MinDelay, "The minimum network delay in ms")
	maxDelay :=
		flag.Int("maxDelay", config.MaxDelay, "The maximum network delay in ms")
	deltaURTS :=
		flag.Float64("deltaURTS", config.DeltaURTS, "in seconds, reference: https://iota.cafe/t/orphanage-with-restricted-urts/1199")
	simulationStopThreshold :=
		flag.Float64("simulationStopThreshold", config.SimulationStopThreshold, "Stop the simulation when >= SimulationStopThreshold * NodesCount have reached the same opinion")
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
		flag.String("simulationMode", config.SimulationMode, "Mode for the DS simulations one of: 'Accidental' - accidental double spends sent by max, min or random weight node from Zipf distrib, 'Adversary' - need to use adversary groups (parameters starting with 'Adversary...')")
	accidentalMana :=
		flag.String("accidentalMana", "", "Defines node which will be used: min, max or random")
	adversarySpeedup :=
		flag.String("adversarySpeedup", "", "Adversary issuing speed relative to their mana, e.g. '10 10' means that nodes in each group will issue 10 times messages than would be allowed by their mana. SimulationTarget must be 'DS'")
	adversaryPeeringAll :=
		flag.Bool("adversaryPeeringAll", config.AdversaryPeeringAll, "Flag indicating whether adversary nodes should be able to gossip messages to all nodes in the network directly, or should follow the peering algorithm.")
	burnPolicies :=
		flag.String("burnPolicies", "", "Space seperated list of policies employed by nodes, e.g., '0 1' . Options include: 0 = noburn, 1 = anxious, 2 = greedy, 3 = random_greedy")
	scriptStartTime :=
		flag.String("scriptStartTime", config.ScriptStartTimeStr, "Time the external script started, to be used for results directory.")

	// Parse the flags
	flag.Parse()

	// Update the configuration parameters
	config.NodesCount = *nodesCountPtr
	config.NodesTotalWeight = *nodesTotalWeightPtr
	config.ZipfParameter = *zipfParameterPtr
	config.ConfirmationThreshold = *confirmationThresholdPtr
	config.ConfirmationThresholdAbsolute = *confirmationThresholdAbsolutePtr
	config.ParentsCount = *parentsCountPtr
	config.WeakTipsRatio = *weakTipsRatioPtr
	config.TSA = *tsaPtr
	config.IssuingRate = *issuingRatePtr
	config.SlowdownFactor = *slowdownFactorPtr
	config.MetricsMonitorTick = *consensusMonitorTickPtr
	config.RelevantValidatorWeight = *relevantValidatorWeightPtr
	config.DoubleSpendDelay = *doubleSpendDelayPtr
	config.PacketLoss = *packetLoss
	config.MinDelay = *minDelay
	config.MaxDelay = *maxDelay
	config.DeltaURTS = *deltaURTS
	config.SimulationStopThreshold = *simulationStopThreshold
	config.ResultDir = *resultDirPtr
	config.IMIF = *imif
	config.RandomnessWS = *randomnessWS
	config.NeighbourCountWS = *neighbourCountWS
	config.SimulationMode = *simulationMode
	config.SchedulingRate = *schedulingRate
	parseBurnPolicies(*burnPolicies)
	config.ScriptStartTimeStr = *scriptStartTime
	parseAccidentalConfig(accidentalMana)
	parseAdversaryConfig(adversaryDelays, adversaryTypes, adversaryMana, adversaryNodeCounts, adversaryInitColors, adversaryPeeringAll, adversarySpeedup)
	log.Info("Current configuration:")
	log.Info("NodesCount: ", config.NodesCount)
	log.Info("NodesTotalWeight: ", config.NodesTotalWeight)
	log.Info("ZipfParameter: ", config.ZipfParameter)
	log.Info("ConfirmationThreshold: ", config.ConfirmationThreshold)
	log.Info("ConfirmationThresholdAbsolute: ", config.ConfirmationThresholdAbsolute)
	log.Info("ParentsCount: ", config.ParentsCount)
	log.Info("WeakTipsRatio: ", config.WeakTipsRatio)
	log.Info("TSA: ", config.TSA)
	log.Info("SchedulingRate:", config.SchedulingRate)
	log.Info("IssuingRate: ", config.IssuingRate)
	log.Info("SlowdownFactor: ", config.SlowdownFactor)
	log.Info("MetricsMonitorTick: ", config.MetricsMonitorTick)
	log.Info("RelevantValidatorWeight: ", config.RelevantValidatorWeight)
	log.Info("DoubleSpendDelay: ", config.DoubleSpendDelay)
	log.Info("PacketLoss: ", config.PacketLoss)
	log.Info("MinDelay: ", config.MinDelay)
	log.Info("MaxDelay: ", config.MaxDelay)
	log.Info("DeltaURTS:", config.DeltaURTS)
	log.Info("SimulationStopThreshold:", config.SimulationStopThreshold)
	log.Info("ResultDir:", config.ResultDir)
	log.Info("IMIF: ", config.IMIF)
	log.Info("WattsStrogatzRandomness: ", config.RandomnessWS)
	log.Info("WattsStrogatzNeighborCount: ", config.NeighbourCountWS)
	log.Info("SimulationMode: ", config.SimulationMode)
	log.Info("AdversaryTypes: ", config.AdversaryTypes)
	log.Info("AdversaryInitColors: ", config.AdversaryInitColors)
	log.Info("AdversaryMana: ", config.AdversaryMana)
	log.Info("AdversaryNodeCounts: ", config.AdversaryNodeCounts)
	log.Info("AdversaryDelays: ", config.AdversaryDelays)
	log.Info("AccidentalMana: ", config.AccidentalMana)
	log.Info("AdversaryPeeringAll: ", config.AdversaryPeeringAll)
	log.Info("AdversarySpeedup: ", config.AdversarySpeedup)

}

func parseBurnPolicies(burnPolicies string) {
	if burnPolicies == "" {
		return
	}
	policiesInt := parseStrToInt(burnPolicies)
	if len(policiesInt) == config.NodesCount {
		config.BurnPolicies = policiesInt
	} else {
		config.BurnPolicies = config.RandomArrayFromValues(0, policiesInt, config.NodesCount)
	}
}

func parseAdversaryConfig(adversaryDelays, adversaryTypes, adversaryMana, adversaryNodeCounts, adversaryInitColors *string, adversaryPeeringAll *bool, adversarySpeedup *string) {
	if config.SimulationMode != "Adversary" {
		config.AdversaryTypes = []int{}
		config.AdversaryNodeCounts = []int{}
		config.AdversaryMana = []float64{}
		config.AdversaryDelays = []int{}
		config.AdversaryInitColors = []string{}
		config.AdversarySpeedup = []float64{}

		return
	}

	config.AdversaryPeeringAll = *adversaryPeeringAll

	if *adversaryDelays != "" {
		config.AdversaryDelays = parseStrToInt(*adversaryDelays)
	}
	if *adversaryTypes != "" {
		config.AdversaryTypes = parseStrToInt(*adversaryTypes)
	}
	if *adversaryMana != "" {
		config.AdversaryMana = parseStrToFloat64(*adversaryMana)
	}
	if *adversaryNodeCounts != "" {
		config.AdversaryNodeCounts = parseStrToInt(*adversaryNodeCounts)
	}
	if *adversaryInitColors != "" {
		config.AdversaryInitColors = parseStr(*adversaryInitColors)
	}
	if *adversarySpeedup != "" {
		config.AdversarySpeedup = parseStrToFloat64(*adversarySpeedup)
	}
	// no adversary if colors are not provided
	if len(config.AdversaryInitColors) != len(config.AdversaryTypes) {
		config.AdversaryTypes = []int{}
	}

	// make sure mana, nodeCounts and delays are only defined when adversary type is provided and have the same length
	if len(config.AdversaryDelays) != 0 && len(config.AdversaryDelays) != len(config.AdversaryTypes) {
		log.Warnf("The AdversaryDelays count is not equal to the AdversaryTypes count!")
		config.AdversaryDelays = []int{}
	}
	if len(config.AdversaryMana) != 0 && len(config.AdversaryMana) != len(config.AdversaryTypes) {
		log.Warnf("The AdversaryMana count is not equal to the AdversaryTypes count!")
		config.AdversaryMana = []float64{}
	}
	if len(config.AdversaryNodeCounts) != 0 && len(config.AdversaryNodeCounts) != len(config.AdversaryTypes) {
		log.Warnf("The AdversaryNodeCounts count is not equal to the AdversaryTypes count!")
		config.AdversaryNodeCounts = []int{}
	}
}

func parseAccidentalConfig(accidentalMana *string) {
	if config.SimulationMode != "Accidental" {
		config.AccidentalMana = []string{}
		return
	}
	if *accidentalMana != "" {
		config.AccidentalMana = parseStr(*accidentalMana)
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

func DumpConfig(fileName string) {
	type Configuration struct {
		NodesCount, NodesTotalWeight, ParentsCount, SchedulingRate, IssuingRate, ConsensusMonitorTick, RelevantValidatorWeight, MinDelay, MaxDelay, SlowdownFactor, DoubleSpendDelay, NeighbourCountWS, MaxBuffer int
		ZipfParameter, WeakTipsRatio, PacketLoss, DeltaURTS, SimulationStopThreshold, RandomnessWS                                                                                                                float64
		ConfirmationThreshold, TSA, ResultDir, IMIF, SimulationTarget, SimulationMode, BurnPolicyNames                                                                                                            string
		AdversaryDelays, AdversaryTypes, AdversaryNodeCounts                                                                                                                                                      []int
		AdversarySpeedup, AdversaryMana                                                                                                                                                                           []float64
		AdversaryInitColor, AccidentalMana                                                                                                                                                                        []string
		AdversaryPeeringAll, ConfEligible                                                                                                                                                                         bool
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
		ConsensusMonitorTick:    config.MetricsMonitorTick,
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
		ConfEligible:            config.ConfEligible,
		MaxBuffer:               config.MaxBuffer,
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
	if os.WriteFile(path.Join(config.ResultDir, fileName), bytes, 0644) != nil {
		log.Error(err)
	}
}
