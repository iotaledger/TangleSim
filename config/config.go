package config

const (
	NodesCount              = 10_000
	NodesTotalWeight        = 100_000_000
	ZipfParameter           = 0.9
	MessageWeightThreshold  = 0.5
	TSA                     = "URTS"
	TPS                     = 1000
	DecelerationFactor      = 5.0 // The factor to control the speed in the simulation.
	SimulationStopThreshold = 0.5 // Stop the simulation when > SimulationStopThreshold * NodesCount have reached the same opinion.
)

var (
	MonitoredAWPeers = [...]int{0}
)
