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
	DecelerationFactor      = 1         // The factor to control the speed in the simulation.
	ConsensusMonitorTick    = 100       // Tick to monitor the consensus, in milliseconds.
	RelevantValidatorWeight = 0         // The node whose weight * ReleventValidatorWeight <= largestWeight will not issue messages (disabled now)
	IMIF                    = "poisson" // Inter Message Issuing Function for time delay between activity messages: poisson or uniform
	PayloadLoss             = 0.0       // The payload loss in the network.
	MinDelay                = 100       // The minimum network delay in ms.
	MaxDelay                = 100       // The maximum network delay in ms.
	DoubleSpendDelay        = 20        // Delay after which double spending transactions will be issued. In seconds.
	DeltaURTS               = 5.0       // in seconds, reference: https://iota.cafe/t/orphanage-with-restricted-urts/1199
	SimulationStopThreshold = 1.0       // Stop the simulation when > SimulationStopThreshold * NodesCount have reached the same opinion.
	SimulationTarget        = "CT"      // The simulation target, CT: Confirmation Time, DS: Double Spending
	ResultDir               = "results" // Path where all the result files will be saved
	RandomnessWS            = 1.0       // WattsStrogatz randomness parameter, gamma parameter described in https://blog.iota.org/the-fast-probabilistic-consensus-simulator-d5963c558b6e/
	NeighbourCountWS        = 8         // Number of neighbors node is connected to in WattsStrogatz network topology
)

var (
	MonitoredAWPeers = [...]int{0}
	MonitoredTPPeer  = 0
)
