package multiverse

import (
	"time"

	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/network"
)

type NoScheduler struct {
	tangle *Tangle

	events *SchedulerEvents
}

func (s *NoScheduler) Setup() {
	s.events.MessageScheduled.Attach(events.NewClosure(func(messageID MessageID) {
		s.tangle.Peer.GossipNetworkMessage(s.tangle.Storage.Message(messageID))
		//		log.Debugf("Peer %d Gossiped message %d", s.tangle.Peer.ID, messageID)
	}))
}
func (s *NoScheduler) IncrementAccessMana(float64)                            {}
func (s *NoScheduler) DecreaseNodeAccessMana(network.PeerID, float64) float64 { return 0 }
func (s *NoScheduler) BurnValue(time.Time) (float64, bool)                    { return 0, false }
func (s *NoScheduler) EnqueueMessage(messageID MessageID) {
	s.events.MessageScheduled.Trigger(messageID)
}
func (s *NoScheduler) ScheduleMessage()                         {}
func (s *NoScheduler) Events() *SchedulerEvents                 { return s.events }
func (s *NoScheduler) ReadyLen() int                            { return 0 }
func (s *NoScheduler) NonReadyLen() int                         { return 0 }
func (s *NoScheduler) GetNodeAccessMana(network.PeerID) float64 { return 0 }
func (s *NoScheduler) GetMaxManaBurn() float64                  { return 0 }
func (s *NoScheduler) IssuerQueueLen(network.PeerID) int        { return 0 }
func (s *NoScheduler) Deficit(network.PeerID) float64           { return 0 }
func (s *NoScheduler) RateSetter() bool                         { return true }
