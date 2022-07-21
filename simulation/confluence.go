package simulation

import (
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/hive.go/types"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/multiverse"
	"github.com/iotaledger/multivers-simulation/network"
	"time"
)

type TipMetadata struct {
	msgID             multiverse.MessageID
	trackedInPastCone map[multiverse.MessageID]types.Empty
}

func NewTipMetadata(msgID multiverse.MessageID) *TipMetadata {
	return &TipMetadata{
		msgID:             msgID,
		trackedInPastCone: make(map[multiverse.MessageID]types.Empty),
	}
}

type MonitoringNode struct {
	*multiverse.Node
}

func NewMonitoringNode() interface{} {
	node := multiverse.NewNode().(*multiverse.Node)
	metricNode := &MonitoringNode{
		Node: node,
	}
	metricNode.setupTipManager()
	return metricNode
}

func (n *MonitoringNode) setupTipManager() {
	tm := n.Tangle().TipManager
	n.Tangle().TipManager = NewMonitoringTipManager(tm)
}

type MonitoringTipManager struct {
	*multiverse.TipManager

	events            *MonitoringTipManagerEvents
	monitorTick       int
	trackedMessageIDs map[multiverse.MessageID]types.Empty
	tipsMetadata      map[multiverse.MessageID]*TipMetadata
	tipSelectedAgo    int
}

func NewMonitoringTipManager(tm multiverse.TipManagerInterface) *MonitoringTipManager {
	return &MonitoringTipManager{
		TipManager: tm.(*multiverse.TipManager),
		events: &MonitoringTipManagerEvents{
			SeenByAllTips: events.NewEvent(seenByAllTipsHandler),
		},
		monitorTick:       config.ConfluenceMonitorTick,
		trackedMessageIDs: make(map[multiverse.MessageID]types.Empty),
		tipsMetadata:      make(map[multiverse.MessageID]*TipMetadata),
		tipSelectedAgo:    0,
	}
}

func (t *MonitoringTipManager) MonitoringEvents() *MonitoringTipManagerEvents {
	return t.events
}

func (t *MonitoringTipManager) AnalyzeMessage(msgID multiverse.MessageID) {
	t.selectMessageToMonitor(msgID)
	t.checkTracked()

	t.TipManager.AnalyzeMessage(msgID)
}

func (t *MonitoringTipManager) selectMessageToMonitor(msgID multiverse.MessageID) {
	if t.tipSelectedAgo == t.monitorTick {
		t.tipSelectedAgo = 0
		t.trackedMessageIDs[msgID] = types.Void

	} else {
		t.tipSelectedAgo++
	}
	msg := t.Tangle().Storage.Message(msgID)
	parents := parentListFromMap(msg)
	t.inheritTipPastCone(msgID, parents)
}

func (t *MonitoringTipManager) isMessageTracked(msgID multiverse.MessageID) bool {
	if _, ok := t.trackedMessageIDs[msgID]; ok {
		return true
	}
	return false
}

func (t *MonitoringTipManager) updateTipsPastCone(msgID, trackedMessageID multiverse.MessageID) {
	metaData, ok := t.tipsMetadata[msgID]
	if ok {
		metaData.trackedInPastCone[trackedMessageID] = types.Void
	}
}

func (t *MonitoringTipManager) removeFromTipsPastCone(msgID, trackedMessageID multiverse.MessageID) {
	metaData, ok := t.tipsMetadata[msgID]
	if ok {
		delete(metaData.trackedInPastCone, trackedMessageID)
	}
}

// inheritTipPastCone each tip inherits all tracked messages from their parents.
// If the parent is tracked it is also included to the tracked past cone.
func (t *MonitoringTipManager) inheritTipPastCone(newTip multiverse.MessageID, parents []multiverse.MessageID) {
	// Create metadata for each tip
	t.tipsMetadata[newTip] = NewTipMetadata(newTip)
	// delete(t.tipsMetadata, parents[0])
	for _, parent := range parents {
		if t.isMessageTracked(parent) {
			t.tipsMetadata[newTip].trackedInPastCone[parent] = types.Void
		}
		// for genesis
		if _, ok := t.tipsMetadata[parent]; !ok {
			t.tipsMetadata[parent] = NewTipMetadata(newTip)
		}
		for trackedMsgID := range t.tipsMetadata[parent].trackedInPastCone {
			if t.isMessageTracked(trackedMsgID) {

				t.tipsMetadata[newTip].trackedInPastCone[trackedMsgID] = types.Void
			}
		}
		// delete(t.tipsMetadata, parents[i])
	}
}

func (t *MonitoringTipManager) checkTracked() {
	for tracked := range t.trackedMessageIDs {
		strong, _ := t.Tips()

		seenByAllTips := true
		for tip := range strong {
			if _, ok := t.tipsMetadata[tip].trackedInPastCone[tracked]; !ok {
				seenByAllTips = false
				break
			}
		}
		if seenByAllTips {
			delete(t.trackedMessageIDs, tracked)

			confluenceTime := time.Now().Sub(t.TipManager.Tangle().Storage.Message(tracked).IssuanceTime)
			// delete from metadata
			t.events.SeenByAllTips.Trigger(confluenceTime, tracked, t.Tangle().Peer.ID)
		}
	}
}

func parentListFromMap(msg *multiverse.Message) []multiverse.MessageID {
	parents := make([]multiverse.MessageID, 0)
	for parent := range msg.StrongParents {
		parents = append(parents, parent)
	}
	return parents
}

type MonitoringTipManagerEvents struct {
	SeenByAllTips    *events.Event
	MessageProcessed *events.Event
}

func seenByAllTipsHandler(handler interface{}, params ...interface{}) {
	handler.(func(confluenceTime time.Duration, messageID multiverse.MessageID, nodeID network.PeerID))(params[0].(time.Duration), params[1].(multiverse.MessageID), params[2].(network.PeerID))
}

func (t *MonitoringTipManager) Setup() {
	t.Tangle().OpinionManager.Events().OpinionFormed.Detach(events.NewClosure(t.AnalyzeMessage))
	t.Tangle().OpinionManager.Events().OpinionFormed.Attach(events.NewClosure(t.AnalyzeMessage))
}
