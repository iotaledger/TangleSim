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
	adversaryDelay   time.Duration
	initialNodeCount int
	godNetworkIndex  int
	// the index of the honest node, that is the last tracked whale
	lastWatchedIndex int
	totalWeights     uint64
	godMana          uint64

	net     *network.Network
	godPeer *network.Peer
	// nodes that will only introduce conflicts
	godPeerHelpers []*network.Peer

	opinionManager *GodOpinionManager

	updating typeutils.AtomicBool

	shutdown chan types.Empty
}

func NewGodMode(simulationMode string, weight int, adversaryDelay time.Duration, totalWeight, lastWatchedIndex, initialNodeCount int) *GodMode {
	if simulationMode != "God" {
		return &GodMode{enabled: false}
	}
	totalWeights := uint64(totalWeight)
	godWeight := uint64(weight) * uint64(config.NodesTotalWeight) / 100

	mode := &GodMode{
		enabled:          true,
		godMana:          godWeight - 2,
		totalWeights:     totalWeights,
		adversaryDelay:   adversaryDelay,
		godNetworkIndex:  initialNodeCount,
		initialNodeCount: initialNodeCount,
		lastWatchedIndex: lastWatchedIndex,
		shutdown:         make(chan types.Empty),
	}
	return mode
}

func (g *GodMode) Setup(net *network.Network) {
	if !g.Enabled() {
		return
	}
	g.net = net
	log.Debugf("Setup GodMode, number of peers:%d, godNetworkIndex: %d", len(g.net.Peers), g.godNetworkIndex)
	g.godPeer = g.net.Peer(g.godNetworkIndex)
	g.godPeerHelpers = append(g.godPeerHelpers, g.net.Peer(g.godNetworkIndex+1))
	g.godPeerHelpers = append(g.godPeerHelpers, g.net.Peer(g.godNetworkIndex+2))
	// needs to be configured before the network start
	g.listenToAllHonestNodes()
	g.setupOpinionManager()

	return
}

func (g *GodMode) Run() {
	for {
		select {
		case <-g.shutdown:
			break
		}
	}
}

func (g *GodMode) Shutdown() {

}

func (g *GodMode) Enabled() bool {
	return g.enabled
}

func (g *GodMode) Weights() []uint64 {
	return []uint64{g.godMana, 1, 1}
}

func (g *GodMode) InitialNodeCount() int {
	return g.initialNodeCount
}

