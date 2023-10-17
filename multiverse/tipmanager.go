package multiverse

import (
	"encoding/csv"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/iotaledger/hive.go/datastructure/randommap"
	"github.com/iotaledger/hive.go/datastructure/walker"
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/multivers-simulation/config"
)

var (
	OptimalStrongParentsCount = int(float64(config.Params.ParentsCount) * (1 - config.Params.WeakTipsRatio))
	OptimalWeakParentsCount   = int(float64(config.Params.ParentsCount) * config.Params.WeakTipsRatio)
)

// region TipManager ///////////////////////////////////////////////////////////////////////////////////////////////////

type TipManager struct {
	Events *TipManagerEvents

	tangle              *Tangle
	tsa                 TipSelector
	tipSets             map[Color]*TipSet
	msgProcessedCounter map[Color]uint64

	confirmationWriter *csv.Writer
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

	// Initialize the counters
	msgProcessedCounter := make(map[Color]uint64)
	msgProcessedCounter[UndefinedColor] = 0
	msgProcessedCounter[Red] = 0
	msgProcessedCounter[Green] = 0

	return &TipManager{
		Events: &TipManagerEvents{
			MessageProcessed: events.NewEvent(messageProcessedHandler),
		},

		tangle:              tangle,
		tsa:                 tsa,
		tipSets:             make(map[Color]*TipSet),
		msgProcessedCounter: msgProcessedCounter,
		confirmationWriter:  NewConfirmationWriter(),
	}
}

func (t *TipManager) Setup() {
	//t.tangle.OpinionManager.Events().OpinionFormed.Attach(events.NewClosure(t.AnalyzeMessage))
	// Try "analysing" on scheduling instead of on opinion formation.
	t.tangle.Scheduler.Events().MessageScheduled.Attach(events.NewClosure(t.AnalyzeMessage))
}

