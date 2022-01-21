package godmode

import (
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/hive.go/types"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/logger"
	"github.com/iotaledger/multivers-simulation/multiverse"
	"github.com/iotaledger/multivers-simulation/network"
	"sync"
	"time"
)

var log = logger.New("God mode")

// region GodMode //////////////////////////////////////////////////////////////////////////////////////////////////////

type messagePeerMap map[multiverse.MessageID]map[network.PeerID]types.Empty

type GodMode struct {
	enabled          bool
	weights          []uint64
	adversaryDelay   time.Duration
	split            int
	initialNodeCount int

	godNetworkIndex int
	lastPeerUsed    int

	net            *network.Network
	godPeerIDs     map[network.PeerID]types.Empty
	seenMessageIDs messagePeerMap
	mu             sync.RWMutex

	supporters     *GodSupporters
	opinionManager *GodOpinionManager
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
		godNetworkIndex:  initialNodeCount,
	}
	return mode
}

func (g *GodMode) Setup(net *network.Network) {
	if !g.Enabled() {
		return
	}
	g.net = net
	log.Debugf("Setup GodMode, number of peers:%d, init nodeCount: %d, godNetworkIndex: %d", len(g.net.Peers), g.initialNodeCount, g.godNetworkIndex)
	for _, peer := range g.godPeers() {
		g.godPeerIDs[peer.ID] = types.Void
	}
	// needs to be configured before the network start
	g.listenToAllHonestNodes()
	g.setupGossipEvents()
	g.setupSupporters()
	g.setupOpinionManager()
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
	go g.processMessage(msgRed)
	go g.processMessage(msgBlue)
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

// RemoveAllGodPeeringConnections clears out all connections to and from God nodes.
func (g *GodMode) RemoveAllGodPeeringConnections() {
	if !g.Enabled() {
		return
	}

	for _, peer := range g.godPeers() {
		peer.Neighbors = make(map[network.PeerID]*network.Connection)
	}

	for _, peer := range g.honestPeers() {
		for neighbor := range peer.Neighbors {
			if g.IsGod(neighbor) {
				delete(peer.Neighbors, neighbor)
			}
		}
	}

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

func (g *GodMode) setupSupporters() {
	godIDs := make([]network.PeerID, 0)
	for _, peer := range g.godPeers() {
		godIDs = append(godIDs, peer.ID)
	}
	g.supporters = NewGodSupporters(godIDs)
}

func (g *GodMode) setupOpinionManager() {
	g.opinionManager = NewGodOpinionManager()
}

func (g *GodMode) listenToAllHonestNodes() {
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

func (g *GodMode) processMessage(message *multiverse.Message) {
	for _, peer := range g.godPeers() {
		// filter already seen messages
		if messageSeen := g.wasMessageSeenByGodNode(message.ID, peer.ID); messageSeen {
			continue
		}
		// adversary nodes will receive messages after honest node's network delay
		// thanks to that when they issue change of opinion they do not help to propagate messages through the network
		// without the delay (honest nodes would request missing parents through solidification)
		// and it does not matter for godMode to have most up-to-date tangle
		time.AfterFunc(time.Millisecond*time.Duration(config.MinDelay), func() {
			g.receiveNetworkMessage(message, peer)
		})
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
	if previousOpinion != multiverse.UndefinedColor {
		log.Debugf("Issue msg on opinion change %s %s", previousOpinion.String(), newOpinion.String())
		g.nextVotingPeer().ReceiveNetworkMessage(newOpinion)
	}
}

func (g *GodMode) gossipOwnProcessedMessage(messageID multiverse.MessageID, peerID network.PeerID) {
	// need to gossip only once
	if peerID != g.godPeers()[0].ID {
		return
	}
	node := g.godPeers()[0].Node.(multiverse.NodeInterface)
	msg := node.Tangle().Storage.Message(messageID)
	// gossip only your own messages
	if g.IsGod(msg.Issuer) {
		// make sure all god node have the message
		g.processMessage(msg)
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

// region supporters ///////////////////////////////////////////////////////////////////////////////////////////////////

type colorPeerMap map[multiverse.Color]map[network.PeerID]types.Empty

type GodSupporters struct {
	singleNodeWeight uint64
	supporters       colorPeerMap
	sync.RWMutex
}

func NewGodSupporters(godPeers []network.PeerID) *GodSupporters {
	return &GodSupporters{
		singleNodeWeight: uint64(config.GodMana * config.NodesTotalWeight / config.GodNodeSplit / 100),
		supporters:       createSupportersMap(godPeers),
	}
}

func createSupportersMap(allPeers []network.PeerID) colorPeerMap {
	m := make(map[multiverse.Color]map[network.PeerID]types.Empty)
	colors := multiverse.GetColorsArray()
	for _, color := range colors {
		m[color] = make(map[network.PeerID]types.Empty)
	}
	// all peers added to an undefined color
	for _, peerID := range allPeers {
		m[multiverse.UndefinedColor][peerID] = types.Void
	}
	return m
}

func (g *GodSupporters) CalculateSupportersNumber(maxOpinionWeight, secondOpinionWeight uint64) int {
	diff := maxOpinionWeight - secondOpinionWeight
	if diff <= 0 {
		return 0
	}
	numberOfSupporters := diff/g.singleNodeWeight + 1
	return int(numberOfSupporters)
}

// GetVoters selects supportersNeeded peers to be moved to support secondOpinion. Firstly, opinions are moved from maxOpinion
// if there is still not enough supporters we check if there are any supporters that have not voted yet (from Undefined),
// the last is checked third, the last one left color
func (g *GodSupporters) GetVoters(supportersNeeded int, maxOpinion, secondOpinion multiverse.Color) colorPeerMap {
	supporters := make(map[multiverse.Color]map[network.PeerID]types.Empty)
	// missing supporters for the second color
	missingSupporters := supportersNeeded - len(g.supporters[secondOpinion])
	if missingSupporters <= 0 { // second color has enough supporters already
		// todo check if anything needed here to do
		panic("should it happen?")
		return nil
	}
	movedSupportersCount := 0
	// use max
	movedSupportersCount, done := g.moveSupporters(missingSupporters, movedSupportersCount, supporters, maxOpinion, secondOpinion)
	if done {
		return supporters
	}
	// use any supporters that have not voted yet
	movedSupportersCount, done = g.moveSupporters(missingSupporters, movedSupportersCount, supporters, multiverse.UndefinedColor, secondOpinion)
	if done {
		return supporters
	}
	// check if there are any supporters for the third one color
	if leftColors := multiverse.GetLeftColors([]multiverse.Color{maxOpinion, secondOpinion}); len(leftColors) > 0 {
		leftColor := leftColors[0]
		g.moveSupporters(missingSupporters, movedSupportersCount, supporters, leftColor, secondOpinion)
	}
	return supporters
}

func (g *GodSupporters) moveSupporters(supportersNeeded, movedSupporters int, supporters colorPeerMap, fromOpinion, targetOpinion multiverse.Color) (int, bool) {
	allMoved := false
	for supporter := range g.supporters[fromOpinion] {
		if movedSupporters == supportersNeeded {
			allMoved = true
			break
		}
		supporters[targetOpinion][supporter] = types.Void
		movedSupporters += 1
	}
	// remove moved supporters from fromColor
	for movedSupporter := range supporters[targetOpinion] {
		delete(g.supporters[fromOpinion], movedSupporter)
	}

	return movedSupporters, allMoved
}

func (g *GodSupporters) UpdateSupportersAfterCastVotes(castedVotes colorPeerMap) {
	for color, supporters := range castedVotes {
		for supporterID := range supporters {
			g.supporters[color][supporterID] = types.Void
		}
	}
}

func (g *GodSupporters) MoveLeftVotersFromMaxOpinion(maxOpinion, secondOpinion multiverse.Color, supporters colorPeerMap) {
	missingSupporters := len(g.supporters[maxOpinion])
	if missingSupporters == 0 {
		return
	}
	// check if there is a third color
	if leftColors := multiverse.GetLeftColors([]multiverse.Color{maxOpinion, secondOpinion}); len(leftColors) > 0 {
		leftColor := leftColors[0]
		movedSupportersCount := 0
		g.moveSupporters(missingSupporters, movedSupportersCount, supporters, maxOpinion, leftColor)
	}
}

// endregion //////////////////////////////////////////////////////////////////////////////////////////////////////

// region GodOpinionManager //////////////////////////////////////////////////////////////////////////////////////////////////////

type GodOpinionManager struct {
	networkOpinions map[multiverse.Color]uint64
	mu              sync.RWMutex
	prevMaxOpinion  multiverse.Color
}

func NewGodOpinionManager() *GodOpinionManager {
	opinions := make(map[multiverse.Color]uint64)
	for _, color := range multiverse.GetColorsArray() {
		opinions[color] = 0
	}
	return &GodOpinionManager{
		networkOpinions: opinions,
		prevMaxOpinion:  multiverse.UndefinedColor,
	}
}

func (g *GodOpinionManager) GetOpinionWeight(opinion multiverse.Color) uint64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.networkOpinions[opinion]
}

// updateNetworkOpinions tracks opinion changes in the network, triggered on opinion change of honest nodes only
func (g *GodOpinionManager) updateNetworkOpinions(prevOpinion, newOpinion multiverse.Color, weight uint64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if prevOpinion == multiverse.UndefinedColor {
		g.networkOpinions[newOpinion] += weight
		return
	}
	g.networkOpinions[prevOpinion] -= weight
	g.networkOpinions[newOpinion] += weight
}

func (g *GodOpinionManager) getMaxSecondOpinions() (maxOpinion, secondOpinion multiverse.Color) {
	if len(g.networkOpinions) <= 1 {
		return
	}
	// copy the map
	opinions := make(map[multiverse.Color]uint64)
	for key, value := range g.networkOpinions {
		opinions[key] = value
	}
	maxOpinion = multiverse.GetMaxOpinion(opinions)
	delete(opinions, maxOpinion)
	secondOpinion = multiverse.GetMaxOpinion(opinions)
	return
}

// endregion //////////////////////////////////////////////////////////////////////////////////////////////////////