func (g *GodMode) IsGod(peerID network.PeerID) bool {
	if !g.Enabled() {
		return false
	}
	if g.godPeer.ID == peerID || g.godPeerHelpers[0].ID == peerID || g.godPeerHelpers[1].ID == peerID {
		return true
	}
	return false
}
func (g *GodMode) IssueDoubleSpend() {
	target1, targets2 := g.chooseWealthiestEqualDoubleSpendTargets()

	peer1 := g.godPeerHelpers[0]
	peer2 := g.godPeerHelpers[1]
	msgRed := g.prepareMessageForDoubleSpend(peer1, multiverse.Red)
	msgBlue := g.prepareMessageForDoubleSpend(peer2, multiverse.Blue)
	// process own message
	go g.processMessageByGodNode(msgRed)
	go g.processMessageByGodNode(msgBlue)
	// send double spend to chosen honest peers
	go func() {
		target1.ReceiveGodMessageBackDoor(msgRed)
	}()
	go func() {
		for _, peer := range targets2 {
			peer.ReceiveGodMessageBackDoor(msgBlue)
		}
	}()

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
func (g *GodMode) IssueDoubleSpendToAllAtOnce() {
	peer1 := g.godPeerHelpers[0]
	peer2 := g.godPeerHelpers[1]
	msgRed := g.prepareMessageForDoubleSpend(peer1, multiverse.Red)
	msgBlue := g.prepareMessageForDoubleSpend(peer2, multiverse.Blue)
	// process own message
	go g.processMessageByGodNode(msgRed)
	go g.processMessageByGodNode(msgBlue)

	// send red to odd peers, and blue to even
	for i, peer := range g.honestPeers() {
		switch i % 2 {
		case 0:
			go peer.ReceiveGodMessageBackDoor(msgRed)
		case 1:
			go peer.ReceiveGodMessageBackDoor(msgBlue)
		}
	}
	// send second color
	for i, peer := range g.honestPeers() {
		switch i % 2 {
		case 0:
			go peer.ReceiveGodMessageBackDoor(msgBlue)
		case 1:
			go peer.ReceiveGodMessageBackDoor(msgRed)
		}
	}
	log.Debugf("update last %d %d", msgRed.ID, msgBlue.ID)
}

func (g *GodMode) GodPeer() *network.Peer {
	if !g.Enabled() {
		return nil
	}
	return g.net.Peers[g.godNetworkIndex]
}

func (g *GodMode) honestPeers() (peers []*network.Peer) {
	if !g.Enabled() {
		return
	}
	return g.net.Peers[:g.godNetworkIndex]
}

func (g *GodMode) setupOpinionManager() {
	g.opinionManager = NewGodOpinionManager(g.godMana, g.totalWeights, g.lastWatchedIndex+1)
	g.opinionManager.Events.updateNeeded.Attach(events.NewClosure(g.updateSupport))
}

// listenToAllHonestNodes listen to all honest messages created in the network to update godNodes tangles, and attaches
// to opinion change events of honest nodes, to track opinions in the network and initiates supporters votes change
func (g *GodMode) listenToAllHonestNodes() {
	for _, peer := range g.honestPeers() {
		peerID := peer.ID
		t := peer.Node.(multiverse.NodeInterface).Tangle()
		t.MessageFactory.Events.MessageCreated.Attach(events.NewClosure(g.processMessageByGodNode))
		t.OpinionManager.Events().OpinionChanged.Attach(events.NewClosure(func(prevOpinion, newOpinion multiverse.Color, weight int64) {
			go g.opinionManager.UpdateNetworkOpinions(prevOpinion, newOpinion, weight, peerID)
		}))
	}

	// listen to the richest whale
	whale := g.net.Peer(0)
	t := whale.Node.(multiverse.NodeInterface).Tangle()
	t.OpinionManager.Events().ApprovalWeightUpdated.Attach(events.NewClosure(func(opinion multiverse.Color, weight int64) {
		//go g.opinionManager.updateWhaleOpinion(opinion, weight)
	}))
}

// RemoveAllGodPeeringConnections clears out all connections to and from God nodes.
func (g *GodMode) RemoveAllGodPeeringConnections() {
	if !g.Enabled() {
		return
	}
	g.godPeer.Neighbors = make(map[network.PeerID]*network.Connection)
	g.godPeerHelpers[0].Neighbors = make(map[network.PeerID]*network.Connection)
	g.godPeerHelpers[0].Neighbors = make(map[network.PeerID]*network.Connection)
	for _, peer := range g.honestPeers() {
		for neighbor := range peer.Neighbors {
			if g.IsGod(neighbor) {
				delete(peer.Neighbors, neighbor)
			}
		}
	}

}

// updateSupport get current honest opinions state and checks if change of support is needed to keep network
// in the undecided state
func (g *GodMode) updateSupport(opinion multiverse.Color) {
	if g.updating.IsSet() {
		log.Debugf("Already updating")
		return
	}
	g.updating.Set()
	defer g.updating.UnSet()
	log.Debugf("START  updating votes")

	msg := g.prepareMessage(g.godPeer, opinion)
	g.processMessageByGodNode(msg)
	g.gossipMessageToHonestNodes(msg)
	g.opinionManager.updateGodSupport(opinion)
	log.Debugf("End updating - end")
}

func (g *GodMode) processMessageByGodNode(message *multiverse.Message) {
	g.GodPeer().ReceiveNetworkMessage(message)
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
				honestPeer.ReceiveGodMessageBackDoor(msg)
			})
		}
	}
}

// endregion //////////////////////////////////////////////////////////////////////////////////////////////////////

// region GodOpinionManager //////////////////////////////////////////////////////////////////////////////////////////////////////

type maxSecondOpinion struct {
	maxOpinion    multiverse.Color
	secondOpinion multiverse.Color
	maxWeight     uint64
	secondWeight  uint64
}

func NewMaxSecondOpinion() *maxSecondOpinion {
	return &maxSecondOpinion{
		maxOpinion:    multiverse.UndefinedColor,
		secondOpinion: multiverse.UndefinedColor,
		maxWeight:     0,
		secondWeight:  0,
	}
}

type OpinionManagerEvents struct {
	updateNeeded *events.Event
}

func updateNeededEventCaller(handler interface{}, params ...interface{}) {
	handler.(func(newOpinion multiverse.Color))(params[0].(multiverse.Color))
}

type GodOpinionManager struct {
	networkOpinions map[multiverse.Color]int64
	whalesOpinions  []map[multiverse.Color]int64
	godOpinion      multiverse.Color
	godWeight       uint64
	mu              sync.RWMutex

	upperThreshold uint64
	lowerThreshold uint64
	whalesNum      int
	Events         *OpinionManagerEvents
}

