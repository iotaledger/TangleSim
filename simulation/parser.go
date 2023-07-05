package simulation

import (
	"encoding/json"
	"flag"
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
		flag.Int("nodesCount", config.Params.NodesCount, "The number of nodes")
	nodesTotalWeightPtr :=
		flag.Int("nodesTotalWeight", config.Params.NodesTotalWeight, "The total weight of nodes")
	zipfParameterPtr :=
		flag.Float64("zipfParameter", config.Params.ZipfParameter, "The zipf's parameter")
	confirmationThresholdPtr :=
		flag.Float64("confirmationThreshold", config.Params.ConfirmationThreshold, "The confirmationThreshold of confirmed messages/color")
	confirmationThresholdAbsolutePtr :=
		flag.Bool("confirmationThresholdAbsolute", config.Params.ConfirmationThresholdAbsolute, "If set to false, the weight is counted by subtracting AW of the two largest conflicting branches.")
	parentsCountPtr :=
		flag.Int("parentsCount", config.Params.ParentsCount, "The parents count for a message")
	weakTipsRatioPtr :=
		flag.Float64("weakTipsRatio", config.Params.WeakTipsRatio, "The ratio of weak tips")
	tsaPtr :=
		flag.String("tsa", config.Params.TSA, "The tip selection algorithm")
	monitoredAWPeers :=
		flag.String("monitoredAWPeers", "", "Space seperated list of nodes to monitored, e.g., '0 1'")
	monitoredWitnessWeightPeerPtr :=
		flag.Int("monitoredWitnessWeightPeer", config.Params.MonitoredWitnessWeightPeer, "The node for which we monitor the WW growth")
	monitoredWitnessWeightMessageIDPtr :=
		flag.Int("monitoredWitnessWeightMessageID", config.Params.MonitoredWitnessWeightMessageID, "The message for which we monitor the WW growth")
	simulationDurationPtr :=
		flag.Duration("simulationDuration", config.Params.SimulationDuration, "The simulation time of the experiment")
	schedulerTypePtr :=
		flag.String("schedulerType", config.Params.SchedulerType, "The type of the scheduler.")
	schedulingRate :=
		flag.Int("schedulingRate", config.Params.SchedulingRate, "The scheduling rate of the scheduler in message per second.")
	maxDeficitPtr :=
		flag.Float64("maxDeficit", config.Params.MaxDeficit, "The maximum deficit for all nodes")
	slotTimePtr :=
		flag.Duration("slotTime", config.Params.SlotTime, "The duration of a slot")
	minCommittableAgePtr :=
		flag.Duration("minCommittableAge", config.Params.MinCommittableAge, "The minimum duration to create a commitment")
	rmcTimePtr :=
		flag.Duration("rmcTime", config.Params.RMCTime, "The duration of a referenced mana cost")
	initialRMCPtr :=
		flag.Float64("initialRMC", config.Params.InitialRMC, "The initial valud of referenced mana cost")
	lowerRMCThresholdPtr :=
		flag.Float64("lowerRMCThreshold", config.Params.LowerRMCThreshold, "The lower bound of RMC threshold")
	upperRMCThresholdPtr :=
		flag.Float64("upperRMCThreshold", config.Params.UpperRMCThreshold, "The upper bound of RMC threshold")
	alphaRMCPtr :=
		flag.Float64("alphaRMC", config.Params.AlphaRMC, "The alpha RMC value")
	betaRMCPtr :=
		flag.Float64("betaRMC", config.Params.BetaRMC, "The beta RMC value")
	rmcMinPtr :=
		flag.Float64("rmcMin", config.Params.RMCmin, "The minimum RMC value")
	rmcMaxPtr :=
		flag.Float64("rmcMax", config.Params.RMCmax, "The maximum RMC value")
	rmcIncreasePtr :=
		flag.Float64("rmcIncrease", config.Params.RMCincrease, "The RMC value to increase")
	rmcDecreasePtr :=
		flag.Float64("rmcDecrease", config.Params.RMCdecrease, "The RMC value to decrease")
	rmcPeriodUpdatePtr :=
		flag.Int("rmcPeriodUpdate", config.Params.RMCPeriodUpdate, "The period to update RMC")
	issuingRatePtr :=
		flag.Int("issuingRate", config.Params.IssuingRate, "the tips per seconds")
	slowdownFactorPtr :=
		flag.Int("slowdownFactor", config.Params.SlowdownFactor, "The factor to control the speed in the simulation")
	consensusMonitorTickPtr :=
		flag.Int("consensusMonitorTick", config.Params.ConsensusMonitorTick, "The tick to monitor the consensus, in milliseconds")
	doubleSpendDelayPtr :=
		flag.Int("doubleSpendDelay", config.Params.DoubleSpendDelay, "Delay for issuing double spend transactions. (Seconds)")
	relevantValidatorWeightPtr :=
		flag.Int("releventValidatorWeight", config.Params.RelevantValidatorWeight, "The node whose weight * RelevantValidatorWeight <= largestWeight will not issue messages")
	packetLoss :=
		flag.Float64("packetLoss", config.Params.PacketLoss, "The packet loss percentage")
	minDelay :=
		flag.Int("minDelay", config.Params.MinDelay, "The minimum network delay in ms")
	maxDelay :=
		flag.Int("maxDelay", config.Params.MaxDelay, "The maximum network delay in ms")
	congestionPeriods :=
		flag.String("congestionPeriods", "", "Space seperated list of congestion to run, e.g., '0.5 1.2 0.5 1.2'")
	initialMana :=
		flag.Float64("initialMana", config.Params.InitialMana, "The initial mana")
	deltaURTS :=
		flag.Float64("deltaURTS", config.Params.DeltaURTS, "in seconds, reference: https://iota.cafe/t/orphanage-with-restricted-urts/1199")
	simulationStopThreshold :=
		flag.Float64("simulationStopThreshold", config.Params.SimulationStopThreshold, "Stop the simulation when >= SimulationStopThreshold * NodesCount have reached the same opinion")
	resultDirPtr :=
		flag.String("resultDir", config.Params.ResultDir, "Directory where the results will be stored")
	imif :=
		flag.String("IMIF", config.Params.IMIF, "Inter Message Issuing Function for time delay between activity messages: poisson or uniform")
	randomnessWS :=
		flag.Float64("WattsStrogatzRandomness", config.Params.RandomnessWS, "WattsStrogatz randomness parameter")
	neighbourCountWS :=
		flag.Int("WattsStrogatzNeighborCount", config.Params.NeighbourCountWS, "Number of neighbors node is connected to in WattsStrogatz network topology")
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
		flag.String("simulationMode", config.Params.SimulationMode, "Mode for the DS simulations one of: 'Accidental' - accidental double spends sent by max, min or random weight node from Zipf distrib, 'Adversary' - need to use adversary groups (parameters starting with 'Adversary...')")
	accidentalMana :=
		flag.String("accidentalMana", "", "Defines node which will be used: min, max or random")
	adversarySpeedup :=
		flag.String("adversarySpeedup", "", "Adversary issuing speed relative to their mana, e.g. '10 10' means that nodes in each group will issue 10 times messages than would be allowed by their mana. SimulationTarget must be 'DS'")
	adversaryPeeringAll :=
		flag.Bool("adversaryPeeringAll", config.Params.AdversaryPeeringAll, "Flag indicating whether adversary nodes should be able to gossip messages to all nodes in the network directly, or should follow the peering algorithm.")
	burnPolicies :=
		flag.String("burnPolicies", "", "Space seperated list of policies employed by nodes, e.g., '0 1' . Options include: 0 = noburn, 1 = anxious, 2 = greedy, 3 = random_greedy")
	scriptStartTime :=
		flag.String("scriptStartTime", config.Params.ScriptStartTimeStr, "Time the external script started, to be used for results directory.")

	// Parse the flags
	flag.Parse()

	// Update the configuration parameters
	config.Params.NodesCount = *nodesCountPtr
	config.Params.NodesTotalWeight = *nodesTotalWeightPtr
	config.Params.ZipfParameter = *zipfParameterPtr
	config.Params.ConfirmationThreshold = *confirmationThresholdPtr
	config.Params.ConfirmationThresholdAbsolute = *confirmationThresholdAbsolutePtr
	config.Params.ParentsCount = *parentsCountPtr
	config.Params.WeakTipsRatio = *weakTipsRatioPtr
	config.Params.TSA = *tsaPtr
	config.Params.IssuingRate = *issuingRatePtr
	config.Params.SlowdownFactor = *slowdownFactorPtr
	config.Params.ConsensusMonitorTick = *consensusMonitorTickPtr
	config.Params.RelevantValidatorWeight = *relevantValidatorWeightPtr
	config.Params.DoubleSpendDelay = *doubleSpendDelayPtr
	config.Params.PacketLoss = *packetLoss
	config.Params.MinDelay = *minDelay
	config.Params.MaxDelay = *maxDelay
	config.Params.DeltaURTS = *deltaURTS
	config.Params.SimulationStopThreshold = *simulationStopThreshold
	config.Params.ResultDir = *resultDirPtr
	config.Params.IMIF = *imif
	config.Params.RandomnessWS = *randomnessWS
	config.Params.NeighbourCountWS = *neighbourCountWS
	config.Params.SimulationMode = *simulationMode
	config.Params.SchedulingRate = *schedulingRate
	parseMonitoredAWPeers(*monitoredAWPeers)
	parseBurnPolicies(*burnPolicies)
	parseCongestionPeriods(*congestionPeriods)
	config.Params.ScriptStartTimeStr = *scriptStartTime
	parseAccidentalConfig(accidentalMana)
	parseAdversaryConfig(adversaryDelays, adversaryTypes, adversaryMana, adversaryNodeCounts, adversaryInitColors, adversaryPeeringAll, adversarySpeedup)

	config.Params.MonitoredWitnessWeightPeer = *monitoredWitnessWeightPeerPtr
	config.Params.MonitoredWitnessWeightMessageID = *monitoredWitnessWeightMessageIDPtr
	config.Params.SimulationDuration = *simulationDurationPtr
	config.Params.SchedulerType = *schedulerTypePtr
	config.Params.MaxDeficit = *maxDeficitPtr
	config.Params.SlotTime = *slotTimePtr
	config.Params.MinCommittableAge = *minCommittableAgePtr
	config.Params.RMCTime = *rmcTimePtr
	config.Params.InitialMana = *initialMana
	config.Params.InitialRMC = *initialRMCPtr
	config.Params.LowerRMCThreshold = *lowerRMCThresholdPtr
	config.Params.UpperRMCThreshold = *upperRMCThresholdPtr
	config.Params.AlphaRMC = *alphaRMCPtr
	config.Params.BetaRMC = *betaRMCPtr
	config.Params.RMCmin = *rmcMinPtr
	config.Params.RMCmax = *rmcMaxPtr
	config.Params.RMCincrease = *rmcIncreasePtr
	config.Params.RMCdecrease = *rmcDecreasePtr
	config.Params.RMCPeriodUpdate = *rmcPeriodUpdatePtr

	log.Info("Current configuration:")
	log.Info("Simulation Duration: ", config.Params.SimulationDuration)
	log.Info("NodesCount: ", config.Params.NodesCount)
	log.Info("NodesTotalWeight: ", config.Params.NodesTotalWeight)
	log.Info("ZipfParameter: ", config.Params.ZipfParameter)
	log.Info("MonitoredAWPeers:", config.Params.MonitoredAWPeers)
	log.Info("MonitoredWitnessWeightPeer: ", config.Params.MonitoredWitnessWeightPeer)
	log.Info("MonitoredWitnessWeightMessageID: ", config.Params.MonitoredWitnessWeightMessageID)
	log.Info("ConfirmationThreshold: ", config.Params.ConfirmationThreshold)
	log.Info("ConfirmationThresholdAbsolute: ", config.Params.ConfirmationThresholdAbsolute)
	log.Info("ParentsCount: ", config.Params.ParentsCount)
	log.Info("WeakTipsRatio: ", config.Params.WeakTipsRatio)
	log.Info("TSA: ", config.Params.TSA)
	log.Info("SchedulerType: ", config.Params.SchedulerType)
	log.Info("SchedulingRate: ", config.Params.SchedulingRate)
	log.Info("IssuingRate: ", config.Params.IssuingRate)
	log.Info("Congestion periods:", config.Params.CongestionPeriods)
	log.Info("SlowdownFactor: ", config.Params.SlowdownFactor)
	log.Info("ConsensusMonitorTick: ", config.Params.ConsensusMonitorTick)
	log.Info("RelevantValidatorWeight: ", config.Params.RelevantValidatorWeight)
	log.Info("Burn Policies:", config.Params.BurnPolicies)
	log.Info("Initial Mana:", config.Params.InitialMana)
	log.Info("Max Buffer size:", config.Params.MaxBuffer)
	log.Info("Max Deficit:", config.Params.MaxDeficit)
	log.Info("Slot time duration:", config.Params.SlotTime)
	log.Info("MinCommittableAge:", config.Params.MinCommittableAge)
	log.Info("RMCTime: ", config.Params.RMCTime)
	log.Info("InitialRMC: ", config.Params.InitialRMC)
	log.Info("LowerRMCThreshold: ", config.Params.LowerRMCThreshold)
	log.Info("UpperRMCThreshold: ", config.Params.UpperRMCThreshold)
	log.Info("AlphaRMC: ", config.Params.AlphaRMC)
	log.Info("BetaRMC: ", config.Params.BetaRMC)
	log.Info("RMCmin: ", config.Params.RMCmin)
	log.Info("RMCmax: ", config.Params.RMCmax)
	log.Info("RMCincrease: ", config.Params.RMCincrease)
	log.Info("RMCdecrease: ", config.Params.RMCdecrease)
	log.Info("RMCPeriodUpdate: ", config.Params.RMCPeriodUpdate)
	log.Info("DoubleSpendDelay: ", config.Params.DoubleSpendDelay)
	log.Info("PacketLoss: ", config.Params.PacketLoss)
	log.Info("MinDelay: ", config.Params.MinDelay)
	log.Info("MaxDelay: ", config.Params.MaxDelay)
	log.Info("DeltaURTS:", config.Params.DeltaURTS)
	log.Info("SimulationStopThreshold:", config.Params.SimulationStopThreshold)
	log.Info("ResultDir:", config.Params.ResultDir)
	log.Info("IMIF: ", config.Params.IMIF)
	log.Info("WattsStrogatzRandomness: ", config.Params.RandomnessWS)
	log.Info("WattsStrogatzNeighborCount: ", config.Params.NeighbourCountWS)
	log.Info("SimulationMode: ", config.Params.SimulationMode)
	log.Info("AdversaryTypes: ", config.Params.AdversaryTypes)
	log.Info("AdversaryInitColors: ", config.Params.AdversaryInitColors)
	log.Info("AdversaryMana: ", config.Params.AdversaryMana)
	log.Info("AdversaryNodeCounts: ", config.Params.AdversaryNodeCounts)
	log.Info("AdversaryDelays: ", config.Params.AdversaryDelays)
	log.Info("AccidentalMana: ", config.Params.AccidentalMana)
	log.Info("AdversaryPeeringAll: ", config.Params.AdversaryPeeringAll)
	log.Info("AdversarySpeedup: ", config.Params.AdversarySpeedup)
}

