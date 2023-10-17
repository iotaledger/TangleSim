package config

import (
	"path"
	"time"
)

// parameters that will be used in multiple settings.
var (
	ResultDir          = "results"
	ScriptStartTimeStr = time.Now().Format("20060102_1504")
	GeneralOutputDir   = path.Join(ResultDir, ScriptStartTimeStr, "general")
	SchedulerOutputDir = path.Join(ResultDir, ScriptStartTimeStr, "scheduler")

	NodesCount = 100

	SchedulingRate = 200

	SlotTime          = time.Duration(1 * float64(time.Second))
	MinCommittableAge = time.Duration(30 * float64(time.Second))
	RMCmin            = 500000.0 // 0.25
)

// simulator settings
var Params = &Config{
	SimulatorSettings: &SimulatorSettings{
		ResultDir:                       ResultDir,
		SimulationTarget:                "CT",
		SimulationStopThreshold:         1.0,
		ConsensusMonitorTick:            100,
		MonitoredAWPeers:                []int{0},
		MonitoredWitnessWeightPeer:      0,
		MonitoredWitnessWeightMessageID: 200,
		ScriptStartTimeStr:              ScriptStartTimeStr,
		GeneralOutputDir:                GeneralOutputDir,
		SchedulerOutputDir:              SchedulerOutputDir,
		SimulationDuration:              time.Duration(1) * time.Minute,
	},
	NetworkSettings: &NetworkSettings{
		CommitteeBandwidth: 0.5,
		NodesCount:         NodesCount,
		SchedulingRate:     SchedulingRate,
		IssuingRate:        SchedulingRate,
		CongestionPeriods:  []float64{1.0, 1.0, 1.0, 1.0},
		ValidatorCount:     20,
		ValidatorBPS:       2,
		ParentsCount:       8,
		ParentCountVB:      2,
		ParentCountNVB:     38,
		NeighbourCountWS:   4,
		RandomnessWS:       1.0,
		IMIF:               "poisson",
		PacketLoss:         0.0,
		MinDelay:           100,
		MaxDelay:           100,

		SlowdownFactor: 1,
	},
	WeightSettings: &WeightSettings{
		NodesTotalWeight:              100_000_000,
		ZipfParameter:                 0.9,
		ConfirmationThreshold:         0.66,
		ConfirmationThresholdAbsolute: true,
		RelevantValidatorWeight:       0,
	},
	TipSelectionAlgorithmSettings: &TipSelectionAlgorithmSettings{
		TSA:           "RURTS",
		DeltaURTS:     30.0,
		WeakTipsRatio: 0.0,
	},
	CongestionControlSettings: &CongestionControlSettings{
		SchedulerType:     "ICCA+",
		BurnPolicies:      RandomArrayFromValues(0, []int{0, 1}, NodesCount),
		InitialMana:       0.0,
		MaxBuffer:         25,
		ConfEligible:      true,
		MaxDeficit:        2.0,
		SlotTime:          time.Duration(1 * float64(time.Second)),
		MinCommittableAge: MinCommittableAge,
		RMCTime:           MinCommittableAge,
		LowerRMCThreshold: 0.5 * float64(SchedulingRate) * SlotTime.Seconds(),
		UpperRMCThreshold: 0.75 * float64(SchedulingRate) * SlotTime.Seconds(),
		AlphaRMC:          0.8,
		BetaRMC:           1.2,
		RMCmin:            RMCmin, // 0.25
		InitialRMC:        RMCmin,
		RMCmax:            5000000.0, //2.0
		RMCincrease:       1000000.0, // 1.0
		RMCdecrease:       500000.0,  // 0.5
		RMCPeriodUpdate:   30,
	},
	AdversarySettings: &AdversarySettings{
		SimulationMode:   "None",
		DoubleSpendDelay: 5,

		AccidentalMana: []string{"random", "random"},

		AdversaryDelays:     []int{},
		AdversaryTypes:      []int{0, 0},
		AdversaryMana:       []float64{},
		AdversaryNodeCounts: []int{},
		AdversaryInitColors: []string{"R", "B"},
		AdversaryPeeringAll: false,
		AdversarySpeedup:    []float64{1.0, 1.0},

		BlowballMana:    20,
		BlowballSize:    20,
		BlowballDelay:   5,
		BlowballMaxSent: 2,
		BlowballNodeID:  0,
	},
}