func NewGodOpinionManager(godMana, totalMana uint64, whalesNum int) *GodOpinionManager {
	opinions := createEmptyColorMap()
	whalesOpinions := make([]map[multiverse.Color]int64, whalesNum)

	for i := range whalesOpinions {
		whalesOpinions[i] = createEmptyColorMap()
	}
	return &GodOpinionManager{
		networkOpinions: opinions,
		whalesOpinions:  whalesOpinions,
		whalesNum:       whalesNum,
		godOpinion:      multiverse.UndefinedColor,
		godWeight:       godMana,
		upperThreshold:  (totalMana + godMana) / 2,
		lowerThreshold:  (totalMana - godMana) / 2,
		Events: &OpinionManagerEvents{
			updateNeeded: events.NewEvent(updateNeededEventCaller),
		},
	}
}

func (g *GodOpinionManager) getOpinionWeight(opinion multiverse.Color) uint64 {
	o := g.networkOpinions[opinion]
	if g.networkOpinions[opinion] < 0 {
		o = 0
	}
	return uint64(o)
}

// UpdateNetworkOpinions tracks opinion changes in the network, triggered on opinion change of honest nodes only
func (g *GodOpinionManager) UpdateNetworkOpinions(prevOpinion, newOpinion multiverse.Color, weight int64, peerID network.PeerID) {

	//log.Debugf("event params %s, %s, %d", prevOpinion, newOpinion, weight/divide)
	//log.Debugf("bef totWeight %d, god weight %d, udf %d red %d, blue %d", config.NodesTotalWeight/divide, g.godWeight/divide, g.networkOpinions[multiverse.UndefinedColor]/divide, g.networkOpinions[multiverse.Red]/divide, g.networkOpinions[multiverse.Blue]/divide)

	g.UpdateOpinionsMap(prevOpinion, newOpinion, weight, g.networkOpinions)
	g.mu.RLock()
	opinions := g.copyOpinionMap(g.networkOpinions)
	g.mu.RUnlock()
	r := float64(opinions[multiverse.Red]) / float64(config.NodesTotalWeight) * 100
	b := float64(opinions[multiverse.Blue]) / float64(config.NodesTotalWeight) * 100
	u := float64(opinions[multiverse.UndefinedColor]) / float64(config.NodesTotalWeight) * 100
	log.Debugf("totWeight %d, god weight %d, udf %d red %d, blue %d", config.NodesTotalWeight/divide, g.godWeight/divide, opinions[multiverse.UndefinedColor]/divide, opinions[multiverse.Red]/divide, opinions[multiverse.Blue]/divide)
	log.Debugf("--- Network opinions updated for peer %d, U: %.2f R: %.2f, B: %.2f", peerID, u, r, b)
	log.Debugf("GodColor: %s", g.godOpinion)
	if opinions[multiverse.Red]+opinions[multiverse.Blue]+opinions[multiverse.UndefinedColor] > int64(config.NodesTotalWeight)+2 {
		log.Debugf("Diff r-b: %d", opinions[multiverse.Red]-opinions[multiverse.Blue], opinions[multiverse.UndefinedColor])
		panic("something is not adding up")
	}
	//whaleIndex := int(peerID)
	//if g.isWhale(whaleIndex) {
	//	g.UpdateOpinionsMap(prevOpinion, newOpinion, weight, g.whalesOpinions[whaleIndex])
	//}

	g.checkOpinionsStatus()
}

//
//// updateWhaleOpinion  tracks opinion changes in the network, triggered on opinion change of honest nodes only
//func (g *GodOpinionManager) updateWhaleOpinion(opinion multiverse.Color, weight int64) {
//	g.mu.Lock()
//	defer g.mu.Unlock()
//
//	//g.whalesOpinions[opinion] += weight
//}
//
////// updateNetworkOpinions tracks opinion changes in the network, triggered on opinion change of honest nodes only
//func (g *GodOpinionManager) getWhaleOpinion(color multiverse.Color, whaleIdx int) int64 {
//	g.mu.RLock()
//	defer g.mu.RUnlock()
//
//	return g.whalesOpinions[whaleIdx][color]
//}

func (g *GodOpinionManager) getMaxSecondOpinions() *maxSecondOpinion {
	g.mu.RLock()
	defer g.mu.RUnlock()

	maxOpinion := multiverse.UndefinedColor
	secondOpinion := multiverse.UndefinedColor
	ms := NewMaxSecondOpinion()
	if len(g.networkOpinions) <= 1 {
		return nil
	}
	// copy the map
	opinions := g.copyOpinionMap(g.networkOpinions)
	delete(opinions, multiverse.UndefinedColor)
	delete(opinions, multiverse.Green)

	maxOpinion = GetMaxOpinion(opinions)
	if opinions[maxOpinion] == 0 {
		return ms
	}

	delete(opinions, maxOpinion)
	secondOpinion = GetMaxOpinion(opinions)

	ms = &maxSecondOpinion{
		maxOpinion:    maxOpinion,
		secondOpinion: secondOpinion,
		maxWeight:     g.getOpinionWeight(maxOpinion),
		secondWeight:  g.getOpinionWeight(secondOpinion),
	}
	return ms
}

