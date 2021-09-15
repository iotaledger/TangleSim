package network

import (
	"github.com/iotaledger/multivers-simulation/adversary"
	"github.com/iotaledger/multivers-simulation/logger"
	"time"

	"github.com/iotaledger/hive.go/crypto"
	"github.com/iotaledger/hive.go/datastructure/set"
)

var log = logger.New("Network")

// region Network //////////////////////////////////////////////////////////////////////////////////////////////////////

type Network struct {
	Peers              []*Peer
	WeightDistribution *ConsensusWeightDistribution
}

func New(option ...Option) (network *Network) {
	log.Debug("Creating Network ...")
	defer log.Info("Creating Network ... [DONE]")

	network = &Network{
		Peers: make([]*Peer, 0),
	}

	configuration := NewConfiguration(option...)
	configuration.CreatePeers(network)
	configuration.ConnectPeers(network)

	return
}

func (n *Network) RandomPeers(count int) (randomPeers []*Peer) {
	selectedPeers := set.New()
	for len(randomPeers) < count {
		if randomIndex := crypto.Randomness.Intn(len(n.Peers)); selectedPeers.Add(randomIndex) {
			randomPeers = append(randomPeers, n.Peers[randomIndex])
		}
	}

	return
}

func (n *Network) Start() {
	for _, peer := range n.Peers {
		peer.Start()
	}
}

func (n *Network) Shutdown() {
	for _, peer := range n.Peers {
		peer.Shutdown()
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region Configuration ////////////////////////////////////////////////////////////////////////////////////////////////

type Configuration struct {
	nodes           []*NodesSpecification
	minDelay        time.Duration
	maxDelay        time.Duration
	minPacketLoss   float64
	maxPacketLoss   float64
	peeringStrategy PeeringStrategy
}

func NewConfiguration(options ...Option) (configuration *Configuration) {
	configuration = &Configuration{}
	for _, currentOption := range options {
		currentOption(configuration)
	}

	return
}

func (c *Configuration) RandomNetworkDelay() time.Duration {
	return c.minDelay + time.Duration(crypto.Randomness.Float64()*float64(c.maxDelay-c.minDelay))
}

func (c *Configuration) RandomPacketLoss() float64 {
	return c.minPacketLoss + crypto.Randomness.Float64()*(c.maxPacketLoss-c.minPacketLoss)
}

func (c *Configuration) CreatePeers(network *Network) {
	log.Debugf("Creating peers ...")
	defer log.Info("Creating peers ... [DONE]")

	// TODO update node creation based on adversary groups
	adversaryGroups := adversary.NewGroups()
	network.WeightDistribution = NewConsensusWeightDistribution()
	for _, nodesSpecification := range c.nodes {
		nodeWeights := nodesSpecification.weightGenerator(nodesSpecification.nodeCount, adversaryGroups)

		for i := 0; i < nodesSpecification.nodeCount; i++ {
			peer := NewPeer(nodesSpecification.nodeFactories[0]())
			network.Peers = append(network.Peers, peer)
			log.Debugf("Created %s ... [DONE]", peer)

			network.WeightDistribution.SetWeight(peer.ID, nodeWeights[i])
			peer.SetupNode(network.WeightDistribution)
		}
	}
}

func (c *Configuration) ConnectPeers(network *Network) {
	log.Debugf("Connecting peers ...")
	defer log.Info("Connecting peers ... [DONE]")

	c.peeringStrategy(network, c)
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region Option ///////////////////////////////////////////////////////////////////////////////////////////////////////

type Option func(*Configuration)

func Nodes(nodeCount int, nodeFactories []NodeFactory, weightGenerator WeightGenerator) Option {
	nodeSpecs := &NodesSpecification{
		nodeCount:       nodeCount,
		nodeFactories:   nodeFactories,
		adversaryGroups: adversary.NewGroups(),
		weightGenerator: weightGenerator,
	}

	return func(config *Configuration) {
		config.nodes = append(config.nodes, nodeSpecs)
	}
}

type NodesSpecification struct {
	nodeCount       int
	nodeFactories   []NodeFactory
	adversaryGroups adversary.Groups
	weightGenerator WeightGenerator
}

func Delay(minDelay time.Duration, maxDelay time.Duration) Option {
	return func(config *Configuration) {
		config.minDelay = minDelay
		config.maxDelay = maxDelay
	}
}

func PacketLoss(minPacketLoss float64, maxPacketLoss float64) Option {
	return func(config *Configuration) {
		config.minPacketLoss = minPacketLoss
		config.maxPacketLoss = maxPacketLoss
	}
}

func Topology(peeringStrategy PeeringStrategy) Option {
	return func(config *Configuration) {
		config.peeringStrategy = peeringStrategy
	}
}

type PeeringStrategy func(network *Network, options *Configuration)

func WattsStrogatz(meanDegree int, randomness float64) PeeringStrategy {
	if meanDegree%2 != 0 {
		panic("Invalid argument: meanDegree needs to be even")
	}

	return func(network *Network, configuration *Configuration) {
		nodeCount := len(network.Peers)
		graph := make(map[int]map[int]bool)

		for nodeID := 0; nodeID < nodeCount; nodeID++ {
			graph[nodeID] = make(map[int]bool)

			for j := nodeID + 1; j <= nodeID+meanDegree/2; j++ {
				graph[nodeID][j%nodeCount] = true
			}
		}

		for tail, edges := range graph {
			for head := range edges {
				if crypto.Randomness.Float64() < randomness {
					newHead := crypto.Randomness.Intn(nodeCount)
					for newHead == tail || graph[newHead][tail] || edges[newHead] {
						newHead = crypto.Randomness.Intn(nodeCount)
					}

					delete(edges, head)
					edges[newHead] = true
				}
			}
		}

		totalNeighborCount := 0
		for sourceNodeID, targetNodeIDs := range graph {
			log.Debugf("Peer: %s: Number of neighbors: %d", network.Peers[sourceNodeID], len(targetNodeIDs))
			totalNeighborCount += len(targetNodeIDs)
			for targetNodeID := range targetNodeIDs {
				randomNetworkDelay := configuration.RandomNetworkDelay()
				randomPacketLoss := configuration.RandomPacketLoss()

				network.Peers[sourceNodeID].Neighbors[PeerID(targetNodeID)] = NewConnection(
					network.Peers[targetNodeID].Socket,
					randomNetworkDelay,
					randomPacketLoss,
				)

				network.Peers[targetNodeID].Neighbors[PeerID(sourceNodeID)] = NewConnection(
					network.Peers[sourceNodeID].Socket,
					randomNetworkDelay,
					randomPacketLoss,
				)

				log.Debugf("Connecting %s <-> %s [network delay (%s), packet loss (%0.4f%%)] ... [DONE]", network.Peers[sourceNodeID], network.Peers[targetNodeID], randomNetworkDelay, randomPacketLoss*100)
			}
		}
		log.Infof("Average number of neighbors: %.1f", float64(totalNeighborCount)/float64(nodeCount))
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
