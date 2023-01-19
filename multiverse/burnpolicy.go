package multiverse

import (
	"math/rand"

	"github.com/iotaledger/multivers-simulation/config"
)

type BurnPolicyType int

const (
	NoBurn       BurnPolicyType = 0
	Anxious      BurnPolicyType = 1
	Greedy       BurnPolicyType = 2
	RandomGreedy BurnPolicyType = 3
)

func BurnMana(tangle *Tangle) (burn float64) {
	switch policy := config.BurnPolicy[tangle.Peer.ID]; BurnPolicyType(policy) {
	case NoBurn:
		return 0.0
	case Anxious:
		if tangle.Scheduler.accessMana > 0 {
			return tangle.Scheduler.accessMana
		} else {
			return 1.0
		}
	case Greedy:
		if !tangle.Scheduler.IsEmpty() {
			return tangle.Scheduler.GetMaxManaBurn() + config.ExtraBurn
		} else {
			return 1.0
		}
	case RandomGreedy:
		if !tangle.Scheduler.IsEmpty() {
			return tangle.Scheduler.GetMaxManaBurn() + config.ExtraBurn*rand.Float64()

		} else {
			return 1.0
		}
	default:
		return 0.0
	}
}
