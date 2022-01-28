package godmode

import (
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/hive.go/types"
	"github.com/iotaledger/hive.go/typeutils"
	"github.com/iotaledger/multivers-simulation/adversary"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/logger"
	"github.com/iotaledger/multivers-simulation/multiverse"
	"github.com/iotaledger/multivers-simulation/network"
	"math"
	"sync"
	"time"
)

var log = logger.New("God mode")

// region GodMode //////////////////////////////////////////////////////////////////////////////////////////////////////

type GodMode struct {
	enabled          bool
	weights          []uint64
	adversaryDelay   time.Duration
	split            int
	initialNodeCount int

	godNetworkIndex int
	lastPeerUsed    int

	net        *network.Network
	godPeerIDs map[network.PeerID]*network.Peer

	supporters     *GodSupporters
	opinionManager *GodOpinionManager

	updating          typeutils.AtomicBool
	lastTriggeredTime time.Time
	maxIdleTime       time.Duration
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
		godPeerIDs:       make(map[network.PeerID]*network.Peer),
		godNetworkIndex:  initialNodeCount,
		maxIdleTime:      time.Nanosecond,
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
		g.godPeerIDs[peer.ID] = peer
	}
	// needs to be configured before the network start
	g.listenToAllHonestNodes()
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
	peer1, peers2 := g.chooseWealthiestEqualDoubleSpendTargets()

	peer1ID, peer2ID := g.supporters.getInitiatorsForDoubleSpend()
	msgRed := g.prepareMessageForDoubleSpend(g.godPeerIDs[peer1ID], multiverse.Red)
	msgBlue := g.prepareMessageForDoubleSpend(g.godPeerIDs[peer2ID], multiverse.Blue)
	// process own message
	go g.processMessageByGodNodes(msgRed)
	go g.processMessageByGodNodes(msgBlue)
	// send double spend to chosen honest peers
	go func() {
		peer1.ReceiveNetworkMessage(msgRed)
	}()
	go func() {
		for _, peer := range peers2 {
			go peer.ReceiveNetworkMessage(msgBlue)
		}
	}()

	g.supporters.UpdateSupportersAfterDoubleSpend(peer1ID, multiverse.Red)
	g.supporters.UpdateSupportersAfterDoubleSpend(peer2ID, multiverse.Blue)
}

