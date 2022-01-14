package adversary

import (
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/multiverse"
	"github.com/iotaledger/multivers-simulation/network"
	"sync"
	"time"
)

// region GodMode //////////////////////////////////////////////////////////////////////////////////////////////////////

func SetupGoMode(net *network.Network) (godMode *GodMode) {
	// needs to be configured before the network start
	if config.SimulationMode == "God" {
		godMode = NewGodMode(net, config.GodDelay)
		godMode.ListenToAllNodes(net)
		godMode.SetupGossipEvents()
	}
	return
}

type GodMode struct {
	net             *network.Network
	peer            *network.Peer
	adversaryDelay  time.Duration
	seenMessageIDs  map[multiverse.MessageID]*multiverse.Message
	mu              sync.Mutex
	godNetworkIndex int
}

func NewGodMode(net *network.Network, adversaryDelay time.Duration) *GodMode {
	godNetworkIndex := len(net.Peers) - 1
	peer := net.Peer(godNetworkIndex)
	//godNode := NewGodNode(peer.Node.(*ShiftingOpinionNode))
	mode := &GodMode{
		net:             net,
		peer:            peer,
		adversaryDelay:  adversaryDelay,
		seenMessageIDs:  make(map[multiverse.MessageID]*multiverse.Message),
		godNetworkIndex: godNetworkIndex,
	}
	return mode
}

func (g *GodMode) IsGod(peer network.PeerID) bool {
	return g.peer.ID == peer
}

func (g *GodMode) ListenToAllNodes(net *network.Network) {
	for _, peer := range g.net.Peers[:g.godNetworkIndex] {
		t := peer.Node.(multiverse.NodeInterface).Tangle()
		t.MessageFactory.Events.MessageCreated.Attach(events.NewClosure(g.onMessageStored))
	}
}

func (g *GodMode) SetupGossipEvents() {
	node := g.peer.Node.(multiverse.NodeInterface)
	node.Tangle().OpinionManager.Events().OpinionChanged.Attach(events.NewClosure(g.issueMessageOnOpinionChange))
	// do not gossip in an honest way
	node.Tangle().Booker.Events.MessageBooked.Detach(events.NewClosure(node.GossipHandler))
	// gossip to all nodes on messageProcessed event
	node.Tangle().Booker.Events.MessageBooked.Attach(events.NewClosure(g.gossipOwnProcessedMessage))
}

func (g *GodMode) IssueDoubleSpend() {
	//peer1, peer2 := g.chooseThePoorestDoubleSpendTargets()
	peer1, peers2 := g.chooseWealthiestEqualDoubleSpendTargets()

	msgRed := g.PrepareMessage(multiverse.Red)
	msgBlue := g.PrepareMessage(multiverse.Blue)
	// process own message
	go g.peer.ReceiveNetworkMessage(msgRed)
	go g.peer.ReceiveNetworkMessage(msgBlue)
	// send double spend
	go func() {
		peer1.Socket <- msgRed
	}()
	go func() {
		for _, peer := range peers2 {
			peer.Socket <- msgBlue
		}
	}()
}

func (g *GodMode) chooseThePoorestDoubleSpendTargets() (*network.Peer, *network.Peer) {
	// the poorest node that is not an adversary
	peer1 := g.net.Peer(g.godNetworkIndex - 1)
	var peer2 *network.Peer

	// look for the first node that is not a neighbor of peer1 starting from the least weight
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
	for i := 1; i < g.godNetworkIndex; i++ {
		peer := g.net.Peer(i)
		weight := g.net.WeightDistribution.Weight(peer.ID)
		accumulatedWeight += weight
		peers2 = append(peers2, peer)
		if accumulatedWeight > peer1Weight {
			break
		}
	}
	return peer1, peers2
}

func (g *GodMode) PrepareMessage(color multiverse.Color) *multiverse.Message {
	node := g.peer.Node.(multiverse.NodeInterface)
	msg := node.Tangle().MessageFactory.CreateMessage(color)
	return msg
}

func (g *GodMode) issueMessageOnOpinionChange(previousOpinion, newOpinion multiverse.Color, weight int64) {
	g.peer.ReceiveNetworkMessage(newOpinion)
}

func (g *GodMode) gossipOwnProcessedMessage(messageID multiverse.MessageID) {
	node := g.peer.Node.(multiverse.NodeInterface)
	msg := node.Tangle().Storage.Message(messageID)
	// gossip only your own messages
	if g.peer.ID == msg.Issuer {
		colorsSupport := node.Tangle().OpinionManager.ApprovalWeights()
		// do nothing unless there are two colors
		if len(colorsSupport) < 2 {
			return
		}
		log.Info("OWN MESSAGE GOSSIPED")
		// iterate over all honest nodes
		for _, peer := range g.net.Peers[:g.godNetworkIndex] {
			time.AfterFunc(g.adversaryDelay, func() {
				peer.Socket <- msg
			})
		}
		g.peer.GossipNetworkMessage(msg)
	}
}

func (g *GodMode) updateSeenMessageIDs(message *multiverse.Message) (updated bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.seenMessageIDs[message.ID]; !ok {
		g.seenMessageIDs[message.ID] = message
		return true
	}
	return false
}

func (g *GodMode) onMessageStored(message *multiverse.Message) {
	firstTimeSeen := g.updateSeenMessageIDs(message)
	if !firstTimeSeen {
		return
	}
	g.peer.ReceiveNetworkMessage(message)
}

// endregion //////////////////////////////////////////////////////////////////////////////////////////////////////
