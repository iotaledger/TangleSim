package simulation

import (
	"fmt"

	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/multiverse"
	"github.com/iotaledger/multivers-simulation/network"
)

// SetupMetrics registers all metrics that are used in the simulation, add any new metric registration here.
func (s *MetricsManager) SetupMetrics() {
	// counters for double spending
	s.ColorCounters.CreateCounter("opinions", s.uRGBColors, int64(config.NodesCount), 0, 0, 0)
	s.ColorCounters.CreateCounter("confirmedNodes", s.uRGBColors)
	s.ColorCounters.CreateCounter("opinionsWeights", s.uRGBColors)
	s.ColorCounters.CreateCounter("likeAccumulatedWeight", s.uRGBColors)
	s.ColorCounters.CreateCounter("processedMessages", s.uRGBColors)
	s.ColorCounters.CreateCounter("requestedMissingMessages", s.uRGBColors)
	s.ColorCounters.CreateCounter("tipPoolSizes", s.uRGBColors)

	s.ColorCounters.CreateCounter("colorUnconfirmed", s.RGBColors)
	s.ColorCounters.CreateCounter("confirmedAccumulatedWeight", s.RGBColors)
	s.ColorCounters.CreateCounter("confirmedAccumulatedWeight", s.RGBColors)
	s.ColorCounters.CreateCounter("unconfirmedAccumulatedWeight", s.RGBColors)

	s.AdversaryCounters.CreateCounter("likeAccumulatedWeight", s.RGBColors)
	s.AdversaryCounters.CreateCounter("opinions", s.RGBColors, int64(s.adversaryNodesCount), 0, 0, 0)
	s.AdversaryCounters.CreateCounter("confirmedNodes", s.RGBColors)
	s.AdversaryCounters.CreateCounter("confirmedAccumulatedWeight", s.RGBColors)

	// all peers and tip pool sizes and processed messages per color
	for _, peerID := range s.allPeerIDs {
		tipCounterName := fmt.Sprint("tipPoolSizes-", peerID)
		processedCounterName := fmt.Sprint("processedMessages-", peerID)
		s.ColorCounters.CreateCounter(tipCounterName, s.uRGBColors)
		s.ColorCounters.CreateCounter(processedCounterName, s.uRGBColors)
	}
	// Initialize the minConfirmedWeight to be the max value (i.e., the total weight)
	s.PeerCounters.CreateCounter("minConfirmedAccumulatedWeight", s.allPeerIDs, int64(config.NodesTotalWeight))
	s.PeerCounters.CreateCounter("unconfirmationCount", s.allPeerIDs, 0)
	s.PeerCounters.CreateCounter("issuedMessages", s.allPeerIDs, 0)
	s.PeerCounters.CreateCounter("confirmedMessageCount", s.watchedPeerIDs)

	s.GlobalCounters.CreateCounter("flips", 0)
	s.GlobalCounters.CreateCounter("honestFlips", 0)
	s.GlobalCounters.CreateCounter("tps", 0)
	s.GlobalCounters.CreateCounter("relevantValidators", 0)
	s.GlobalCounters.CreateCounter("issuedMessages", 0)

}

func (s *MetricsManager) SetupMetricsCollection() {
	for _, p := range s.network.Peers {
		peerID := p.ID

		p.Node.(multiverse.NodeInterface).Tangle().OpinionManager.Events().OpinionChanged.Attach(events.NewClosure(func(oldOpinion multiverse.Color, newOpinion multiverse.Color, weight int64) {
			s.opinionChangedCollectorFunc(oldOpinion, newOpinion, weight, peerID)
		}))
		p.Node.(multiverse.NodeInterface).Tangle().OpinionManager.Events().ColorConfirmed.Attach(events.NewClosure(func(confirmedColor multiverse.Color, weight int64) {
			s.colorConfirmedCollectorFunc(confirmedColor, weight, peerID)
		}))
		p.Node.(multiverse.NodeInterface).Tangle().OpinionManager.Events().ColorUnconfirmed.Attach(events.NewClosure(func(unconfirmedColor multiverse.Color, unconfirmedSupport int64, weight int64) {
			s.colorUnconfirmedCollectorFunc(unconfirmedColor, unconfirmedSupport, weight, peerID)
		}))
		p.Node.(multiverse.NodeInterface).Tangle().OpinionManager.Events().MinConfirmedWeightUpdated.Attach(events.NewClosure(func(minConfirmedWeight int64) {
			s.minConfirmedWeightUpdatedCollectorFunc(minConfirmedWeight, peerID)
		}))
		tipCounterName := fmt.Sprint("tipPoolSizes-", peerID)
		processedCounterName := fmt.Sprint("processedMessages-", peerID)
		p.Node.(multiverse.NodeInterface).Tangle().TipManager.Events.MessageProcessed.Attach(events.NewClosure(
			func(opinion multiverse.Color, tipPoolSize int, processedMessages uint64, issuedMessages int64) {
				s.ColorCounters.Set(tipCounterName, int64(tipPoolSize), opinion)
				s.ColorCounters.Set(processedCounterName, int64(processedMessages), opinion)
				s.PeerCounters.Set("issuedMessages", issuedMessages, peerID)
			}))
	}

	// Here we only monitor the opinion weight of node w/ the highest weight
	highestWeightPeer := s.network.Peers[s.highestWeightPeerID]
	highestWeightPeer.Node.(multiverse.NodeInterface).Tangle().OpinionManager.Events().ApprovalWeightUpdated.Attach(events.NewClosure(
		s.approvalWeightUpdatedCollectorFunc,
	))

	// Here we only monitor the tip pool size of node w/ the highest weight
	highestWeightPeer.Node.(multiverse.NodeInterface).Tangle().TipManager.Events.MessageProcessed.Attach(events.NewClosure(
		s.messageProcessedCollectFunc,
	))
	highestWeightPeer.Node.(multiverse.NodeInterface).Tangle().Requester.Events.Request.Attach(events.NewClosure(
		s.requestMissingMessageCollectFunc,
	))
}

