package multiverse

import (
	"math"
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
		tangle.Scheduler.DecreaseNodeAccessMana(tangle.Peer.ID, burn)
		return
	case Greedy:
		burn = tangle.Scheduler.GetMaxManaBurn() + config.ExtraBurn
		manabalance := tangle.Scheduler.GetNodeAccessMana(tangle.Peer.ID)
		burn = math.Min(burn, manabalance) // can't spend more than mana balance
		tangle.Scheduler.DecreaseNodeAccessMana(tangle.Peer.ID, burn)
		return

	case RandomGreedy:
		burn = tangle.Scheduler.GetMaxManaBurn() + config.ExtraBurn*rand.Float64()
		manabalance := tangle.Scheduler.GetNodeAccessMana(tangle.Peer.ID)
		burn = math.Min(burn, manabalance) // can't spend more than mana balance
		tangle.Scheduler.DecreaseNodeAccessMana(tangle.Peer.ID, burn)
		return
	default:
		return 0.0
	}
}
