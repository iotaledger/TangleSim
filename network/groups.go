package network

import (
	"github.com/iotaledger/multivers-simulation/config"
	"time"
)

type AdversaryType int

const (
	HonestOpinion AdversaryType = iota
	ShiftOpinion
	TheSameOpinion
)

func ToType(adv int) AdversaryType {
	switch adv {
	case int(ShiftOpinion):
		return ShiftOpinion
	case int(TheSameOpinion):
		return TheSameOpinion
	default:
		return HonestOpinion
	}
}

type AdversaryGroups []*AdversaryGroup

type AdversaryGroup struct {
	NodeIDs       []int
	GroupMana     float64
	TargetMana    float64
	Delay         time.Duration
	AdversaryType AdversaryType
}

func NewAdversaryGroups() (groups AdversaryGroups) {
	groups = make(AdversaryGroups, 0, len(config.AdversaryTypes))
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

		groups = append(groups, &AdversaryGroup{
			NodeIDs:       make([]int, 0),
			TargetMana:    targetMana,
			Delay:         time.Millisecond * time.Duration(delay),
			AdversaryType: ToType(configAdvType),
		})
	}
	return
}
