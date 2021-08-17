package config

const (
	NodesCount             = 10
	NodesTotalWeight       = 100000000
	ZipfParameter          = 0.9
	MessageWeightThreshold = 0.5
	TSA                    = "URTS"
	TPS                    = 1000
	DecelerationFactor     = 5000.0 // The factor to control the speed in the simulation.
)

var (
	MonitoredAWPeers = [...]int{0}
)
