package config

var (
	NodesCount              = 200
	NodesTotalWeight        = 100_000_000
	ZipfParameter           = 0.9
	MessageWeightThreshold  = 0.5
	TipsCount               = 4   // The TipsCount for a message
	WeakTipsRatio           = 0.0 // The ratio of weak tips
	TSA                     = "URTS"
	TPS                     = 1000
	DecelerationFactor      = 1.0  // The factor to control the speed in the simulation.
	ConsensusMonitorTick    = 1000 // Tick to monitor the consensus, in milliseconds.
	ReleventValidatorWeight = 1000 // The node whose weight * ReleventValidatorWeight <= largestWeight will not issue messages
	PayloadLoss             = 0.0  // The payload loss in the network
	MinDelay                = 0    // The minimum network delay in ms
	MaxDelay                = 0    // The maximum network delay in ms
)

var (
	MonitoredAWPeers = [...]int{0}
)
