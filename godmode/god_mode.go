package godmode

import (
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/hive.go/types"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/multiverse"
	"github.com/iotaledger/multivers-simulation/network"
	"sync"
	"time"
)

// region GodMode //////////////////////////////////////////////////////////////////////////////////////////////////////

type messagePeerMap map[multiverse.MessageID]map[network.PeerID]types.Empty

type GodMode struct {
	enabled          bool
	weights          []uint64
	adversaryDelay   time.Duration
	split            int
	initialNodeCount int

	net            *network.Network
	godPeerIDs     map[network.PeerID]types.Empty
	seenMessageIDs messagePeerMap
	mu             sync.RWMutex

	godNetworkIndex int
	lastPeerUsed    int
}

func NewGodMode(simulationMode string, weight int, adversaryDelay time.Duration, split int, initialNodeCount int) *GodMode {
	if simulationMode != "God" {
		return &GodMode{enabled: false}
	}
	partialWeight := uint64(weight) / uint64(split) * uint64(config.NodesTotalWeight) / 100
	weights := make([]uint64, split)
	for i := range weights {
		weights[i] = partialWeight
	}

	mode := &GodMode{
		enabled:          true,
		weights:          weights,
		adversaryDelay:   adversaryDelay,
		split:            split,
		initialNodeCount: initialNodeCount,
		seenMessageIDs:   make(messagePeerMap),
		godPeerIDs:       make(map[network.PeerID]types.Empty),
		godNetworkIndex:  initialNodeCount - 1,
	}
	return mode
}

func (g *GodMode) Setup(net *network.Network) {
	if !g.Enabled() {
		return
	}
	g.net = net
	for _, peer := range g.godPeers() {
		g.godPeerIDs[peer.ID] = types.Void
	}
	// needs to be configured before the network start
	g.ListenToAllHonestNodes()
	g.setupGossipEvents()

	return
}

func (g *GodMode) Enabled() bool {
	return g.enabled
}

func (g *GodMode) InitialNodeCount() int {
	return g.initialNodeCount
}

func (g *GodMode) Split() int {
	return g.split
}

func (g *GodMode) Weights() []uint64 {
	return g.weights
}

func (g *GodMode) IsGod(peerID network.PeerID) bool {
	if !g.Enabled() {
		return false
	}
	if _, ok := g.godPeerIDs[peerID]; ok {
		return true
	}
	return false
}

func (g *GodMode) IssueDoubleSpend() {
	//peer1, peer2 := g.chooseThePoorestDoubleSpendTargets()
	peer1, peers2 := g.chooseWealthiestEqualDoubleSpendTargets()

	msgRed := g.prepareMessage(multiverse.Red)
	msgBlue := g.prepareMessage(multiverse.Blue)
	// process own message
	go g.processOwnMessage(msgRed)
	go g.processOwnMessage(msgBlue)
	// send double spend
	go func() {
		peer1.ReceiveNetworkMessage(msgRed)
	}()
	go func() {
		for _, peer := range peers2 {
			peer.ReceiveNetworkMessage(msgBlue)
		}
	}()
}

func (g *GodMode) godPeers() (peers []*network.Peer) {
	if !g.Enabled() {
		return
	}
	return g.net.Peers[g.godNetworkIndex:]
}

func (g *GodMode) honestPeers() (peers []*network.Peer) {
	if !g.Enabled() {
		return
	}
	return g.net.Peers[:g.godNetworkIndex]
}

func (g *GodMode) nextVotingPeer() *network.Peer {
	newIndex := (g.lastPeerUsed + 1) % g.split
	g.lastPeerUsed = newIndex
	return g.godPeers()[newIndex]
}

func (g *GodMode) ListenToAllHonestNodes() {
	for _, peer := range g.honestPeers() {
		t := peer.Node.(multiverse.NodeInterface).Tangle()
		t.MessageFactory.Events.MessageCreated.Attach(events.NewClosure(g.processOwnMessage))
	}
}

