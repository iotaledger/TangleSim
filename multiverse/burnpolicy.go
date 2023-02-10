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
	switch policy := config.BurnPolicies[tangle.Peer.ID]; BurnPolicyType(policy) {
	case NoBurn:
		return 0.0
	case Anxious:
		return tangle.Scheduler.GetNodeAccessMana(tangle.Peer.ID)
	case Greedy:
		return tangle.Scheduler.GetMaxManaBurn() + config.ExtraBurn
	case RandomGreedy:
		return tangle.Scheduler.GetMaxManaBurn() + config.ExtraBurn*rand.Float64()
	default:
		return 0.0
	}
}
