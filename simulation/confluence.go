package simulation

import (
	"github.com/iotaledger/hive.go/types"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/multiverse"
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

	monitorTick       int
	trackedMessageIDs map[multiverse.MessageID]types.Empty
	tipsMetadata      map[multiverse.MessageID]*TipMetadata
}

func NewMonitoringNode() interface{} {
	node := multiverse.NewNode().(*multiverse.Node)
	metricNode := &MonitoringNode{
		Node:         node,
		monitorTick:  config.ConfluenceMonitorTick,
		tipsMetadata: make(map[multiverse.MessageID]*TipMetadata),
	}
	return metricNode
}

func (n *MonitoringNode) isMessageTracked(msgID multiverse.MessageID) bool {
	if _, ok := n.trackedMessageIDs[msgID]; ok {
		return true
	}
	return false
}

func (n *MonitoringNode) updateTipsPastCone(msgID, trackedMessageID multiverse.MessageID) {
	metaData, ok := n.tipsMetadata[msgID]
	if ok {
		metaData.trackedInPastCone[trackedMessageID] = types.Void
	}
}

func (n *MonitoringNode) removeFromTipsPastCone(msgID, trackedMessageID multiverse.MessageID) {
	metaData, ok := n.tipsMetadata[msgID]
	if ok {
		delete(metaData.trackedInPastCone, trackedMessageID)
	}
}

func (n *MonitoringNode) inheritTipPastCone(newTip multiverse.MessageID, parents []multiverse.MessageID) {
	oldTipMeta, ok := n.tipsMetadata[parents[0]]
	if !ok {
		return
	}
	n.tipsMetadata[newTip] = oldTipMeta
	delete(n.tipsMetadata, parents[0])
	for i := 1; i < len(parents)-1; i++ {
		for trackedMsgID := range n.tipsMetadata[parents[i]].trackedInPastCone {
			n.tipsMetadata[newTip].trackedInPastCone[trackedMsgID] = types.Void
		}
		delete(n.tipsMetadata, parents[i])
	}
}
