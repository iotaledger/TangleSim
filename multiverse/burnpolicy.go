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
		burn = tangle.Scheduler.GetNodeAccessMana(tangle.Peer.ID)
		return
	case Greedy:
		burn = tangle.Scheduler.GetMaxManaBurn() + config.ExtraBurn
		return
	case RandomGreedy:
		burn = tangle.Scheduler.GetMaxManaBurn() + config.ExtraBurn*rand.Float64()
		return
	default:
		return 0.0
	}
}
