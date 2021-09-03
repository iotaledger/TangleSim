package multiverse

import (
	"strings"
	"time"

	"github.com/iotaledger/hive.go/datastructure/randommap"
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/config"
)

var (
	OptimalStrongTipsCount = int(float64(config.TipsCount) * (1 - config.WeakTipsRatio))
	OptimalWeakTipsCount   = int(float64(config.TipsCount) * config.WeakTipsRatio)
)

// region TipManager ///////////////////////////////////////////////////////////////////////////////////////////////////

type TipManager struct {
	tangle  *Tangle
	tsa     TipSelector
	tipSets map[Color]*TipSet
}

func NewTipManager(tangle *Tangle, tsaString string) (tipManager *TipManager) {
	tsaString = strings.ToUpper(tsaString) // make sure string is upper case
	var tsa TipSelector
	switch tsaString {
	case "URTS":
		tsa = URTS{}
	case "RURTS":
		tsa = RURTS{}
	default:
		tsa = URTS{}
	}
	return &TipManager{
		tangle:  tangle,
		tsa:     tsa,
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
	// maintain the tip pool in reality where nodes opinion is not color
	for color, tipSet := range t.TipSets(UndefinedColor) {
		// add as a weak tip to colored tip sets except message color
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
	// The tips is selected form the tipSet of the current ownOpinion
	tipSet := t.TipSet(t.tangle.OpinionManager.Opinion())

	strongTips = tipSet.StrongTips(config.TipsCount, t.tsa)
	weakTips = tipSet.WeakTips(config.TipsCount-1, t.tsa)

	// if no weak tips is available, then return whatever number of strong tips
	if len(weakTips) == 0 {
		return
	}

	if strongTipsCount := len(strongTips); strongTipsCount < OptimalStrongTipsCount {
		fillUpCount := config.TipsCount - strongTipsCount

		if fillUpCount >= len(weakTips) {
			return
		}

		weakTips.Trim(fillUpCount)
		return
	}

	if weakTipsCount := len(weakTips); weakTipsCount < OptimalWeakTipsCount {
		fillUpCount := config.TipsCount - weakTipsCount

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
	for strongParent := range message.StrongParents {
		t.strongTips.Delete(strongParent)
	}

	for weakParent := range message.WeakParents {
		t.weakTips.Delete(weakParent)
	}
}

func (t *TipSet) AddWeakTip(message *Message) {
	t.weakTips.Set(message.ID, message)
}

func (t *TipSet) StrongTips(maxAmount int, tsa TipSelector) (strongTips MessageIDs) {
	if t.strongTips.Size() == 0 {
		strongTips = NewMessageIDs(Genesis)
		return
	}

	strongTips = make(MessageIDs)
	for _, strongTip := range tsa.TipSelect(t.strongTips, maxAmount) {
		strongTips.Add(strongTip.(*Message).ID)
	}

	return
}

func (t *TipSet) WeakTips(maxAmount int, tsa TipSelector) (weakTips MessageIDs) {
	if t.weakTips.Size() == 0 {
		return
	}

	weakTips = make(MessageIDs)
	for _, weakTip := range tsa.TipSelect(t.weakTips, maxAmount) {
		weakTips.Add(weakTip.(*Message).ID)
	}

	return
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region TipSelector //////////////////////////////////////////////////////////////////////////////////////////////////

// TipSelector defines the interface for a TSA
type TipSelector interface {
	TipSelect(tips *randommap.RandomMap, maxAmount int) []interface{}
}

// URTS implements the uniform random tip selection algorithm
type URTS struct {
	TipSelector
}

// RURTS implements the restricted uniform random tip selection algorithm, where txs are only valid tips up to some age D
type RURTS struct {
	TipSelector
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// TipSelect selects maxAmount tips
func (URTS) TipSelect(tips *randommap.RandomMap, maxAmount int) []interface{} {
	return tips.RandomUniqueEntries(maxAmount)

}

// TipSelect selects maxAmount tips
// RURTS: URTS with max parent age restriction
func (RURTS) TipSelect(tips *randommap.RandomMap, maxAmount int) []interface{} {

	var tipsNew []interface{}
	var tipsToReturn []interface{}
	amountLeft := maxAmount

	for {
		// Get amountLeft tips
		tipsNew = tips.RandomUniqueEntries(amountLeft)

		// If there are no tips, return the tipsToReturn
		if len(tipsNew) == 0 {
			break
		}

		// Get the current time
		currentTime := time.Now()
		for _, tip := range tipsNew {

			// If the time difference is greater than DeltaURTS, delete it from tips
			if currentTime.Sub(tip.(*Message).IssuanceTime).Seconds() > config.DeltaURTS {
				tips.Delete(tip)
			} else {
				// Append the valid tip to tipsToReturn and decrease the amountLeft
				tipsToReturn = append(tipsToReturn, tip)
				amountLeft--
			}
		}

		// If maxAmount tips are appended to tipsToReturn already, return the tipsToReturn
		if amountLeft == 0 {
			break
		}
	}

	return tipsToReturn

}
