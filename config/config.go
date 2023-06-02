package config

import (
	"path"
	"time"
)

// simulator settings

var (
	ResultDir                       = "results"                            // Path where all the result files will be saved
	SimulationTarget                = "CT"                                 // The simulation target, CT: Confirmation Time, DS: Double Spending
	SimulationStopThreshold         = 1.0                                  // Stop the simulation when > SimulationStopThreshold * NodesCount have reached the same opinion.
	ConsensusMonitorTick            = 100                                  // Tick to monitor the consensus, in milliseconds.
	MonitoredAWPeers                = [...]int{0}                          // Nodes for which we monitor the AW growth
	MonitoredWitnessWeightPeer      = 0                                    // Peer for which we monitor Witness Weight
	MonitoredWitnessWeightMessageID = 200                                  // A specified message ID to monitor the witness weights
	ScriptStartTimeStr              = time.Now().Format("20060102_150405") // A string indicating the start time of a simulation started by an external script
	GeneralOutputDir                = path.Join(ResultDir, ScriptStartTimeStr, "general")
	SchedulerOutputDir              = path.Join(ResultDir, ScriptStartTimeStr, "scheduler")
	SimulationDuration              = time.Duration(1) * time.Minute
)

// Network setup

var (
	NodesCount        = 100                           // NodesCount is the total number of nodes simulated in the network.
	SchedulingRate    = 100                           // Scheduler rate in units of messages per second.
	IssuingRate       = SchedulingRate                // Total rate of issuing messages in units of messages per second.
	CongestionPeriods = []float64{1.5, 1.5, 1.5, 1.5} //, 0.5, 1.5, 1.5, 0.5} // congested/uncongested periods
	ParentsCount      = 8                             // ParentsCount that a new message is selecting from the tip pool.
	NeighbourCountWS  = 4                             // Number of neighbors node is connected to in WattsStrogatz network topology.
	RandomnessWS      = 1.0                           // WattsStrogatz randomness parameter, gamma parameter described in https://blog.iota.org/the-fast-probabilistic-consensus-simulator-d5963c558b6e/
	IMIF              = "poisson"                     // IMIF Inter Message Issuing Function for time delay between activity messages: poisson or uniform.
	PacketLoss        = 0.0                           // The packet loss in the network.
	MinDelay          = 100                           // The minimum network delay in ms.
	MaxDelay          = 100                           // The maximum network delay in ms.

	SlowdownFactor = 1 // The factor to control the speed in the simulation.
)

// Weight setup

var (
	NodesTotalWeight              = 100_000_000 // Total number of weight for the whole network.
	ZipfParameter                 = 0.9         // the 's' parameter for the Zipf distribution used to model weight distribution. s=0 all nodes are equal, s=2 network centralized.
	ConfirmationThreshold         = 0.66        // Threshold for AW collection above which messages are considered confirmed.
	ConfirmationThresholdAbsolute = true        // If true the threshold is always counted from zero if false the weight collected is counted from the next peer weight.
	RelevantValidatorWeight       = 0           // The node whose weight * RelevantValidatorWeight <= largestWeight will not issue messages (disabled now)
)

// Tip Selection Algorithm setup

var (
	TSA           = "RURTS" // Currently only one supported TSA is URTS
	DeltaURTS     = 5.0     // in seconds, reference: https://iota.cafe/t/orphanage-with-restricted-urts/1199
	WeakTipsRatio = 0.0     // The ratio of weak tips
)

// Congestion Control

var (
	SchedulerType     = "ICCA+" // ManaBurn or ICCA+
	BurnPolicies      = RandomArrayFromValues(0, []int{0, 1}, NodesCount)
	InitialMana       = 0.0
	MaxBuffer         = 200
	ConfEligible      = true // if true, then confirmed is used for eligible check. else just scheduled
	MaxDeficit        = 2.0  // maximum deficit for any id
	SlotTime          = time.Duration(1 * float64(time.Second))
	MinCommittableAge = time.Duration(5 * float64(time.Second))
	RMCTime           = 2 * MinCommittableAge
	InitialRMC        = 1.0                                                 // inital value of RMC
	LowerRMCThreshold = 0.5 * float64(SchedulingRate) * SlotTime.Seconds()  // T1 for RMC
	UpperRMCThreshold = 0.75 * float64(SchedulingRate) * SlotTime.Seconds() // T2 for RMC
	AlphaRMC          = 0.8
	BetaRMC           = 1.2
	RMCmin            = 0.25
	RMCmax            = 2.0
	RMCincrease       = 1.0
	RMCdecrease       = 0.5
	RMCPeriodUpdate   = 5
)

// Adversary setup - enabled by setting SimulationTarget="DS"
var (
	// SimulationMode defines the type of adversary simulation, one of the following:
	// 'None' - no adversary simulation,
	// 'Accidental' - accidental double spends sent by max, min or random weight node from Zipf distribution,
	// 'Adversary' - need to use adversary groups (parameters starting with 'Adversary...')
	// 'Blowball' - enables adversary node that is able to perform a blowball attack.
	SimulationMode   = "None" // "None", "Accidental", "Adversary", "Blowball"
	DoubleSpendDelay = 5      // Delay after which double spending transactions will be issued. In seconds.

	AccidentalMana = []string{"random", "random"} // Defines nodes which will be used: 'min', 'max', 'random' or valid nodeID

	AdversaryDelays     = []int{}             // Delays in ms of adversary nodes, eg '50 100 200', SimulationTarget must be 'DS'
	AdversaryTypes      = []int{0, 0}         // Defines group attack strategy, one of the following: 0 - honest node behavior, 1 - shifts opinion, 2 - keeps the same opinion, 3 - nodes not gossiping anything, even DS. SimulationTarget must be 'DS'
	AdversaryMana       = []float64{}         // Adversary nodes mana in %, e.g. '10 10'. Default value: 1%. SimulationTarget must be 'DS'
	AdversaryNodeCounts = []int{}             // Defines number of adversary nodes in the group. Leave empty for default value: 1.
	AdversaryInitColors = []string{"R", "B"}  // Defines initial color for adversary group, one of following: 'R', 'G', 'B'. Mandatory for each group.
	AdversaryPeeringAll = false               // Defines a flag indicating whether adversarial nodes should be able to send messages to all nodes in the network, instead of following regular peering algorithm.
	AdversarySpeedup    = []float64{1.0, 1.0} // Defines how many more messages should adversary nodes issue.

	BlowballMana    = 20 // The mana of the blowball node in % of total mana
	BlowballSize    = 20 // The size of the blowball
	BlowballDelay   = 5  // The delay in seconds between the consecutive blowballs
	BlowballMaxSent = 2  // The maximum number of blowballs sent to the network
	BlowballNodeID  = 0  // The node ID of the blowball node
)