func (s *MetricsManager) opinionChangedCollectorFunc(oldOpinion multiverse.Color, newOpinion multiverse.Color, weight int64, peerID network.PeerID) {
	s.ColorCounters.Add("opinions", -1, oldOpinion)
	s.ColorCounters.Add("opinions", 1, newOpinion)

	s.ColorCounters.Add("likeAccumulatedWeight", -weight, oldOpinion)
	s.ColorCounters.Add("likeAccumulatedWeight", weight, newOpinion)

	// todo implement in simulator
	//r, g, b := getLikesPerRGB(colorCounters, "opinions")
	//if mostLikedColorChanged(r, g, b, &mostLikedColor) {
	//	atomicCounters.Add("flips", 1)
	//}

	if network.IsAdversary(int(peerID)) {
		s.AdversaryCounters.Add("likeAccumulatedWeight", -weight, oldOpinion)
		s.AdversaryCounters.Add("likeAccumulatedWeight", weight, newOpinion)
		s.AdversaryCounters.Add("opinions", -1, oldOpinion)
		s.AdversaryCounters.Add("opinions", 1, newOpinion)
	}

	//ar, ag, ab := getLikesPerRGB(adversaryCounters, "opinions")
	//// honest nodes likes status only, flips
	//if mostLikedColorChanged(r-ar, g-ag, b-ab, &honestOnlyMostLikedColor) {
	//	atomicCounters.Add("honestFlips", 1)
	//}
}

func (s *MetricsManager) colorConfirmedCollectorFunc(confirmedColor multiverse.Color, weight int64, peerID network.PeerID) {
	s.ColorCounters.Add("confirmedNodes", 1, confirmedColor)
	s.ColorCounters.Add("confirmedAccumulatedWeight", weight, confirmedColor)
	if network.IsAdversary(int(peerID)) {
		s.AdversaryCounters.Add("confirmedNodes", 1, confirmedColor)
		s.AdversaryCounters.Add("confirmedAccumulatedWeight", weight, confirmedColor)
	}
}

func (s *MetricsManager) colorUnconfirmedCollectorFunc(unconfirmedColor multiverse.Color, unconfirmedSupport int64, weight int64, peerID network.PeerID) {
	s.ColorCounters.Add("colorUnconfirmed", 1, unconfirmedColor)
	s.ColorCounters.Add("confirmedNodes", -1, unconfirmedColor)

	s.ColorCounters.Add("unconfirmedAccumulatedWeight", weight, unconfirmedColor)
	s.ColorCounters.Add("confirmedAccumulatedWeight", -weight, unconfirmedColor)

	// When the color is unconfirmed, the min confirmed accumulated weight should be reset
	s.PeerCounters.Set("minConfirmedAccumulatedWeight", int64(config.NodesTotalWeight), peerID)

	// Accumulate the unconfirmed count for each node
	s.PeerCounters.Add("unconfirmationCount", 1, peerID)
}

func (s *MetricsManager) minConfirmedWeightUpdatedCollectorFunc(minConfirmedWeight int64, peerID network.PeerID) {
	if s.PeerCounters.Get("minConfirmedAccumulatedWeight", peerID) > minConfirmedWeight {
		s.PeerCounters.Set("minConfirmedAccumulatedWeight", minConfirmedWeight, peerID)
	}
}

func (s *MetricsManager) approvalWeightUpdatedCollectorFunc(opinion multiverse.Color, deltaWeight int64) {
	s.ColorCounters.Add("opinionsWeights", deltaWeight, opinion)
}

func (s *MetricsManager) messageProcessedCollectFunc(opinion multiverse.Color, tipPoolSize int, processedMessages uint64, issuedMessages int64) {
	s.ColorCounters.Set("tipPoolSizes", int64(tipPoolSize), opinion)
	s.ColorCounters.Set("processedMessages", int64(processedMessages), opinion)
	s.GlobalCounters.Set("issuedMessages", issuedMessages)
}

func (s *MetricsManager) requestMissingMessageCollectFunc(messageID multiverse.MessageID) {
	s.ColorCounters.Add("requestedMissingMessages", 1, multiverse.UndefinedColor)
}
