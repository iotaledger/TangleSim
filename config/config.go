package config

var (
	NodesCount              = 100
	NodesTotalWeight        = 100_000_000
	ZipfParameter           = 2.0
	MessageWeightThreshold  = 0.5
	TipsCount               = 4   // The TipsCount for a message
	WeakTipsRatio           = 0.0 // The ratio of weak tips
	TSA                     = "URTS"
	TPS                     = 100
	DecelerationFactor      = 1.0  // The factor to control the speed in the simulation.
	ConsensusMonitorTick    = 1000 // Tick to monitor the consensus, in milliseconds.
	ReleventValidatorWeight = 1000 // The node whose weight * ReleventValidatorWeight <= largestWeight will not issue messages
)

var (
	MonitoredAWPeers = [...]int{0}
)