func (g *GodOpinionManager) copyOpinionMap(mapToCopy map[multiverse.Color]int64) map[multiverse.Color]int64 {
	m := make(map[multiverse.Color]int64)
	for key, value := range mapToCopy {
		m[key] = value
	}
	return m
}

func (g *GodOpinionManager) checkOpinionsStatus() {
	ms := g.getMaxSecondOpinions()
	log.Debugf("--- check opinions")
	if g.godOpinion != ms.maxOpinion && g.godOpinion != multiverse.UndefinedColor {
		log.Debugf("there is nothing I can do...")
		return
	}

	if g.isMaxOverThreshold(ms, g.upperThreshold) {
		log.Debugf("Max opinion is over the upper threshold, triggering...")
		g.triggerVote(ms.secondOpinion)
		return
	}
	if g.isMaxOverThreshold(ms, g.lowerThreshold) {
		if ms.maxWeight-ms.secondWeight > g.godWeight/2 {
			log.Debugf("Diff is greater than 1/godWeight: diff %d, half god %d, max %d, sec %d", ms.maxWeight-ms.secondWeight, g.godWeight/2, ms.maxWeight, ms.secondWeight)
			g.triggerVote(ms.secondOpinion)
			return
		}
	}
	log.Debugf(":END)")
	//g.triggerVote(ms.secondOpinion)
}

// maxColor is far from upper threshold but difference between max and second is too big
//if g.isMaxBelowThreshold(ms, g.lowerThreshold) {
//if ms.maxWeight-ms.secondWeight > g.godWeight/2 {
//	log.Debugf("Diff is greater than 1/godWeight: diff %d, half god %d, max %d, sec %d", ms.maxWeight-ms.secondWeight, g.godWeight/2, ms.maxWeight, ms.secondWeight)
//	g.triggerVote(ms.secondOpinion)
//	return
//}
//if g.godOpinion != ms.maxOpinion {
//	g.triggerVote(ms.secondOpinion)
//

func (g *GodOpinionManager) triggerVote(opinion multiverse.Color) {
	if g.godOpinion == opinion {
		log.Debug("god already support this opinion")
		return
	}
	log.Debugf("Trigger vote!")
	go g.Events.updateNeeded.Trigger(opinion)
}

func (g *GodOpinionManager) isMaxOverThreshold(maxSec *maxSecondOpinion, threshold uint64) bool {
	log.Debugf("max: %d, threshold %d", maxSec.maxWeight, threshold)
	if maxSec.maxWeight > threshold {
		return true
	}
	return false
}

func (g *GodOpinionManager) isMaxBelowThreshold(maxSec *maxSecondOpinion, threshold uint64) bool {
	if maxSec.maxWeight < threshold {
		return true
	}
	return false
}

func (g *GodOpinionManager) updateGodSupport(opinion multiverse.Color) {
	g.UpdateOpinionsMap(g.godOpinion, opinion, int64(g.godWeight), g.networkOpinions)
	g.godOpinion = opinion
}

func (g *GodOpinionManager) isWhale(index int) bool {
	return index < g.whalesNum
}

func (g *GodOpinionManager) UpdateOpinionsMap(prevOpinion, newOpinion multiverse.Color, weight int64, opinions map[multiverse.Color]int64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	opinions[prevOpinion] -= weight
	opinions[newOpinion] += weight
}

func (g *GodOpinionManager) getOpinion(color multiverse.Color) int64 {
	return g.networkOpinions[color]
}

func GetMaxOpinion(aw map[multiverse.Color]int64) multiverse.Color {
	maxApprovalWeight := int64(0)
	maxOpinion := multiverse.UndefinedColor
	for color, approvalWeight := range aw {
		if color == multiverse.UndefinedColor {
			continue
		}
		if approvalWeight > maxApprovalWeight || approvalWeight == maxApprovalWeight && color < maxOpinion || maxOpinion == multiverse.UndefinedColor {
			maxApprovalWeight = approvalWeight
			maxOpinion = color
		}
	}
	return maxOpinion
}

func createEmptyColorMap() map[multiverse.Color]int64 {
	m := make(map[multiverse.Color]int64)
	for _, color := range multiverse.GetColorsArray() {
		m[color] = 0
	}
	m[multiverse.UndefinedColor] = int64(config.NodesTotalWeight)
	return m
}

// endregion //////////////////////////////////////////////////////////////////////////////////////////////////////
