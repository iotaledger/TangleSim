package config

import (
	"time"
)

type Config struct {
	*SimulatorSettings
	*NetworkSettings
	*WeightSettings
	*CongestionControlSettings
	*TipSelectionAlgorithmSettings
	*AdversarySettings
}

// ParametersProtocol contains the definition of the configuration parameters used by the Protocol.
type SimulatorSettings struct {
	// Path where all the result files will be saved
	ResultDir string `default:"results"`
	// The simulation target, CT: Confirmation Time, DS: Double Spending
	SimulationTarget string `default:"CT"`
	// Stop the simulation when > SimulationStopThreshold * NodesCount have reached the same opinion.
	SimulationStopThreshold float64 `default:"1.0"`
	// Tick to monitor the consensus, in milliseconds.
	ConsensusMonitorTick int `default:"100"`
	// Nodes for which we monitor the AW growth
	MonitoredAWPeers []int
	// Peer for which we monitor Witness Weight
	MonitoredWitnessWeightPeer int `default:"0"`
	// A specified message ID to monitor the witness weights
	MonitoredWitnessWeightMessageID int `default:"200"`
	// A string indicating the start time of a simulation started by an external script
	ScriptStartTimeStr string        `default:"20060102_1504"`
	GeneralOutputDir   string        `default:"results/20060102_1504/general"`
	SchedulerOutputDir string        `default:"results/20060102_1504/scheduler"`
	SimulationDuration time.Duration `default:"1m"`
}

type NetworkSettings struct {
	// NodesCount is the total number of nodes simulated in the network.
	NodesCount int `default:"100"`
	// ValidatorCount is the total number of nodes simulated in the network.
	ValidatorCount int `default:"20"`
	// CommitteeBandwidth is the total bandwidth of the committee in the network.
	CommitteeBandwidth float64 `default:"0.5"`
	// ValidatorBPS is the rate of validation blocks simulated in the network per validator node.
	ValidatorBPS int `default:"1"`
	// Scheduler rate in units of messages per second.
	SchedulingRate int `default:"200"`
	// Total rate of issuing messages in units of messages per second.
	IssuingRate int `default:"100"`
	//, 0.5, 1.5, 1.5, 0.5} // congested/uncongested periods
	CongestionPeriods []float64
	// ParentsCount that a new message is selecting from the tip pool.
	ParentsCount int `default:"8"`
	// ParentCountVB is the number of validation block parents for validation block tsa.
	ParentCountVB int `default:"2"`
	// ParentCountNVB is the number of non-validation block parents for validation block tsa.
	ParentCountNVB int `default:"38"`
	// Number of neighbors node is connected to in WattsStrogatz network topology.
	NeighbourCountWS int `default:"4"`
	// WattsStrogatz randomness parameter, gamma parameter described in https://blog.iota.org/the-fast-probabilistic-consensus-simulator-d5963c558b6e/
	RandomnessWS float64 `default:"1.0"`
	// IMIF Inter Message Issuing Function for time delay between activity messages: poisson or uniform.
	IMIF string `default:"poisson"`
	// The packet loss in the network.
	PacketLoss float64 `default:"0.0"`
	// The minimum network delay in ms.
	MinDelay int `default:"100"`
	// The maximum network delay in ms.
	MaxDelay int `default:"100"`
	// The factor to control the speed in the simulation.
	SlowdownFactor int `default:"1"`
}

// Weight setup

type WeightSettings struct {
	// Total number of weight for the whole network.
	NodesTotalWeight int `default:"100_000_000"`
	// the 's' parameter for the Zipf distribution used to model weight distribution. s=0 all nodes are equal, s=2 network centralized.
	ZipfParameter float64 `default:"0.9"`
	// Threshold for AW collection above which messages are considered confirmed.
	ConfirmationThreshold float64 `default:"0.66"`
	// If true the threshold is always counted from zero if false the weight collected is counted from the next peer weight.
	ConfirmationThresholdAbsolute bool `default:"true"`
	// The node whose weight * RelevantValidatorWeight <= largestWeight will not issue messages (disabled now)
	RelevantValidatorWeight int `default:"0"`
}