func parseMonitoredAWPeers(peers string) {
	if peers == "" {
		return
	}
	peersInt := parseStrToInt(peers)
	config.Params.MonitoredAWPeers = peersInt
}

func parseCongestionPeriods(periods string) {
	if periods == "" {
		return
	}
	periodsFloat := parseStrToFloat64(periods)
	config.Params.CongestionPeriods = periodsFloat
}

func parseBurnPolicies(burnPolicies string) {
	if burnPolicies == "" {
		return
	}
	policiesInt := parseStrToInt(burnPolicies)
	if len(policiesInt) == config.Params.NodesCount {
		config.Params.BurnPolicies = policiesInt
	} else {
		config.Params.BurnPolicies = config.RandomArrayFromValues(0, policiesInt, config.Params.NodesCount)
	}
}

func parseAdversaryConfig(adversaryDelays, adversaryTypes, adversaryMana, adversaryNodeCounts, adversaryInitColors *string, adversaryPeeringAll *bool, adversarySpeedup *string) {
	if config.Params.SimulationMode != "Adversary" {
		config.Params.AdversaryTypes = []int{}
		config.Params.AdversaryNodeCounts = []int{}
		config.Params.AdversaryMana = []float64{}
		config.Params.AdversaryDelays = []int{}
		config.Params.AdversaryInitColors = []string{}
		config.Params.AdversarySpeedup = []float64{}

		return
	}

	config.Params.AdversaryPeeringAll = *adversaryPeeringAll

	if *adversaryDelays != "" {
		config.Params.AdversaryDelays = parseStrToInt(*adversaryDelays)
	}
	if *adversaryTypes != "" {
		config.Params.AdversaryTypes = parseStrToInt(*adversaryTypes)
	}
	if *adversaryMana != "" {
		config.Params.AdversaryMana = parseStrToFloat64(*adversaryMana)
	}
	if *adversaryNodeCounts != "" {
		config.Params.AdversaryNodeCounts = parseStrToInt(*adversaryNodeCounts)
	}
	if *adversaryInitColors != "" {
		config.Params.AdversaryInitColors = parseStr(*adversaryInitColors)
	}
	if *adversarySpeedup != "" {
		config.Params.AdversarySpeedup = parseStrToFloat64(*adversarySpeedup)
	}
	// no adversary if colors are not provided
	if len(config.Params.AdversaryInitColors) != len(config.Params.AdversaryTypes) {
		config.Params.AdversaryTypes = []int{}
	}

	// make sure mana, nodeCounts and delays are only defined when adversary type is provided and have the same length
	if len(config.Params.AdversaryDelays) != 0 && len(config.Params.AdversaryDelays) != len(config.Params.AdversaryTypes) {
		log.Warnf("The AdversaryDelays count is not equal to the AdversaryTypes count!")
		config.Params.AdversaryDelays = []int{}
	}
	if len(config.Params.AdversaryMana) != 0 && len(config.Params.AdversaryMana) != len(config.Params.AdversaryTypes) {
		log.Warnf("The AdversaryMana count is not equal to the AdversaryTypes count!")
		config.Params.AdversaryMana = []float64{}
	}
	if len(config.Params.AdversaryNodeCounts) != 0 && len(config.Params.AdversaryNodeCounts) != len(config.Params.AdversaryTypes) {
		log.Warnf("The AdversaryNodeCounts count is not equal to the AdversaryTypes count!")
		config.Params.AdversaryNodeCounts = []int{}
	}
}

func parseAccidentalConfig(accidentalMana *string) {
	if config.Params.SimulationMode != "Accidental" {
		config.Params.AccidentalMana = []string{}
		return
	}
	if *accidentalMana != "" {
		config.Params.AccidentalMana = parseStr(*accidentalMana)
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
	bytes, err := json.MarshalIndent(config.Params, "", " ")
	if err != nil {
		log.Error(err)
	}
	if _, err := os.Stat(config.Params.ResultDir); os.IsNotExist(err) {
		err = os.Mkdir(config.Params.ResultDir, 0700)
		if err != nil {
			log.Error(err)
		}
	}
	if err := os.WriteFile(path.Join(config.Params.ResultDir, fileName), bytes, 0644); err != nil {
		log.Error(err)
	}

}
