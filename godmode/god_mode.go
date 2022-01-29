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
	"go.uber.org/atomic"
	"math"
	"sync"
	"time"
)

const (
	divide = 100000
)

var (
	log = logger.New("God mode")
)

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

	updating           typeutils.AtomicBool
	maxIdleTime        time.Duration
	lastMessageIDsUsed map[multiverse.MessageID]types.Empty
	weightCount        *atomic.Uint64
	lastIDMutex        sync.Mutex
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
		enabled:            true,
		weights:            weights,
		adversaryDelay:     adversaryDelay,
		split:              split,
		initialNodeCount:   initialNodeCount,
		godPeerIDs:         make(map[network.PeerID]*network.Peer),
		godNetworkIndex:    initialNodeCount,
		maxIdleTime:        time.Nanosecond,
		lastMessageIDsUsed: make(map[multiverse.MessageID]types.Empty),
		weightCount:        atomic.NewUint64(0),
	}
	return mode
}

// todo honest nodes are processing slower than god nodes, no matter which one are issuing first - god nodes messages still needs to wait for its turn in the socket

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

func (g *GodMode) updateLastMessageIDs(id multiverse.MessageID, weight uint64) {
	g.lastIDMutex.Lock()
	g.lastMessageIDsUsed[id] = types.Void
	g.lastIDMutex.Unlock()
	g.weightCount = atomic.NewUint64(weight)
}

func (g *GodMode) increaseLastMessageWeight(weight uint64) {
	g.weightCount.Add(weight)
}

func (g *GodMode) IssueDoubleSpend() {
	peer1ID, peer2ID := g.supporters.getInitiatorsForDoubleSpend()
	msgRed := g.prepareMessageForDoubleSpend(g.godPeerIDs[peer1ID], multiverse.Red)
	msgBlue := g.prepareMessageForDoubleSpend(g.godPeerIDs[peer2ID], multiverse.Blue)
	// process own message
	go g.processMessageByGodNodes(msgRed)
	go g.processMessageByGodNodes(msgBlue)
	// update god messages used in trackOwnProcessedMessages
	g.updateLastMessageIDs(msgBlue.ID, 0)
	g.updateLastMessageIDs(msgRed.ID, 0)

	// send red to odd peers, and blue to even
	for i, peer := range g.honestPeers() {
		switch i % 2 {
		case 0:
			go peer.ReceiveNetworkMessage(msgRed)
		case 1:
			go peer.ReceiveNetworkMessage(msgBlue)
		}
	}
	log.Debugf("update last %d %d", msgRed.ID, msgBlue.ID)

	g.supporters.UpdateSupportersAfterDoubleSpend(peer1ID, multiverse.Red)
	if peer1ID != peer2ID {
		g.supporters.UpdateSupportersAfterDoubleSpend(peer2ID, multiverse.Blue)
	}
}

// IssueThirdDoubleSpend introduces third color, to have more flexibility when moving adversary support for colors
func (g *GodMode) IssueThirdDoubleSpend(issuerID network.PeerID) {
	msgGreen := g.prepareMessageForDoubleSpend(g.godPeerIDs[issuerID], multiverse.Red)
	log.Debugf("Introducing green color, msgID: %d", msgGreen.ID)
	g.processMessageByGodNodes(msgGreen)
	g.gossipMessageToHonestNodes(msgGreen)
	g.updateLastMessageIDs(msgGreen.ID, 0)
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
}

// listenToAllHonestNodes listen to all honest messages created in the network to update godNodes tangles, and attaches
// to opinion change events of honest nodes, to track opinions in the network and initiates supporters votes change
func (g *GodMode) listenToAllHonestNodes() {
	for _, peer := range g.honestPeers() {
		peerID := peer.ID
		t := peer.Node.(multiverse.NodeInterface).Tangle()
		t.MessageFactory.Events.MessageCreated.Attach(events.NewClosure(g.processMessageByGodNodes))
		t.OpinionManager.Events().OpinionChanged.Attach(events.NewClosure(func(prevOpinion, newOpinion multiverse.Color, weight int64) {
			go g.opinionManager.updateNetworkOpinions(prevOpinion, newOpinion, weight, peerID)
		}))
		t.OpinionManager.Events().GodMessageProcessed.Attach(events.NewClosure(func(processingPeer network.PeerID, id multiverse.MessageID) {
			go g.trackOwnProcessedMessages(processingPeer, id)
		}))
	}
}

// Track if certain portion of honest nodes (based on its weight) have processed last messages. If yes, we can clear
// adversary messages issued during last updateSupport from lastMessageIDsUsed map and trigger next updateSupport.
func (g *GodMode) trackOwnProcessedMessages(processingPeer network.PeerID, id multiverse.MessageID) {
	g.lastIDMutex.Lock()

	if _, ok := g.lastMessageIDsUsed[id]; !ok {
		g.lastIDMutex.Unlock()
		return
	}
	g.lastIDMutex.Unlock()

	peerWeight := g.net.WeightDistribution.Weight(processingPeer)
	log.Debugf("Updating msg %d, peer weight %d", id, peerWeight)

	g.increaseLastMessageWeight(peerWeight)

	honestMana := uint64(config.NodesTotalWeight) - uint64(config.GodMana*config.NodesTotalWeight/100)

	log.Debugf("Hoest mana %d; weight %d, processed weight %d", honestMana, peerWeight, g.lastMessageIDsUsed[id])

	if g.weightCount.Load() > honestMana/3 {
		go g.updateSupport()
		return
	}
}

