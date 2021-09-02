package config

var (
	NodesCount              = 10_000
	NodesTotalWeight        = 100_000_000
	ZipfParameter           = 0.9
	MessageWeightThreshold  = 0.5
	TipsCount               = 4    // The TipsCount for a message
	WeakTipsRatio           = 0.25 // The ratio of weak tips
	TSA                     = "RURTS"
	TPS                     = 1000
	DecelerationFactor      = 5.0  // The factor to control the speed in the simulation.
	ConsensusMonitorTick    = 1000 // Tick to monitor the consensus, in milliseconds.
	ReleventValidatorWeight = 1000 // The node whose weight * ReleventValidatorWeight <= largestWeight will not issue messages
	DeltaURTS               = 5.0  // in seconds, reference: https://iota.cafe/t/orphanage-with-restricted-urts/1199
	SimulationStopThreshold = 0.5 // Stop the simulation when > SimulationStopThreshold * NodesCount have reached the same opinion.
)

var (
	MonitoredAWPeers = [...]int{0}
)
