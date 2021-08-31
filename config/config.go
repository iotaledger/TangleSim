package config

var (
	NodesCount              = 1_000
	NodesTotalWeight        = 100_000_000
	ZipfParameter           = 0.9
	MessageWeightThreshold  = 0.5
	TipsCount               = 4    // The TipsCount for a message
	WeakTipsRatio           = 0.25 // The ratio of weak tips
	TSA                     = "URTS"
	TPS                     = 1000
	DecelerationFactor      = 2.0  // The factor to control the speed in the simulation.
	ConsensusMonitorTick    = 100  // Tick to monitor the consensus, in milliseconds.
	ReleventValidatorWeight = 1000 // The node whose weight * ReleventValidatorWeight <= largestWeight will not issue messages
)

var (
	MonitoredAWPeers = [...]int{0}
)
