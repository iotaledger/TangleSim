package config

var (
	NodesCount              = 100
	NodesTotalWeight        = 100_000_000
	ZipfParameter           = 0.9
	MessageWeightThreshold  = 0.5
	TipsCount               = 4   // The TipsCount for a message
	WeakTipsRatio           = 0.0 // The ratio of weak tips
	TSA                     = "URTS"
	TPS                     = 100
	DecelerationFactor      = 1.0 // The factor to control the speed in the simulation.
	ConsensusMonitorTick    = 100 // Tick to monitor the consensus, in milliseconds.
	ReleventValidatorWeight = 0   // The node whose weight * ReleventValidatorWeight <= largestWeight will not issue messages (disabled now)
	PayloadLoss             = 0.0 // The payload loss in the network.
	MinDelay                = 100 // The minimum network delay in ms.
	MaxDelay                = 100 // The maximum network delay in ms.
	DeltaURTS               = 5.0 // in seconds, reference: https://iota.cafe/t/orphanage-with-restricted-urts/1199
)

var (
	MonitoredAWPeers = [...]int{0}
)
