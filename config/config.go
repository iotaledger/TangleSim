package config

const (
	NodesCount             = 10000
	NodesTotalWeight       = 100000000
	ZipfParameter          = 0.9
	MessageWeightThreshold = 0.5
	TSA                    = "RURTS"
	TPS                    = 1000
	DecelerationFactor     = 5.0  // The factor to control the speed in the simulation.
	TipsCount              = 4    // The TipsCount for a message
	WeakTipsRatio          = 0.25 // The ratio of weak tips
	DeltaURTS              = 5.0  // in seconds, reference: https://iota.cafe/t/orphanage-with-restricted-urts/1199
)