// IssueThirdDoubleSpend introduces third color, to have more flexibility when moving adversary support for colors
func (g *GodMode) IssueThirdDoubleSpend(issuerID network.PeerID) {
	msgGreen := g.prepareMessageForDoubleSpend(g.godPeerIDs[issuerID], multiverse.Red)
	go g.processMessageByGodNodes(msgGreen)
	poorestNode := g.honestPeers()[len(g.honestPeers())-1]
	go poorestNode.ReceiveNetworkMessage(msgGreen)
	g.supporters.thirdColorIntroduced = true
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

func (g *GodMode) setupSupporters() {
	godIDs := make([]network.PeerID, 0)
	for _, peer := range g.godPeers() {
		godIDs = append(godIDs, peer.ID)
	}
	g.supporters = NewGodSupporters(godIDs)
}

func (g *GodMode) setupOpinionManager() {
	g.opinionManager = NewGodOpinionManager()
	g.opinionManager.Events.updateNeeded.Attach(events.NewClosure(g.updateSupport))
}

// listenToAllHonestNodes listen to all honest messages created in the network to update godNodes tangles, and attaches
// to opinion change events of honest nodes, to track opinions in the network and initiates supporters votes change
func (g *GodMode) listenToAllHonestNodes() {
	for _, peer := range g.honestPeers() {
		peerID := peer.ID
		t := peer.Node.(multiverse.NodeInterface).Tangle()
		t.MessageFactory.Events.MessageCreated.Attach(events.NewClosure(g.processMessageByGodNodes))
		t.OpinionManager.Events().OpinionChanged.Attach(events.NewClosure(func(prevOpinion, newOpinion multiverse.Color, weight int64) {
			log.Debugf("Peer %d before updateNetworkOpinions", peerID)
			go g.opinionManager.updateNetworkOpinions(prevOpinion, newOpinion, weight, peerID)
		}))
	}
}

// updateSupport get current honest opinions state and checks if change of support is needed to keep network
// in the undecided state
func (g *GodMode) updateSupport(maxOpinion, secondOpinion multiverse.Color, maxWeight, secondWeight uint64, peerID network.PeerID) {
	//// update only if it is the reaches honest node
	//if peerID != 0 {
	//	log.Debugf("not a richest! ID: %d", peerID)
	//	return
	//}
	if g.updating.IsSet() {
		log.Debugf("Already updating")
		return
	}
	g.supporters.Lock()
	defer g.supporters.Unlock()

	if maxOpinion == multiverse.UndefinedColor {
		log.Debugf("opinions are undefined")
		return
	}
	if peerID < 5 || time.Now().Sub(g.lastTriggeredTime) < g.maxIdleTime {
		return
	}
	g.updating.Set()
	defer g.updating.UnSet()
	log.Debugf("Start updating votes")
	g.lastTriggeredTime = time.Now()

	log.Debugf("Start current setup:  U: %d, R: %d, B: %d, G: %d",
		len(g.supporters.supporters[multiverse.UndefinedColor]),
		len(g.supporters.supporters[multiverse.Red]),
		len(g.supporters.supporters[multiverse.Blue]),
		len(g.supporters.supporters[multiverse.Green]),
	)
	log.Debugf("Updating support, single weight %d;  max %s - %d, second %s - %d", g.supporters.singleNodeWeight/100000, maxOpinion.String(), maxWeight/100000, secondOpinion.String(), secondWeight/100000)
	g.lastTriggeredTime = time.Now()
	supportersNeeded := g.supporters.CalculateSupportersNumber(maxWeight, secondWeight, maxOpinion)
	votersForColor := g.supporters.GetVoters(supportersNeeded, maxOpinion, secondOpinion)
	if votersForColor == nil {
		log.Debugf("End updating - voters nil")
		return
	}
	g.supporters.MoveLeftVotersFromMaxOpinion(maxOpinion, secondOpinion, votersForColor)

	log.Debugf("Votes for color: %v", votersForColor)
	g.castVotes(votersForColor)
	log.Debugf("End of updates - current setup:  U: %d, R: %d, B: %d, G: %d",
		len(g.supporters.supporters[multiverse.UndefinedColor]),
		len(g.supporters.supporters[multiverse.Red]),
		len(g.supporters.supporters[multiverse.Blue]),
		len(g.supporters.supporters[multiverse.Green]),
	)
	log.Debugf("End updating - end")
}

// castVotes makes each peer that needs to change its opinion: create a colored message
func (g *GodMode) castVotes(votersForColor ColorPeerMap) {
	for color, voters := range votersForColor {

		for peerID := range voters {
			// if third color was not introduced until now issue the double spend first
			if color == multiverse.Green && !g.supporters.thirdColorIntroduced {
				g.IssueThirdDoubleSpend(peerID)
				continue
			}
			peer := g.godPeerIDs[peerID]
			msg := g.prepareMessage(peer, color)
			g.processMessageByGodNodes(msg)
			g.gossipMessageToHonestNodes(msg)
		}
	}
	g.supporters.UpdateSupportersAfterCastVotes(votersForColor)
}

func (g *GodMode) processMessageByGodNodes(message *multiverse.Message) {
	for _, peer := range g.godPeers() {
		peer.ReceiveNetworkMessage(message)
	}
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

// prepareMessage creates valid message, it changes nodes opinion to color right before creation
func (g *GodMode) prepareMessage(peer *network.Peer, color multiverse.Color) *multiverse.Message {
	node := peer.Node.(multiverse.NodeInterface)

	// update the opinion in node's opinion manager, so during message creation the right tips will be selected
	adversary.CastAdversary(peer.Node).AssignColor(color)
	msg := node.Tangle().MessageFactory.CreateMessage(multiverse.UndefinedColor)
	return msg
}

func (g *GodMode) prepareMessageForDoubleSpend(peer *network.Peer, color multiverse.Color) *multiverse.Message {
	node := peer.Node.(multiverse.NodeInterface)
	msg := node.Tangle().MessageFactory.CreateMessage(color)
	return msg
}

func (g *GodMode) gossipMessageToHonestNodes(msg *multiverse.Message) {
	// gossip only your own messages
	if g.IsGod(msg.Issuer) {
		// iterate over all honest nodes
		for _, honestPeer := range g.honestPeers() {
			time.AfterFunc(g.adversaryDelay, func() {
				honestPeer.ReceiveNetworkMessage(msg)
			})
		}
	}
}

// endregion //////////////////////////////////////////////////////////////////////////////////////////////////////

// region supporters ///////////////////////////////////////////////////////////////////////////////////////////////////

type ColorPeerMap map[multiverse.Color]map[network.PeerID]types.Empty

type GodSupporters struct {
	singleNodeWeight uint64
	numOfSupporters  uint64

	supporters           ColorPeerMap
	thirdColorIntroduced bool
	sync.RWMutex
}

func NewGodSupporters(godPeers []network.PeerID) *GodSupporters {
	return &GodSupporters{
		singleNodeWeight: uint64(config.GodMana * config.NodesTotalWeight / 100 / config.GodNodeSplit),
		numOfSupporters:  uint64(config.GodNodeSplit),
		supporters:       createSupportersMap(godPeers),
	}
}

func createSupportersMap(allPeers []network.PeerID) ColorPeerMap {
	m := make(ColorPeerMap)
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

func (g *GodSupporters) CalculateSupportersNumber(maxOpinionWeight, secondOpinionWeight uint64, maxOpinion multiverse.Color) int {
	if secondOpinionWeight > maxOpinionWeight {
		return 0
	}
	diff := maxOpinionWeight - secondOpinionWeight
	log.Debugf("Calculate supporters num, maxColor - %d, secColor - %d, diff: %d", maxOpinionWeight/100000, secondOpinionWeight/100000, diff/100000)

	// if the diff is higher than half of adv total mana, move all adv mana
	if diff+uint64(len(g.supporters[maxOpinion]))*g.singleNodeWeight > g.singleNodeWeight*g.numOfSupporters/2 {
		log.Debugf("Move all supporters!")
		return int(g.numOfSupporters)
	}

	numberOfSupporters := diff/g.singleNodeWeight + 2
	log.Debugf("Calculate supporters num, number of supporters: %d, single: %d", numberOfSupporters, g.singleNodeWeight/100000)

	return int(numberOfSupporters)
}

// GetVoters selects supportersNeeded peers to be moved to support secondOpinion. Firstly, opinions are moved from maxOpinion
// if there is still not enough supporters we check if there are any supporters that have not voted yet (from Undefined),
// the last is checked third, the last one left color
func (g *GodSupporters) GetVoters(supportersNeeded int, maxOpinion, secondOpinion multiverse.Color) ColorPeerMap {
	supporters := make(map[multiverse.Color]map[network.PeerID]types.Empty)
	colors := multiverse.GetColorsArray()
	for _, color := range colors {
		supporters[color] = make(map[network.PeerID]types.Empty)
	}
	// missing supporters for the second color
	missingSupporters := supportersNeeded - len(g.supporters[secondOpinion])
	if missingSupporters == 0 {
		return supporters
	}
	if missingSupporters < 0 { // second color has enough supporters already
		log.Info("Too many supporters on second color moving to third one")
		if leftColors := multiverse.GetLeftColors([]multiverse.Color{maxOpinion, secondOpinion}); len(leftColors) > 0 {
			leftColor := leftColors[0]
			g.moveSupporters(-missingSupporters, 0, supporters, secondOpinion, leftColor)
		}
		return supporters
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

func (g *GodSupporters) moveSupporters(supportersNeeded, movedSupporters int, supporters ColorPeerMap, fromOpinion, targetOpinion multiverse.Color) (int, bool) {
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

func (g *GodSupporters) UpdateSupportersAfterCastVotes(castedVotes ColorPeerMap) {
	for color, supporters := range castedVotes {
		for supporterID := range supporters {
			g.supporters[color][supporterID] = types.Void
		}
	}
}

func (g *GodSupporters) MoveLeftVotersFromMaxOpinion(maxOpinion, secondOpinion multiverse.Color, supporters ColorPeerMap) {
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

func (g *GodSupporters) getInitiatorsForDoubleSpend() (network.PeerID, network.PeerID) {
	var peer1, peer2 network.PeerID
	for supporter := range g.supporters[multiverse.UndefinedColor] {
		if peer1 == 0 {
			peer1 = supporter
			continue
		}
		if peer2 == 0 {
			peer2 = supporter
			break
		}
	}
	if peer2 == 0 {
		peer2 = peer1
	}
	return peer1, peer2
}

// UpdateSupportersAfterDoubleSpend updates support by moving supporter from Undefined color to double spend color
func (g *GodSupporters) UpdateSupportersAfterDoubleSpend(peerID network.PeerID, opinion multiverse.Color) {
	delete(g.supporters[multiverse.UndefinedColor], peerID)
	g.supporters[opinion][peerID] = types.Void
}

// endregion //////////////////////////////////////////////////////////////////////////////////////////////////////

// region GodOpinionManager //////////////////////////////////////////////////////////////////////////////////////////////////////

type OpinionManagerEvents struct {
	updateNeeded *events.Event
}

func updateNeededEventCaller(handler interface{}, params ...interface{}) {
	handler.(func(maxOpinion, secondOpinion multiverse.Color, maxWeight, secondWeight uint64, peerID network.PeerID))(params[0].(multiverse.Color), params[1].(multiverse.Color), params[2].(uint64), params[3].(uint64), params[4].(network.PeerID))
}

type GodOpinionManager struct {
	networkOpinions                 map[multiverse.Color]uint64
	diffToPrevOpinionFromLastUpdate map[multiverse.Color]int64
	mu                              sync.RWMutex

	singleNodeWeight uint64
	numOfSupporters  uint64

	Events *OpinionManagerEvents
}

func NewGodOpinionManager() *GodOpinionManager {
	opinions := make(map[multiverse.Color]uint64)
	opinionDiff := make(map[multiverse.Color]int64)
	for _, color := range multiverse.GetColorsArray() {
		opinions[color] = 0
		opinionDiff[color] = 0
	}
	return &GodOpinionManager{
		networkOpinions:                 opinions,
		diffToPrevOpinionFromLastUpdate: opinionDiff,
		singleNodeWeight:                uint64(config.GodMana * config.NodesTotalWeight / 100 / config.GodNodeSplit),
		numOfSupporters:                 uint64(config.GodNodeSplit),
		Events: &OpinionManagerEvents{
			updateNeeded: events.NewEvent(updateNeededEventCaller),
		},
	}
}

func (g *GodOpinionManager) GetOpinionWeight(opinion multiverse.Color) uint64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.networkOpinions[opinion]
}

// updateNetworkOpinions tracks opinion changes in the network, triggered on opinion change of honest nodes only
func (g *GodOpinionManager) updateNetworkOpinions(prevOpinion, newOpinion multiverse.Color, weight int64, peerID network.PeerID) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if prevOpinion != multiverse.UndefinedColor {
		g.networkOpinions[prevOpinion] -= uint64(weight)
		g.diffToPrevOpinionFromLastUpdate[newOpinion] -= weight

	}
	g.networkOpinions[newOpinion] += uint64(weight)
	g.diffToPrevOpinionFromLastUpdate[newOpinion] += weight

	log.Debugf("Network opinions updated, U: %d, R: %d, B: %d, G: %d",
		g.networkOpinions[multiverse.UndefinedColor]/100000,
		g.networkOpinions[multiverse.Red]/100000,
		g.networkOpinions[multiverse.Blue]/100000,
		g.networkOpinions[multiverse.Green]/100000,
	)

	g.triggerUpdateIfNeeded(peerID)
}

// triggerUpdateIfNeeded initiates an update of voters by triggering an updateNeeded event when the difference from previous opinions changed significantly
func (g *GodOpinionManager) triggerUpdateIfNeeded(peerID network.PeerID) {
	for _, diff := range g.diffToPrevOpinionFromLastUpdate {
		// any change comparing to last update is two times higher than single node weight
		if g.ifDiffIsTwiceAsSingleGodWeight(diff) || g.numOfSupporters <= 3 {
			maxOpinion, secondOpinion := g.getMaxSecondOpinions()
			// todo maybe it blocks go routine of honest nodes!
			go g.Events.updateNeeded.Trigger(maxOpinion, secondOpinion, g.networkOpinions[maxOpinion], g.networkOpinions[secondOpinion], peerID)
		}
	}
}

func (g *GodOpinionManager) ifDiffIsTwiceAsSingleGodWeight(diff int64) bool {
	return uint64(math.Abs(float64(diff))) >= 2*g.singleNodeWeight
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
	if g.networkOpinions[maxOpinion] == 0 {
		return multiverse.UndefinedColor, multiverse.UndefinedColor
	}
	if secondOpinion == multiverse.UndefinedColor {
		delete(opinions, secondOpinion)
		secondOpinion = multiverse.GetMaxOpinion(opinions)
	}
	return
}

// endregion //////////////////////////////////////////////////////////////////////////////////////////////////////
