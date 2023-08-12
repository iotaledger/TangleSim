package multiverse

import (
	"time"

	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/network"
)

type Tangle struct {
	Peer                  *network.Peer
	WeightDistribution    *network.ConsensusWeightDistribution
	BandwidthDistribution *network.BandwidthDistribution
	GenesisTime           time.Time
	Storage               *Storage
	Solidifier            *Solidifier
	ApprovalManager       *ApprovalManager
	Requester             *Requester
	Booker                *Booker
	OpinionManager        OpinionManagerInterface
	TipManager            *TipManager
	MessageFactory        *MessageFactory
	Utils                 *Utils
	Scheduler             Scheduler
}

func NewTangle() (tangle *Tangle) {
	tangle = &Tangle{}

	tangle.Storage = NewStorage()
	tangle.Solidifier = NewSolidifier(tangle)
	tangle.Requester = NewRequester(tangle)
	tangle.Booker = NewBooker(tangle)
	tangle.OpinionManager = NewOpinionManager(tangle)
	tangle.TipManager = NewTipManager(tangle, config.Params.TSA)
	tangle.MessageFactory = NewMessageFactory(tangle, uint64(config.Params.NodesCount))
	tangle.ApprovalManager = NewApprovalManager(tangle)
	tangle.Utils = NewUtils(tangle)
	tangle.Scheduler = NewScheduler(tangle)
	return
}

func (t *Tangle) Setup(peer *network.Peer, weightDistribution *network.ConsensusWeightDistribution, genesisTime time.Time) {
	t.Peer = peer
	t.WeightDistribution = weightDistribution

	t.Storage.Setup(genesisTime)
	t.Solidifier.Setup()
	t.Requester.Setup()
	t.Booker.Setup()
	t.OpinionManager.Setup()
	t.TipManager.Setup()
	t.ApprovalManager.Setup()
	t.Scheduler.Setup()
}

func (t *Tangle) ProcessMessage(message *Message) {
	if messageMetadata, stored := t.Storage.Store(message); stored {
		t.Storage.Events.MessageStored.Trigger(message.ID, message, messageMetadata)
	}
}
