package config

var (
	NodesCount              = 100
	NodesTotalWeight        = 100_000_000
	ZipfParameter           = 0.9
	MessageWeightThreshold  = 0.5
	TipsCount               = 8   // The TipsCount for a message
	WeakTipsRatio           = 0.0 // The ratio of weak tips
	TSA                     = "URTS"
	TPS                     = 100
	DecelerationFactor      = 10           // The factor to control the speed in the simulation.
	ConsensusMonitorTick    = 100          // Tick to monitor the consensus, in milliseconds.
	RelevantValidatorWeight = 0            // The node whose weight * RelevantValidatorWeight <= largestWeight will not issue messages (disabled now)
	IMIF                    = "poisson"    // Inter Message Issuing Function for time delay between activity messages: poisson or uniform
	PayloadLoss             = 0.0          // The payload loss in the network.
	MinDelay                = 100          // The minimum network delay in ms.
	MaxDelay                = 100          // The maximum network delay in ms.
	DoubleSpendDelay        = 1            // Delay after which double spending transactions will be issued. In seconds.
	DeltaURTS               = 5.0          // in seconds, reference: https://iota.cafe/t/orphanage-with-restricted-urts/1199
	SimulationStopThreshold = 1.0          // Stop the simulation when > SimulationStopThreshold * NodesCount have reached the same opinion.
	SimulationTarget        = "DS"         // The simulation target, CT: Confirmation Time, DS: Double Spending
	ResultDir               = "results"    // Path where all the result files will be saved
	RandomnessWS            = 1.0          // WattsStrogatz randomness parameter, gamma parameter described in https://blog.iota.org/the-fast-probabilistic-consensus-simulator-d5963c558b6e/
	NeighbourCountWS        = 8            // Number of neighbors node is connected to in WattsStrogatz network topology
	AdversaryDelays         = []int{}      // Delays in ms of adversary nodes, eg '50 100 200', SimulationTarget must be 'DS'
	AdversaryTypes          = []int{0, 0}  // Defines attack strategy, one of the following: 0 - honest node behavior, 1 - shifts opinion, 2 - keeps the same opinion. Max 3 types, one per color. SimulationTarget must be 'DS'
	AdversaryMana           = []float64{}  // Adversary nodes mana in %, e.g. '10 10' or leave empty if adversary nodes should be selected randomly, 0 means low mana node, 100 means high mana node. SimulationTarget must be 'DS'
	AdversaryErrorThreshold = 0.00_00_00_1 // The error threshold of q - percentage of mana held by adversary. SimulationTarget must be 'DS'
	AdversaryIndexStart     = 1            // Skipping wealthiest nodes in zipfs distribution, during adversary groups creation.  A way to increase number of adversary nodes in groups when percentage of mana is given in AdversaryMana
)

var (
	MonitoredAWPeers = [...]int{0}
	MonitoredTPPeer  = 0
)
