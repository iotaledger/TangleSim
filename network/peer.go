package network

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/iotaledger/hive.go/crypto"
	"github.com/iotaledger/hive.go/timedexecutor"
)

// region Peer /////////////////////////////////////////////////////////////////////////////////////////////////////////

type Peer struct {
	ID               PeerID
	Neighbors        map[PeerID]*Connection
	Socket           chan interface{}
	Node             Node
	AdversarySpeedup float64

	startOnce      sync.Once
	shutdownOnce   sync.Once
	shutdownSignal chan struct{}
}

func NewPeer(node Node) (peer *Peer) {
	peer = &Peer{
		ID:        NewPeerID(),
		Neighbors: make(map[PeerID]*Connection),
		Socket:    make(chan interface{}, 1024),
		Node:      node,

		shutdownSignal: make(chan struct{}, 1),
	}

	return
}

func (p *Peer) SetupNode(consensusWeightDistribution *ConsensusWeightDistribution) {
	p.Node.Setup(p, consensusWeightDistribution)
}

func (p *Peer) Start() {
	p.startOnce.Do(func() {
		go p.run()
	})
}

func (p *Peer) Shutdown() {
	p.shutdownOnce.Do(func() {
		close(p.shutdownSignal)
	})
}

func (p *Peer) ReceiveNetworkMessage(message interface{}) {
	p.Socket <- message
}

func (p *Peer) GossipNetworkMessage(message interface{}) {
	for _, neighborConnection := range p.Neighbors {
		neighborConnection.Send(message)
	}
}

func (p *Peer) String() string {
	return fmt.Sprintf("Peer%d", p.ID)
}

func (p *Peer) run() {
	for {
		select {
		case <-p.shutdownSignal:
			return
		case networkMessage := <-p.Socket:
			p.Node.HandleNetworkMessage(networkMessage)
		}
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region PeerID ///////////////////////////////////////////////////////////////////////////////////////////////////////

type PeerID int64

var peerIDCounter int64

func NewPeerID() PeerID {
	return PeerID(atomic.AddInt64(&peerIDCounter, 1) - 1)
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region Connection ///////////////////////////////////////////////////////////////////////////////////////////////////

type Connection struct {
	socket        chan<- interface{}
	networkDelay  time.Duration
	packetLoss    float64
	timedExecutor *timedexecutor.TimedExecutor
	shutdownOnce  sync.Once
	configuration *Configuration
}

func NewConnection(socket chan<- interface{}, networkDelay time.Duration, packetLoss float64, configuration *Configuration) (connection *Connection) {
	connection = &Connection{
		socket:        socket,
		networkDelay:  networkDelay,
		packetLoss:    packetLoss,
		timedExecutor: timedexecutor.New(1),
		configuration: configuration,
	}

	return
}

func (c *Connection) NetworkDelay() time.Duration {
	return c.networkDelay
}

func (c *Connection) PacketLoss() float64 {
	return c.packetLoss
}

func (c *Connection) Send(message interface{}) {
	if crypto.Randomness.Float64() <= c.packetLoss {
		return
	}
	c.timedExecutor.ExecuteAfter(func() {
		c.socket <- message
	}, c.configuration.ExpRandomNetworkDelay())
}

func (c *Connection) SetDelay(delay time.Duration) {
	c.networkDelay = delay
}

func (c *Connection) Shutdown() {
	c.shutdownOnce.Do(func() {
		c.timedExecutor.Shutdown(timedexecutor.CancelPendingTasks)
	})
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
