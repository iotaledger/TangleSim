package main

import (
	"encoding/csv"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/logger"
	"github.com/iotaledger/multivers-simulation/multiverse"
	"github.com/iotaledger/multivers-simulation/network"
)

var log = logger.New("Simulation")

func main() {
	log.Info("Starting simulation ... [DONE]")
	defer log.Info("Shutting down simulation ... [DONE]")

	testNetwork := network.New(
		network.Nodes(config.NodesCount, multiverse.NewNode, network.ZIPFDistribution(config.ZipfParameter, config.NodesTotalWeight)),
		network.Delay(30*time.Millisecond, 250*time.Millisecond),
		network.PacketLoss(0, 0.05),
		network.Topology(network.WattsStrogatz(4, 1)),
	)
	testNetwork.Start()
	defer testNetwork.Shutdown()

	monitorNetworkState(testNetwork)
	secureNetwork(testNetwork, config.DecelerationFactor)

	time.Sleep(2 * time.Second)

	attackers := testNetwork.RandomPeers(3)
	sendMessage(attackers[0], multiverse.Red)
	sendMessage(attackers[1], multiverse.Blue)
	sendMessage(attackers[2], multiverse.Green)

	time.Sleep(30 * time.Second)
}

var (
	tpsCounter = uint64(0)

	opinions = make(map[multiverse.Color]int)

	confirmedMessageCounter = int64(0)

	opinionMutex sync.Mutex

	relevantValidators int
)

func monitorNetworkState(testNetwork *network.Network) {
	opinions[multiverse.UndefinedColor] = config.NodesCount
	opinions[multiverse.Blue] = 0
	opinions[multiverse.Red] = 0
	opinions[multiverse.Green] = 0

	for _, peer := range testNetwork.Peers {
		peer.Node.(*multiverse.Node).Tangle.OpinionManager.Events.OpinionChanged.Attach(events.NewClosure(func(oldOpinion multiverse.Color, newOpinion multiverse.Color) {
			opinionMutex.Lock()
			defer opinionMutex.Unlock()

			opinions[oldOpinion]--
			opinions[newOpinion]++
		}))

		peer.Node.(*multiverse.Node).Tangle.ApprovalManager.Events.MessageConfirmed.Attach(events.NewClosure(func(nodeID network.PeerID, issuerID network.PeerID, messageID multiverse.MessageID, IssuanceTime time.Time, confirmationTime time.Time, weight uint64) {
			confirmedMessageCounter = atomic.AddInt64(&confirmedMessageCounter, 1)

			record := []string{string(nodeID), string(issuerID), string(messageID), IssuanceTime.String(), confirmationTime.String(), string(weight), string(confirmedMessageCounter)}
			w := csv.NewWriter(os.Stdout)

			if err := w.Write(record); err != nil {
				log.Fatal("error writing record to csv:", err)
			}

			// Write any buffered data to the underlying writer (standard output).
			w.Flush()

			if err := w.Error(); err != nil {
				log.Fatal(err)
			}
		}))
	}

	go func() {
		for range time.Tick(1000 * time.Millisecond) {
			log.Infof("Network Status: %d TPS :: Consensus[ %d Undefined / %d Blue / %d Red / %d Green ] :: %d Nodes :: %d Validators",
				atomic.LoadUint64(&tpsCounter),
				opinions[multiverse.UndefinedColor],
				opinions[multiverse.Blue],
				opinions[multiverse.Red],
				opinions[multiverse.Green],
				config.NodesCount,
				relevantValidators,
			)

			atomic.StoreUint64(&tpsCounter, 0)
		}
	}()
}

func secureNetwork(testNetwork *network.Network, decelerationFactor float64) {
	largestWeight := float64(testNetwork.WeightDistribution.LargestWeight())

	for _, peer := range testNetwork.Peers {
		weightOfPeer := float64(testNetwork.WeightDistribution.Weight(peer.ID))

		if 1000*weightOfPeer <= largestWeight {
			continue
		}

		relevantValidators++

		// Weight: 100, 20, 1
		// TPS: 1000
		// Sleep time: 121/100000, 121/20000, 121/1000
		// Issuing message count per second: 100000/121 + 20000/121 + 1000/121 = 1000

		// Each peer should send messages according to their mana: Fix TPS for example 1000;
		// A node with a x% of mana will issue 1000*x% messages per second
		issuingPeriod := config.NodesTotalWeight / config.TPS / weightOfPeer

		go startSecurityWorker(peer, time.Duration(issuingPeriod*decelerationFactor)*time.Second)
	}
}

func startSecurityWorker(peer *network.Peer, pace time.Duration) {
	for range time.Tick(pace) {
		sendMessage(peer)
	}
}

func sendMessage(peer *network.Peer, optionalColor ...multiverse.Color) {
	atomic.AddUint64(&tpsCounter, 1)

	if len(optionalColor) >= 1 {
		peer.Node.(*multiverse.Node).IssuePayload(optionalColor[0])
	}

	peer.Node.(*multiverse.Node).IssuePayload(multiverse.UndefinedColor)
}
