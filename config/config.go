package config

var (
	NodesCount              = 100
	NodesTotalWeight        = 100_000_000
	ZipfParameter           = 2.0
	WeightThreshold         = 0.66
	WeightThresholdAbsolute = true
	WeightThresholdRandom   = true // If set to true, set the threshold to be in the range of [0.5,  WeightThreshold) and enable FPCS.
	TipsCount               = 8    // The TipsCount for a message
	WeakTipsRatio           = 0.0  // The ratio of weak tips
	TSA                     = "URTS"
	TPS                     = 100
	DecelerationFactor      = 1                  // The factor to control the speed in the simulation.
	ConsensusMonitorTick    = 100                // Tick to monitor the consensus, in milliseconds.
	RelevantValidatorWeight = 0                  // The node whose weight * RelevantValidatorWeight <= largestWeight will not issue messages (disabled now)
	IMIF                    = "poisson"          // Inter Message Issuing Function for time delay between activity messages: poisson or uniform
	PayloadLoss             = 0.0                // The payload loss in the network.
	MinDelay                = 100                // The minimum network delay in ms.
	MaxDelay                = 100                // The maximum network delay in ms.
	DoubleSpendDelay        = 20                 // Delay after which double spending transactions will be issued. In seconds.
	DeltaURTS               = 5.0                // in seconds, reference: https://iota.cafe/t/orphanage-with-restricted-urts/1199
	SimulationStopThreshold = 1.0                // Stop the simulation when > SimulationStopThreshold * NodesCount have reached the same opinion.
	SimulationTarget        = "DS"               // The simulation target, CT: Confirmation Time, DS: Double Spending
	ResultDir               = "results"          // Path where all the result files will be saved
	RandomnessWS            = 1.0                // WattsStrogatz randomness parameter, gamma parameter described in https://blog.iota.org/the-fast-probabilistic-consensus-simulator-d5963c558b6e/
	NeighbourCountWS        = 8                  // Number of neighbors node is connected to in WattsStrogatz network topology
	SimulationMode          = "Adversary"        // Mode for the DS simulations one of: 'Accidental' - accidental double spends sent by max, min or random weight node from Zipf distrib, 'Adversary' - need to use adversary groups (parameters starting with 'Adversary...')
	AccidentalMana          = []string{"random"} // Defines nodes which will be used: 'min', 'max', 'random' or valid nodeID
	AdversaryDelays         = []int{}            // Delays in ms of adversary nodes, eg '50 100 200', SimulationTarget must be 'DS'
	AdversaryTypes          = []int{4}           // Defines group attack strategy, one of the following: 0 - honest node behavior, 1 - shifts opinion, 2 - keeps the same opinion, 3 - nodes not gossiping anything, even DS. SimulationTarget must be 'DS', 4 - creating new opinion
	AdversaryMana           = []float64{5}       // Adversary nodes mana in %, e.g. '10 10'. Default value: 1%. SimulationTarget must be 'DS'
	AdversaryNodeCounts     = []int{1}           // Defines number of adversary nodes in the group. Leave empty for default value: 1.
	AdversaryInitColors     = []string{"B"}      // Defines initial color for adversary group, one of following: 'R', 'G', 'B'. Mandatory for each group.
	AdversaryPeeringAll     = false              // Defines a flag indicating whether adversarial nodes should be able to send messages to all nodes in the network, instead of following regular peering algorithm.
	AdversarySpeedup        = []float64{1.0}     // Defines how many more messages should adversary nodes issue.
	FPCSEpochPeriod         = 30                 // The period of generation a new random number in seconds.
	FPCSLowerBound          = 500                // The lower bound of the generated random number.
	FPCSUpperBound          = 660                // The upper bound of the generated random number.
	FPCSTriggerTime         = 5                  // The time we trigger FPCS after the adversary starts to attack.
)

var (
	MonitoredAWPeers = [...]int{0}
	MonitoredTPPeer  = 0
)