// updateSupport get current honest opinions state and checks if change of support is needed to keep network
// in the undecided state
func (g *GodMode) updateSupport() {
	if g.updating.IsSet() {
		log.Debugf("Already updating")
		return
	}
	g.supporters.Lock()
	defer g.supporters.Unlock()

	maxOpinion, secondOpinion := g.opinionManager.getMaxSecondOpinions()
	if maxOpinion == multiverse.UndefinedColor {
		log.Debugf("opinions are undefined")
		return
	}

	g.updating.Set()
	defer g.updating.UnSet()
	log.Debugf("Start updating votes")

	log.Debugf("Start current god nodes counts:  U: %d, R: %d, B: %d, G: %d",
		len(g.supporters.supporters[multiverse.UndefinedColor]),
		len(g.supporters.supporters[multiverse.Red]),
		len(g.supporters.supporters[multiverse.Blue]),
		len(g.supporters.supporters[multiverse.Green]),
	)
	log.Debugf("Start - supporters:  U: %d, R: %d, B: %d, G: %d",
		g.supporters.supporters[multiverse.UndefinedColor],
		g.supporters.supporters[multiverse.Red],
		g.supporters.supporters[multiverse.Blue],
		g.supporters.supporters[multiverse.Green],
	)

	g.lastIDMutex.Lock()
	g.lastMessageIDsUsed = make(map[multiverse.MessageID]types.Empty)
	g.lastIDMutex.Unlock()

	maxWeight := g.opinionManager.GetOpinionWeight(maxOpinion)
	secondWeight := g.opinionManager.GetOpinionWeight(secondOpinion)

	log.Debugf("Updating support, single weight %d;  max %s - %d, second %s - %d", g.supporters.singleNodeWeight/divide, maxOpinion.String(), maxWeight/divide, secondOpinion.String(), secondWeight/divide)
	supportersNeeded := g.supporters.CalculateSupportersNumber(maxWeight, secondWeight, maxOpinion, secondOpinion)
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
	log.Debugf("End - supporters:  U: %d, R: %d, B: %d, G: %d",
		g.supporters.supporters[multiverse.UndefinedColor],
		g.supporters.supporters[multiverse.Red],
		g.supporters.supporters[multiverse.Blue],
		g.supporters.supporters[multiverse.Green],
	)
	log.Debugf("End updating - end")
}

// castVotes makes each peer that needs to change its opinion: create a colored message
func (g *GodMode) castVotes(votersForColor ColorPeerMap) {
	for color, voters := range votersForColor {
		log.Debugf("Cast vote color: %s", color.String())
		if color == multiverse.UndefinedColor {
			if len(voters) > 0 {
				log.Warn("casting for undef")
			}
		}
		for peerID := range voters {
			// if third color was not introduced until now issue the double spend first
			if color == multiverse.Green && !g.supporters.thirdColorIntroduced {
				g.IssueThirdDoubleSpend(peerID)
				continue
			}
			peer := g.godPeerIDs[peerID]
			msg := g.prepareMessage(peer, color)
			log.Debugf("Issue msg nr %d, color %s, by node %d", msg.ID, color.String(), peerID)
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
		g.updateLastMessageIDs(msg.ID, 0)
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

func (g *GodSupporters) CalculateSupportersNumber(maxOpinionWeight, secondOpinionWeight uint64, maxOpinion, secondOpinion multiverse.Color) int {
	honestMaxWeight := maxOpinionWeight - uint64(len(g.supporters[maxOpinion]))*g.singleNodeWeight
	honestSecondWeight := secondOpinionWeight - uint64(len(g.supporters[secondOpinion]))*g.singleNodeWeight
	if honestSecondWeight > honestMaxWeight {
		return 0
	}
	if maxOpinionWeight < honestMaxWeight {
		log.Warnf("honestMaxWeight < 0")
		honestMaxWeight = maxOpinionWeight
		honestSecondWeight = secondOpinionWeight
	}
	if secondOpinionWeight < honestSecondWeight {
		log.Warnf("honestSecondWeight < 0")
		honestMaxWeight = maxOpinionWeight
		honestSecondWeight = secondOpinionWeight
	}
	diff := honestMaxWeight - honestSecondWeight
	log.Debugf("Calculate supporters num, maxColor - %d, secColor - %d, diff: %d", maxOpinionWeight/divide, secondOpinionWeight/divide, diff/divide)
	log.Debugf("Calculate supporters num, honest maxColor - %d, secColor - %d, diff: %d", honestMaxWeight/divide, honestSecondWeight/divide, diff/divide)
	if diff > (g.numOfSupporters+1)*g.singleNodeWeight {
		panic("We are lost!")
	}
	// if the diff is higher than half of adv total mana, move all adv mana
	if diff+uint64(len(g.supporters[maxOpinion]))*g.singleNodeWeight > g.singleNodeWeight*g.numOfSupporters/2 {
		log.Debugf("Move all supporters!")
		return int(g.numOfSupporters)
	}

	numberOfSupporters := diff/g.singleNodeWeight + 1
	log.Debugf("Calculate supporters num, number of supporters: %d, single node weight: %d", numberOfSupporters, g.singleNodeWeight/divide)

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

type GodOpinionManager struct {
	networkOpinions                 map[multiverse.Color]uint64
	diffToPrevOpinionFromLastUpdate map[multiverse.Color]int64
	mu                              sync.RWMutex

	singleNodeWeight uint64
	numOfSupporters  uint64
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

	log.Debugf("Network opinions updated for peer %d, U: %d, R: %d, B: %d, G: %d",
		peerID,
		g.networkOpinions[multiverse.UndefinedColor]/divide,
		g.networkOpinions[multiverse.Red]/divide,
		g.networkOpinions[multiverse.Blue]/divide,
		g.networkOpinions[multiverse.Green]/divide,
	)
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