func (g *GodMode) setupGossipEvents() {
	// track changes only on the first God peer and issue new opinion if prev opinion changed
	g.godPeers()[0] = g.net.Peers[g.godNetworkIndex]
	g.godPeers()[0].Node.(multiverse.NodeInterface).Tangle().OpinionManager.Events().OpinionChanged.Attach(events.NewClosure(g.issueMessageOnOpinionChange))
	for _, peer := range g.godPeers() {
		peerID := peer.ID
		node := peer.Node.(multiverse.NodeInterface)
		// do not gossip in an honest way
		node.Tangle().Booker.Events.MessageBooked.Detach(events.NewClosure(node.GossipHandler))
		// gossip own message to all nodes with possible delay for honest targets
		node.Tangle().Booker.Events.MessageBooked.Attach(events.NewClosure(
			func(messageID multiverse.MessageID) {
				g.gossipOwnProcessedMessage(messageID, peerID)
			},
		))
	}
}

func (g *GodMode) receiveNetworkMessage(message *multiverse.Message, peer *network.Peer) {
	g.updateSeenMessagesMap(message.ID, peer.ID)
	peer.ReceiveNetworkMessage(message)
}

func (g *GodMode) processOwnMessage(message *multiverse.Message) {
	for _, peer := range g.godPeers() {
		if messageSeen := g.wasMessageSeenByGodNode(message.ID, peer.ID); messageSeen {
			continue
		}
		g.receiveNetworkMessage(message, peer)
	}
}

func (g *GodMode) chooseThePoorestDoubleSpendTargets() (*network.Peer, *network.Peer) {
	// the poorest node that is not an adversary
	peer1 := g.net.Peer(g.godNetworkIndex - 1)
	var peer2 *network.Peer

	for i := g.godNetworkIndex - 2; i >= 0; i-- {
		peer2 = g.net.Peer(i)
		if _, ok := peer1.Neighbors[peer2.ID]; ok {
			continue
		}
		break
	}
	return peer1, peer2
}

func (g *GodMode) chooseWealthiestEqualDoubleSpendTargets() (*network.Peer, []*network.Peer) {
	// the wealthiest node
	peer1 := g.net.Peer(0)
	peer1Weight := g.net.WeightDistribution.Weight(peer1.ID)
	peers2 := make([]*network.Peer, 0)
	// collect target peers with sum of weights closest to the wealthiest one weight
	var accumulatedWeight uint64 = 0
	for _, peer := range g.honestPeers()[1:] {
		weight := g.net.WeightDistribution.Weight(peer.ID)
		accumulatedWeight += weight
		peers2 = append(peers2, peer)
		if accumulatedWeight > peer1Weight {
			break
		}
	}
	return peer1, peers2
}

func (g *GodMode) prepareMessage(color multiverse.Color) *multiverse.Message {
	node := g.nextVotingPeer().Node.(multiverse.NodeInterface)
	msg := node.Tangle().MessageFactory.CreateMessage(color)
	return msg
}

func (g *GodMode) issueMessageOnOpinionChange(previousOpinion, newOpinion multiverse.Color, weight int64) {
	g.nextVotingPeer().ReceiveNetworkMessage(newOpinion)
}

func (g *GodMode) gossipOwnProcessedMessage(messageID multiverse.MessageID, peerID network.PeerID) {
	if !g.Enabled() {
		return
	}
	// need to gossip only once
	if peerID != g.godPeers()[0].ID {
		return
	}
	node := g.godPeers()[0].Node.(multiverse.NodeInterface)
	msg := node.Tangle().Storage.Message(messageID)
	// gossip only your own messages
	if g.IsGod(msg.Issuer) {
		// make sure all god node have the message
		g.processOwnMessage(msg)
		// iterate over all honest nodes
		for _, peer := range g.honestPeers() {
			time.AfterFunc(g.adversaryDelay, func() {
				peer.ReceiveNetworkMessage(msg)
			})
		}
	}
}

func (g *GodMode) updateSeenMessagesMap(messageID multiverse.MessageID, peerId network.PeerID) (updated bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, ok := g.seenMessageIDs[messageID]; !ok {
		g.seenMessageIDs[messageID] = make(map[network.PeerID]types.Empty)
	}
	if _, ok := g.seenMessageIDs[messageID][peerId]; !ok {
		g.seenMessageIDs[messageID][peerId] = types.Void
		updated = true
	}
	return
}

func (g *GodMode) wasMessageSeenByGodNode(messageID multiverse.MessageID, peerId network.PeerID) (seen bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if peerMap, ok := g.seenMessageIDs[messageID]; ok {
		if _, okPeer := peerMap[peerId]; okPeer {
			seen = true
		}
	}
	return
}

// endregion //////////////////////////////////////////////////////////////////////////////////////////////////////
