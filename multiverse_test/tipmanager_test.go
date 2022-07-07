package multiverse

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/logger"
	"github.com/iotaledger/multivers-simulation/multiverse"
	"github.com/iotaledger/multivers-simulation/network"
	"github.com/stretchr/testify/assert"
)

var (
	log                     = logger.New("TestTipManager")
	nodeCount               = 10
	confirmedMessageCounter = int64(0)
)

func TestTipManager(t *testing.T) {
	log.Info("Starting TipManager Test ... [DONE]")
	defer log.Info("Shutting down TipManager Test ... [DONE]")
	nodeFactories := map[network.SpecialNodeType]network.NodeFactory{
		network.HonestNode: network.NodeClosure(multiverse.NewNode),
	}
	testNetwork := network.New(
		network.Nodes(nodeCount, nodeFactories, network.ZIPFDistribution(config.ZipfParameter)),
		network.Delay(30*time.Millisecond, 250*time.Millisecond),
		network.PacketLoss(0, 0.05),
		network.Topology(network.WattsStrogatz(4, 1)),
	)
	testNetwork.Start()
	defer testNetwork.Shutdown()

	monitorNetworkState(testNetwork)
	secureNetwork(testNetwork, config.DecelerationFactor)

	time.Sleep(30 * time.Second)
	log.Info("Number of confirmed message for node 0: ", confirmedMessageCounter)
	assert.NotZero(t, confirmedMessageCounter, "No messages are confirmed!")
}

func monitorNetworkState(testNetwork *network.Network) {

	for _, id := range config.MonitoredAWPeers {
		awPeer := testNetwork.Peers[id]
		awPeer.Node.(multiverse.NodeInterface).Tangle().ApprovalManager.Events.MessageConfirmed.Attach(
			events.NewClosure(func(message *multiverse.Message, messageMetadata *multiverse.MessageMetadata, weight uint64) {
				atomic.AddInt64(&confirmedMessageCounter, 1)
			}))
	}

	return
}

func secureNetwork(testNetwork *network.Network, decelerationFactor int) {
	largestWeight := float64(testNetwork.WeightDistribution.LargestWeight())

	for _, peer := range testNetwork.Peers {
		weightOfPeer := float64(testNetwork.WeightDistribution.Weight(peer.ID))

		if 1000*weightOfPeer <= largestWeight {
			continue
		}

		issuingPeriod := float64(config.NodesTotalWeight) / float64(config.TPS) / weightOfPeer
		pace := time.Duration(issuingPeriod * float64(decelerationFactor) * float64(time.Second))
		go startSecurityWorker(peer, pace)
	}
}

func startSecurityWorker(peer *network.Peer, pace time.Duration) {
	for range time.Tick(pace) {
		sendMessage(peer)
	}
}

func sendMessage(peer *network.Peer, optionalColor ...multiverse.Color) {
	peer.Node.(*multiverse.Node).IssuePayload(multiverse.UndefinedColor)
}
