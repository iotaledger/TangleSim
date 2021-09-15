package adversary

import (
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/multiverse"
	"github.com/iotaledger/multivers-simulation/network"
	"time"
)

type Node struct {
	Peer    *network.Peer
	Tangle  *multiverse.Tangle
	AdvType Type
}

type Type int

const (
	HonestOpinion Type = iota
	ShiftOpinion
	TheSameOpinion
)

func ToType(adv int) Type {
	switch adv {
	case ShiftOpinion:
		return ShiftOpinion
	case TheSameOpinion:
		return TheSameOpinion
	default:
		return HonestOpinion
	}
}

type Groups []*Group

type Group struct {
	NodeIDs       []int
	GroupMana     float64
	TargetMana    float64
	Delay         time.Duration
	AdversaryType Type
}

func NewGroups() (groups Groups) {
	groups = make(Groups, 0, len(config.AdversaryTypes))
	for i, configAdvType := range config.AdversaryTypes {
		// -1 indicates that there is no target mana provided, adversary will be selected randomly
		targetMana := float64(-1)
		delay := config.MinDelay

		if len(config.AdversaryMana) > 0 {
			targetMana = config.AdversaryMana[i]
		}

		if len(config.AdversaryDelays) > 0 {
			delay = config.AdversaryDelays[i]
		}

		groups = append(groups, &Group{
			NodeIDs:       make([]int, 0),
			TargetMana:    targetMana,
			Delay:         time.Millisecond * time.Duration(delay),
			AdversaryType: ToType(configAdvType),
		})
	}
	return
}