// Tip Selection Algorithm setup

type TipSelectionAlgorithmSettings struct {
	// Currently only one supported TSA is RURTS
	TSA string `default:"RURTS"`
	// in seconds, reference: https://iota.cafe/t/orphanage-with-restricted-urts/1199
	DeltaURTS float64 `default:"5.0"`
	// The ratio of weak tips
	WeakTipsRatio float64 `default:"0.0"`
}

// Congestion Control

type CongestionControlSettings struct {
	SchedulerType     string `default:"ICCA+"` // ManaBurn or ICCA+
	BurnPolicies      []int
	InitialMana       float64       `default:"0.0"`
	MaxBuffer         int           `default:"25"`
	ConfEligible      bool          `default:"true"` // if true, then confirmed is used for eligible check. else just scheduled
	MaxDeficit        float64       `default:"2.0"`  // maximum deficit for any id
	SlotTime          time.Duration `default:"1s"`
	MinCommittableAge time.Duration `default:"4s"`
	RMCTime           time.Duration `default:"4s"`
	// inital value of RMC
	InitialRMC float64 `default:"1.0"`
	// T1 for RMC
	LowerRMCThreshold float64 `default:"50"`
	// T2 for RMC
	UpperRMCThreshold float64 `default:"75"`
	AlphaRMC          float64 `default:"0.8"`
	BetaRMC           float64 `default:"1.2"`
	RMCmin            float64 `default:"0.25"`
	RMCmax            float64 `default:"2.0"`
	RMCincrease       float64 `default:"1.0"`
	RMCdecrease       float64 `default:"0.5"`
	RMCPeriodUpdate   int     `default:"5"`
}

// Adversary setup - enabled by setting SimulationTarget="DS"
type AdversarySettings struct {
	// SimulationMode defines the type of adversary simulation, one of the following:
	// 'None' - no adversary simulation,
	// 'Accidental' - accidental double spends sent by max, min or random weight node from Zipf distribution,
	// 'Adversary' - need to use adversary groups (parameters starting with 'Adversary...')
	// 'Blowball' - enables adversary node that is able to perform a blowball attack.
	SimulationMode string `default:"None"`
	// Delay after which double spending transactions will be issued. In seconds.
	DoubleSpendDelay int `default:"5"`
	// Defines nodes which will be used: 'min', 'max', 'random' or valid nodeID
	AccidentalMana []string
	// Delays in ms of adversary nodes, eg '50 100 200', SimulationTarget must be 'DS'
	AdversaryDelays []int
	// Defines group attack strategy, one of the following: 0 - honest node behavior, 1 - shifts opinion, 2 - keeps the same opinion, 3 - nodes not gossiping anything, even DS. SimulationTarget must be 'DS'
	AdversaryTypes []int
	// Adversary nodes mana in %, e.g. '10 10'. Default value: 1%. SimulationTarget must be 'DS'
	AdversaryMana []float64
	// Defines number of adversary nodes in the group. Leave empty for default value: 1.
	AdversaryNodeCounts []int
	// Defines initial color for adversary group, one of following: 'R', 'G', 'B'. Mandatory for each group.
	AdversaryInitColors []string
	// Defines a flag indicating whether adversarial nodes should be able to send messages to all nodes in the network, instead of following regular peering algorithm.
	AdversaryPeeringAll bool `default:"false"`
	// Defines how many more messages should adversary nodes issue.
	AdversarySpeedup []float64

	// The mana of the blowball node in % of total mana
	BlowballMana int `default:"20"`
	// The size of the blowball
	BlowballSize int `default:"20"`
	// The delay in seconds between the consecutive blowballs
	BlowballDelay int `default:"5"`
	// The maximum number of blowballs sent to the network
	BlowballMaxSent int `default:"2"`
	// The node ID of the blowball node
	BlowballNodeID int `default:"0"`
}