func (t *TipManager) AnalyzeMessage(messageID MessageID) {
	message := t.tangle.Storage.Message(messageID)
	messageMetadata := t.tangle.Storage.MessageMetadata(messageID)
	inheritedColor := messageMetadata.InheritedColor()
	tipSet := t.TipSet(inheritedColor)
	// Calculate the current tip pool size before calling AddStrongTip
	currentTipPoolSize := tipSet.strongTips.Size()

	if time.Since(message.IssuanceTime).Seconds() < config.Params.DeltaURTS || config.Params.TSA != "RURTS" {
		addedAsStrongTip := make(map[Color]bool)
		for color, tipSet := range t.TipSets(inheritedColor) {
			addedAsStrongTip[color] = true
			tipSet.AddStrongTip(message)

			if message.Validation {
				tipSet.AddValidatorValidationTip(message)
			} else {
				tipSet.AddValidatorStrongTip(message)
			}

			t.msgProcessedCounter[color] += 1
		}
	}

	// Color, tips pool count, processed messages issued messages
	t.Events.MessageProcessed.Trigger(inheritedColor, currentTipPoolSize,
		t.msgProcessedCounter[inheritedColor], messageIDCounter)

	// Remove the weak tip codes
	// for color, tipSet := range t.TipSets(inheritedColor) {
	// 	if !addedAsStrongTip[color] {
	// 		tipSet.AddWeakTip(message)
	// 	}
	// }
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

func (t *TipManager) Tips(validation bool) (strongTips MessageIDs, weakTips MessageIDs) {
	// The tips is selected form the tipSet of the current ownOpinion
	tipSet := t.TipSet(t.tangle.OpinionManager.Opinion())

	// peerID := t.tangle.Peer.ID
	// if peerID == 99 {
	// 	t.WalkForOldestUnconfirmed(tipSet)
	// }

	if !validation {
		strongTips = tipSet.StrongTips(config.Params.ParentsCount, t.tsa)
	} else {
		// strongTips = tipSet.ValidationTips(config.Params.ParentCountVB, config.Params.ParentCountNVB, t.tsa)
		strongTips = tipSet.StrongTips(config.Params.ParentCountVB + config.Params.ParentCountNVB, t.tsa)
	}
	// In the paper we consider all strong tips
	// weakTips = tipSet.WeakTips(config.Params.ParentsCount-1, t.tsa)

	// Remove the weakTips-related codes
	// if len(weakTips) == 0 {
	// 	return
	// }

	// if strongParentsCount := len(strongTips); strongParentsCount < OptimalStrongParentsCount {
	// 	fillUpCount := config.Params.ParentsCount - strongParentsCount

	// 	if fillUpCount >= len(weakTips) {
	// 		return
	// 	}

	// 	weakTips.Trim(fillUpCount)
	// 	return
	// }

	// if weakParentsCount := len(weakTips); weakParentsCount < OptimalWeakParentsCount {
	// 	fillUpCount := config.Params.ParentsCount - weakParentsCount

	// 	if fillUpCount >= len(strongTips) {
	// 		return
	// 	}

	// 	strongTips.Trim(fillUpCount)
	// 	return
	// }

	// strongTips.Trim(OptimalStrongParentsCount)
	// weakTips.Trim(OptimalWeakParentsCount)

	return
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region TipSet ///////////////////////////////////////////////////////////////////////////////////////////////////////

type TipSet struct {
	// for non-validation block
	strongTips *randommap.RandomMap
	weakTips   *randommap.RandomMap

	// for validation block
	validatorStrongTips     *randommap.RandomMap
	validatorValidationTips *randommap.RandomMap
	validatorWeakTips       *randommap.RandomMap
}

func NewTipSet(tipsToInherit *TipSet) (tipSet *TipSet) {
	tipSet = &TipSet{
		strongTips:              randommap.New(),
		weakTips:                randommap.New(),
		validatorStrongTips:     randommap.New(),
		validatorWeakTips:       randommap.New(),
		validatorValidationTips: randommap.New(),
	}

	if tipsToInherit != nil {
		tipsToInherit.strongTips.ForEach(func(key interface{}, value interface{}) {
			tipSet.strongTips.Set(key, value)

			msg := value.(*Message)
			if msg.Validation {
				tipSet.validatorValidationTips.Set(key, value)
			} else {
				tipSet.validatorStrongTips.Set(key, value)
			}
		})
		tipsToInherit.weakTips.ForEach(func(key interface{}, value interface{}) {
			tipSet.weakTips.Set(key, value)
			tipSet.validatorWeakTips.Set(key, value)
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

func (t *TipSet) AddValidatorStrongTip(message *Message) {
	t.validatorStrongTips.Set(message.ID, message)
	for strongParent := range message.StrongParents {
		t.validatorStrongTips.Delete(strongParent)
	}

	for weakParent := range message.WeakParents {
		t.validatorWeakTips.Delete(weakParent)
	}
}

func (t *TipSet) AddValidatorValidationTip(message *Message) {
	if !message.Validation {
		fmt.Println("AddValidatorValidationTip: message is not validation block", message.ID)
		return
	}

	t.validatorValidationTips.Set(message.ID, message)
	for strongParent := range message.StrongParents {
		t.validatorValidationTips.Delete(strongParent)
	}
}

func (t *TipSet) AddValidatorWeakTip(message *Message) {
	t.validatorWeakTips.Set(message.ID, message)
}

func (t *TipSet) Size() int {
	return t.strongTips.Size()
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

func (t *TipSet) ValidationTips(maxVBAmount, maxNVBAmount int, tsa TipSelector) (validationTips MessageIDs) {
	if t.validatorValidationTips.Size() == 0 && t.validatorStrongTips.Size() == 0 {
		validationTips = NewMessageIDs(Genesis)
		return
	}

	validationTips = make(MessageIDs)
	for _, strongTip := range tsa.TipSelect(t.validatorValidationTips, maxVBAmount) {
		validationTips.Add(strongTip.(*Message).ID)
	}

	for _, strongTip := range tsa.TipSelect(t.validatorStrongTips, maxNVBAmount) {
		validationTips.Add(strongTip.(*Message).ID)
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

// region TipSelect Algorithm /////////////////////////////////////////////////////////////////////////////////////////
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
			if currentTime.Sub(tip.(*Message).IssuanceTime).Seconds() > config.Params.DeltaURTS {
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

func (t *TipManager) WalkForOldestUnconfirmed(tipSet *TipSet) (oldestMessage MessageID) {
	strongKeys := tipSet.strongTips.Keys()

	for _, tip := range strongKeys {
		messageID := tip.(MessageID)
		currentTangleTime := time.Now()
		tipTangleTime := t.tangle.Storage.Message(messageID).IssuanceTime
		hasConfirmedParents := false

		for parent := range t.tangle.Storage.Message(messageID).StrongParents {
			if parent == Genesis {
				continue
			}

			oldestUnconfirmedTime := currentTangleTime
			oldestConfirmationTime := currentTangleTime

			// Walk through the past cone to find the oldest unconfirmed blocks
			t.tangle.Utils.WalkMessagesAndMetadata(func(message *Message, messageMetadata *MessageMetadata, walker *walker.Walker) {
				issuanceTime := message.IssuanceTime
				// Reaches the confirmed blocks, stop traversing
				if messageMetadata.Confirmed() {
					hasConfirmedParents = true
					// Use the issuance time of the youngest confirmed block
					if issuanceTime.Before(oldestConfirmationTime) {
						oldestConfirmationTime = issuanceTime
					}
				} else {
					if issuanceTime.Before(oldestUnconfirmedTime) {
						oldestUnconfirmedTime = issuanceTime
						oldestMessage = message.ID
					}
					// Only continue the BFS when the current block is unconfirmed
					for strongChildID := range message.StrongParents {
						walker.Push(strongChildID)
					}
				}

			}, NewMessageIDs(parent), false)

			t.dumpAges(hasConfirmedParents, currentTangleTime, oldestUnconfirmedTime, oldestConfirmationTime, tipTangleTime)
			// if timeSinceConfirmation > tsc_condition {
			// 	oldTips[tip.(*Message).ID] = void{}
			// 	fmt.Printf("Prune %d\n", tip.(*Message).ID)
			// }
		}
	}
	return 0
}

func (t *TipManager) dumpAges(hasConfirmedParents bool, currentTangleTime time.Time, oldestUnconfirmedTime time.Time, oldestConfirmationTime time.Time, tipTangleTime time.Time) {
	// Distance between (Now, Issuance Time of the oldest UNCONFIRMED block that has confirmed parents)
	t.confirmationWriter.Write([]string{"UnconfirmationAge", fmt.Sprintf("%f", currentTangleTime.Sub(oldestUnconfirmedTime).Seconds())})

	// Distance between (Issuance Time of the tip, Issuance Time of the oldest UNCONFIRMED block that has confirmed parents)
	t.confirmationWriter.Write([]string{"UnconfirmationAgeSinceTip", fmt.Sprintf("%f", tipTangleTime.Sub(oldestUnconfirmedTime).Seconds())})

	if hasConfirmedParents {
		// Distance between (Issuance Time of the tip, Issuance Time of the oldest CONFIRMED block that has no confirmed children)
		t.confirmationWriter.Write([]string{"ConfirmationAgeSinceTip", fmt.Sprintf("%f", tipTangleTime.Sub(oldestConfirmationTime).Seconds())})

		// Distance between (Now, Issuance Time of the oldest CONFIRMED block that has no confirmed children)
		t.confirmationWriter.Write([]string{"ConfirmationAge", fmt.Sprintf("%f", currentTangleTime.Sub(oldestConfirmationTime).Seconds())})

	}
	t.confirmationWriter.Flush()
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region TipManagerEvents /////////////////////////////////////////////////////////////////////////////////////////

type TipManagerEvents struct {
	MessageProcessed *events.Event
}

func messageProcessedHandler(handler interface{}, params ...interface{}) {
	handler.(func(Color, int, uint64, int64))(params[0].(Color), params[1].(int), params[2].(uint64), params[3].(int64))
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

func NewConfirmationWriter() *csv.Writer {
	// define header with time of dump and each node ID
	gmHeader := []string{
		"Title",
		"Time (s)",
	}

	path := path.Join(config.Params.GeneralOutputDir, "confirmationThreshold.csv")
	if err := os.MkdirAll(filepath.Dir(path), 0770); err != nil {
		panic(err)
	}
	file, err := os.Create(path)
	if err != nil {
		panic(err)
	}

	confirmationWriter := csv.NewWriter(file)
	if err := confirmationWriter.Write(gmHeader); err != nil {
		panic(err)
	}

	return confirmationWriter
}
