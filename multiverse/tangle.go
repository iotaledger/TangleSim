package multiverse

import (
	"github.com/iotaledger/multivers-simulation/network"
)

type Tangle struct {
	Peer               *network.Peer
	WeightDistribution *network.ConsensusWeightDistribution
	Storage            *Storage
	Solidifier         *Solidifier
	Requester          *Requester
	Booker             *Booker
	OpinionManager     *OpinionManager
	TipManager         *TipManager
	MessageFactory     *MessageFactory
	Utils              *Utils
}

func NewTangle() (tangle *Tangle) {
	tangle = &Tangle{}

	tangle.Storage = NewStorage(tangle)
	tangle.Solidifier = NewSolidifier(tangle)
	tangle.Requester = NewRequester(tangle)
	tangle.Booker = NewBooker(tangle)
	tangle.OpinionManager = NewOpinionManager(tangle)
	tangle.TipManager = NewTipManager(tangle)
	tangle.MessageFactory = NewMessageFactory(tangle)
	tangle.Utils = NewUtils(tangle)

	return
}

func (t *Tangle) Setup(peer *network.Peer, weightDistribution *network.ConsensusWeightDistribution) {
	t.Peer = peer
	t.WeightDistribution = weightDistribution

	t.Solidifier.Setup()
	t.Requester.Setup()
	t.Booker.Setup()
	t.OpinionManager.Setup()
	t.TipManager.Setup()
}

func (t *Tangle) ProcessMessage(message *Message) {
	t.Storage.Store(message)
}
