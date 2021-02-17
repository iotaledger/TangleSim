package multiverse

import (
	"github.com/iotaledger/hive.go/datastructure/randommap"
	"github.com/iotaledger/hive.go/events"
)

const (
	TipsCount              = 4
	WeakTipsRatio          = 0.25
	OptimalStrongTipsCount = int(float64(TipsCount) * (1 - WeakTipsRatio))
	OptimalWeakTipsCount   = int(float64(TipsCount) * WeakTipsRatio)
)

// region TipManager ///////////////////////////////////////////////////////////////////////////////////////////////////

type TipManager struct {
	tangle  *Tangle
	tipSets map[Color]*TipSet
}

func NewTipManager(tangle *Tangle) (tipManager *TipManager) {
	return &TipManager{
		tangle:  tangle,
		tipSets: make(map[Color]*TipSet),
	}
}

func (t *TipManager) Setup() {
	t.tangle.OpinionManager.Events.OpinionFormed.Attach(events.NewClosure(t.AnalyzeMessage))
}

func (t *TipManager) AnalyzeMessage(messageID MessageID) {
	message := t.tangle.Storage.Message(messageID)
	messageMetadata := t.tangle.Storage.MessageMetadata(messageID)

	addedAsStrongTip := make(map[Color]bool)
	for color, tipSet := range t.TipSets(messageMetadata.InheritedColor()) {
		addedAsStrongTip[color] = true
		tipSet.AddStrongTip(message)
	}

	for color, tipSet := range t.TipSets(messageMetadata.InheritedColor()) {
		if !addedAsStrongTip[color] {
			tipSet.AddWeakTip(message)
		}
	}
}

func (t *TipManager) TipSets(color Color) map[Color]*TipSet {
	if _, exists := t.tipSets[color]; !exists {
		t.tipSets[color] = NewTipSet(t.tipSets[UndefinedColor])
	}

	if color == UndefinedColor {
		return t.tipSets
	}

	return map[Color]*TipSet{
		color: t.tipSets[color],
	}
}

func (t *TipManager) TipSet(color Color) (tipSet *TipSet) {
	tipSet, exists := t.tipSets[color]
	if !exists {
		tipSet = NewTipSet(t.tipSets[UndefinedColor])
		t.tipSets[color] = tipSet
	}

	return
}

func (t *TipManager) Tips() (strongTips MessageIDs, weakTips MessageIDs) {
	tipSet := t.TipSet(t.tangle.OpinionManager.Opinion())

	strongTips = tipSet.StrongTips(TipsCount)
	weakTips = tipSet.WeakTips(TipsCount - 1)

	if len(weakTips) == 0 {
		return
	}

	if strongTipsCount := len(strongTips); strongTipsCount < OptimalStrongTipsCount {
		fillUpCount := TipsCount - strongTipsCount

		if fillUpCount >= len(weakTips) {
			return
		}

		weakTips.Trim(fillUpCount)
		return
	}

	if weakTipsCount := len(weakTips); weakTipsCount < OptimalWeakTipsCount {
		fillUpCount := TipsCount - weakTipsCount

		if fillUpCount >= len(strongTips) {
			return
		}

		strongTips.Trim(fillUpCount)
		return
	}

	strongTips.Trim(OptimalStrongTipsCount)
	weakTips.Trim(OptimalWeakTipsCount)

	return
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region TipSet ///////////////////////////////////////////////////////////////////////////////////////////////////////

type TipSet struct {
	strongTips *randommap.RandomMap
	weakTips   *randommap.RandomMap
}

func NewTipSet(tipsToInherit *TipSet) (tipSet *TipSet) {
	tipSet = &TipSet{
		strongTips: randommap.New(),
		weakTips:   randommap.New(),
	}

	if tipsToInherit != nil {
		tipsToInherit.strongTips.ForEach(func(key interface{}, value interface{}) {
			tipSet.strongTips.Set(key, value)
		})
		tipsToInherit.weakTips.ForEach(func(key interface{}, value interface{}) {
			tipSet.weakTips.Set(key, value)
		})
	}

	return
}

func (t *TipSet) AddStrongTip(message *Message) {
	t.strongTips.Set(message.ID, message)

	for _, strongParent := range message.StrongParents {
		t.strongTips.Delete(strongParent)
	}

	for _, weakParent := range message.WeakParents {
		t.weakTips.Delete(weakParent)
	}
}

func (t *TipSet) AddWeakTip(message *Message) {
	t.weakTips.Set(message.ID, message)
}

func (t *TipSet) StrongTips(maxAmount int) (strongTips MessageIDs) {
	if t.strongTips.Size() == 0 {
		strongTips = NewMessageIDs(Genesis)
		return
	}

	strongTips = make(MessageIDs)
	for _, strongTip := range t.strongTips.RandomUniqueEntries(maxAmount) {
		strongTips.Add(strongTip.(*Message).ID)
	}

	return
}

func (t *TipSet) WeakTips(maxAmount int) (weakTips MessageIDs) {
	if t.weakTips.Size() == 0 {
		return
	}

	weakTips = make(MessageIDs)
	for _, weakTip := range t.weakTips.RandomUniqueEntries(maxAmount) {
		weakTips.Add(weakTip.(*Message).ID)
	}

	return
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
