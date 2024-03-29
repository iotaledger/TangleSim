package network

import (
	"math/rand"
	"time"

	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/logger"

	"github.com/iotaledger/hive.go/crypto"
	"github.com/iotaledger/hive.go/datastructure/set"
)

var log = logger.New("Network")

// region Network //////////////////////////////////////////////////////////////////////////////////////////////////////

type Network struct {
	Peers                 []*Peer
	WeightDistribution    *ConsensusWeightDistribution
	BandwidthDistribution *BandwidthDistribution
	AdversaryGroups       AdversaryGroups
	Attacker              *SingleAttacker
}

func New(option ...Option) (network *Network) {
	log.Debug("Creating Network ...")
	defer log.Info("Creating Network ... [DONE]")

	network = &Network{
		Peers:           make([]*Peer, 0),
		AdversaryGroups: NewAdversaryGroups(),
		Attacker:        NewSingleAttacker(),
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

func (n *Network) Shutdown() {
	for _, peer := range n.Peers {
		peer.Shutdown()
	}
}

func (n *Network) Peer(index int) *Peer {
	return n.Peers[index]
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region Configuration ////////////////////////////////////////////////////////////////////////////////////////////////

type Configuration struct {
	nodes               []*NodesSpecification
	minDelay            time.Duration
	maxDelay            time.Duration
	minPacketLoss       float64
	maxPacketLoss       float64
	peeringStrategy     PeeringStrategy
	adversaryPeeringAll bool
	adversarySpeedup    []float64
	genesisTime         time.Time
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

func (c *Configuration) ExpRandomNetworkDelay() time.Duration {
	return time.Duration(rand.ExpFloat64() * (float64(c.maxDelay+c.minDelay) / 2))
}

func (c *Configuration) RandomPacketLoss() float64 {
	return c.minPacketLoss + crypto.Randomness.Float64()*(c.maxPacketLoss-c.minPacketLoss)
}

func (c *Configuration) CreatePeers(network *Network) {
	log.Debugf("Creating peers ...")
	defer log.Info("Creating peers ... [DONE]")

	network.WeightDistribution = NewConsensusWeightDistribution()
	network.BandwidthDistribution = NewBandwidthDistribution()

	for _, nodesSpecification := range c.nodes {
		nodeWeights := nodesSpecification.ConfigureWeights(network)
		nodeBandwidth := nodesSpecification.ConfigureBandwidth(network)

		for i := 0; i < nodesSpecification.nodeCount; i++ {
			nodeType := HonestNode
			speedupFactor := 1.0
			// this is adversary node
			if groupIndex, ok := AdversaryNodeIDToGroupIDMap[i]; ok {
				nodeType = network.AdversaryGroups[groupIndex].AdversaryType
				speedupFactor = c.adversarySpeedup[groupIndex]
			}
			if IsAttacker(i) {
				nodeType = Blowball
			}
			nodeFactory := nodesSpecification.nodeFactories[nodeType]

			peer := NewPeer(nodeFactory())
			peer.AdversarySpeedup = speedupFactor
			network.Peers = append(network.Peers, peer)
			log.Debugf("Created %s ... [DONE]", peer)

			network.WeightDistribution.SetWeight(peer.ID, nodeWeights[i])
			network.BandwidthDistribution.SetBandwidth(peer.ID, nodeBandwidth[i])
		}
		for _, peer := range network.Peers {
			peer.SetupNode(network.WeightDistribution, network.BandwidthDistribution, c.genesisTime)
			log.Debugf("Setup %s ... [DONE]", peer)
			log.Debugf("%s weight %d bandwidth %f",
				peer,
				network.WeightDistribution.Weight(peer.ID),
				network.BandwidthDistribution.Bandwidth(peer.ID))
		}
	}
}

func (c *Configuration) ConnectPeers(network *Network) {
	log.Debugf("Connecting peers ...")
	defer log.Info("Connecting peers ... [DONE]")

	c.peeringStrategy(network, c)
	if c.adversaryPeeringAll {
		network.AdversaryGroups.ApplyNeighborsAdversaryNodes(network, c)
	}
	network.AdversaryGroups.ApplyNetworkDelayForAdversaryNodes(network)

}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region Option ///////////////////////////////////////////////////////////////////////////////////////////////////////

type Option func(*Configuration)

func Nodes(nodeCount int,
	nodeFactories map[AdversaryType]NodeFactory,
	weightGenerator WeightGenerator,
	bandwidthGenerator BandwidthGenerator,
) Option {
	nodeSpecs := &NodesSpecification{
		nodeCount:          nodeCount,
		nodeFactories:      nodeFactories,
		weightGenerator:    weightGenerator,
		bandwidthGenerator: bandwidthGenerator,
	}

	return func(config *Configuration) {
		config.nodes = append(config.nodes, nodeSpecs)
	}
}

type NodesSpecification struct {
	nodeCount          int
	nodeFactories      map[AdversaryType]NodeFactory
	weightGenerator    WeightGenerator
	bandwidthGenerator BandwidthGenerator
}

func (n *NodesSpecification) ConfigureWeights(network *Network) []uint64 {
	var nodesCount int
	var totalWeight float64
	var nodeWeights []uint64

	switch config.Params.SimulationMode {
	case "Adversary":
		nodesCount, totalWeight = network.AdversaryGroups.CalculateWeightTotalConfig()
		nodeWeights = n.weightGenerator(nodesCount, totalWeight)
		// update adversary groups and get new mana distribution with adversary nodes included
		nodeWeights = network.AdversaryGroups.UpdateAdversaryNodes(nodeWeights)
	case "Accidental":
		nodeWeights = n.weightGenerator(config.Params.NodesCount, float64(config.Params.NodesTotalWeight))
	case "Blowball":
		nodesCount, totalWeight = network.Attacker.CalculateWeightTotalConfig()
		nodeWeights = n.weightGenerator(nodesCount, totalWeight)
		nodeWeights = network.Attacker.UpdateAttackerWeight(nodeWeights)
	default:
		// nodeWeights = n.weightGenerator(config.Params.NodesCount, float64(config.Params.NodesTotalWeight))
		nodeWeights = EqualDistribution(
			config.Params.ValidatorCount,
			config.Params.NodesCount-config.Params.ValidatorCount,
			config.Params.NodesTotalWeight,
		)
	}

	return nodeWeights
}

func (n *NodesSpecification) ConfigureBandwidth(network *Network) []float64 {
	var nodeBandwidth []float64

	switch config.Params.SimulationMode {
	default:
		nodeBandwidth = n.bandwidthGenerator(
			config.Params.ValidatorCount,
			config.Params.NodesCount-config.Params.ValidatorCount,
			float64(float64(config.Params.SchedulingRate)*(config.Params.CommitteeBandwidth)),
			float64(float64(config.Params.SchedulingRate)*(1-config.Params.CommitteeBandwidth)))
	}
	return nodeBandwidth
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

func AdversaryPeeringAll(adversaryPeeringAll bool) Option {
	return func(config *Configuration) {
		config.adversaryPeeringAll = adversaryPeeringAll
	}
}

func AdversarySpeedup(adversarySpeedupFactors []float64) Option {
	return func(config *Configuration) {
		config.adversarySpeedup = adversarySpeedupFactors
	}
}

func GenesisTime(genesisTime time.Time) Option {
	return func(config *Configuration) {
		config.genesisTime = genesisTime
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
		for sourceNodeID, targetNodeIDs := range graph {
			for targetNodeID := range targetNodeIDs {
				randomNetworkDelay := configuration.RandomNetworkDelay()
				randomPacketLoss := configuration.RandomPacketLoss()

				network.Peers[sourceNodeID].Neighbors[PeerID(targetNodeID)] = NewConnection(
					network.Peers[targetNodeID].Socket,
					randomNetworkDelay,
					randomPacketLoss,
					configuration,
				)

				network.Peers[targetNodeID].Neighbors[PeerID(sourceNodeID)] = NewConnection(
					network.Peers[sourceNodeID].Socket,
					randomNetworkDelay,
					randomPacketLoss,
					configuration,
				)

				log.Debugf("Connecting %s <-> %s [network delay (%s), packet loss (%0.4f%%)] ... [DONE]", network.Peers[sourceNodeID], network.Peers[targetNodeID], randomNetworkDelay, randomPacketLoss*100)
			}
		}
		totalNeighborCount := 0
		for _, peer := range network.Peers {
			log.Debugf("%d %d", peer.ID, len(peer.Neighbors))
			totalNeighborCount += len(peer.Neighbors)
		}
		log.Infof("Average number of neighbors: %.1f", float64(totalNeighborCount)/float64(nodeCount))
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
