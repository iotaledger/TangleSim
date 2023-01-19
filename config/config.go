package config

// simulator settings

var (
	ResultDir                       = "results"   // Path where all the result files will be saved
	SimulationTarget                = "CT"        // The simulation target, CT: Confirmation Time, DS: Double Spending
	SimulationStopThreshold         = 1.0         // Stop the simulation when > SimulationStopThreshold * NodesCount have reached the same opinion.
	ConsensusMonitorTick            = 100         // Tick to monitor the consensus, in milliseconds.
	MonitoredAWPeers                = [...]int{0} // Nodes for which we monitor the AW growth
	MonitoredWitnessWeightPeer      = 0           // Peer for which we monitor Witness Weight
	MonitoredWitnessWeightMessageID = 200         // A specified message ID to monitor the witness weights
)

// Network setup

var (
	NodesCount       = 100       // NodesCount is the total number of nodes simulated in the network.
	TPS              = 100       // TPS defines the total network throughput.
	ParentsCount     = 8         // ParentsCount that a new message is selecting from the tip pool.
	NeighbourCountWS = 8         // Number of neighbors node is connected to in WattsStrogatz network topology.
	RandomnessWS     = 1.0       // WattsStrogatz randomness parameter, gamma parameter described in https://blog.iota.org/the-fast-probabilistic-consensus-simulator-d5963c558b6e/
	IMIF             = "poisson" // IMIF Inter Message Issuing Function for time delay between activity messages: poisson or uniform.
	PacketLoss       = 0.0       // The packet loss in the network.
	MinDelay         = 100       // The minimum network delay in ms.
	MaxDelay         = 100       // The maximum network delay in ms.

	SlowdownFactor = 1 // The factor to control the speed in the simulation.
)

// Weight setup

var (
	NodesTotalWeight              = 100_000_000 // Total number of weight for the whole network.
	ZipfParameter                 = 0.9         // the 's' parameter for the Zipf distribution used to model weight distribution. s=0 all nodes are equal, s=2 network centralized.
	ConfirmationThreshold         = 0.66        // Threshold for AW collection above which messages are considered confirmed.
	ConfirmationThresholdAbsolute = true        // If true the threshold is alway counted from zero if false the weight collected is counted from the next peer weight.
	RelevantValidatorWeight       = 0           // The node whose weight * RelevantValidatorWeight <= largestWeight will not issue messages (disabled now)
)

// Tip Selection Algorithm setup

var (
	TSA           = "URTS" // Currently only one supported TSA is URTS
	DeltaURTS     = 5.0    // in seconds, reference: https://iota.cafe/t/orphanage-with-restricted-urts/1199
	WeakTipsRatio = 0.0    // The ratio of weak tips
)

// TODO: expose the configuration in Parser
// Mana Burn Setup
// 0 = noburn, 1 = anxious, 2 = greedy, 3 = random_greedy
var (
	// BurnPolicy = ZeroValueArray(NodesCount)
	BurnPolicy         = RandomValueArray(99, 2, 3, NodesCount)
	NodeInitAccessMana = ZeroValueArray(NodesCount)
	ExtraBurn          = 1.0
)

// Adversary setup - enabled by setting SimulationTarget="DS"
var (
	// SimulationMode for the DS simulations one of:
	// 'Accidental' - accidental double spends sent by max, min or random weight node from Zipf distrib,
	// 'Adversary' - need to use adversary groups (parameters starting with 'Adversary...')
	SimulationMode   = "Accidental"
	DoubleSpendDelay = 20 // Delay after which double spending transactions will be issued. In seconds.

	AccidentalMana = []string{"random", "random"} // Defines nodes which will be used: 'min', 'max', 'random' or valid nodeID

	AdversaryDelays     = []int{}             // Delays in ms of adversary nodes, eg '50 100 200', SimulationTarget must be 'DS'
	AdversaryTypes      = []int{0, 0}         // Defines group attack strategy, one of the following: 0 - honest node behavior, 1 - shifts opinion, 2 - keeps the same opinion, 3 - nodes not gossiping anything, even DS. SimulationTarget must be 'DS'
	AdversaryMana       = []float64{}         // Adversary nodes mana in %, e.g. '10 10'. Default value: 1%. SimulationTarget must be 'DS'
	AdversaryNodeCounts = []int{}             // Defines number of adversary nodes in the group. Leave empty for default value: 1.
	AdversaryInitColors = []string{"R", "B"}  // Defines initial color for adversary group, one of following: 'R', 'G', 'B'. Mandatory for each group.
	AdversaryPeeringAll = false               // Defines a flag indicating whether adversarial nodes should be able to send messages to all nodes in the network, instead of following regular peering algorithm.
	AdversarySpeedup    = []float64{1.0, 1.0} // Defines how many more messages should adversary nodes issue.
)
