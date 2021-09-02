package multiverse

import (
	"github.com/iotaledger/hive.go/datastructure/walker"
)

// region //////////////////////////////////////////////////////////////////////////////////////////////////////////////

type Utils struct {
	tangle *Tangle
}

func NewUtils(tangle *Tangle) *Utils {
	return &Utils{
		tangle: tangle,
	}
}

func (u *Utils) WalkMessageIDs(callback func(messageID MessageID, walker *walker.Walker), entryPoints MessageIDs, revisitElements ...bool) {
	if len(entryPoints) == 0 {
		panic("you need to provide at least one entry point")
	}

	messageWalker := walker.New(revisitElements...)
	for messageID := range entryPoints {
		messageWalker.Push(messageID)
	}

	for messageWalker.HasNext() {
		messageID := messageWalker.Next().(MessageID)
		if messageID != Genesis {
			callback(messageID, messageWalker)
		}
	}
}

func (u *Utils) WalkMessages(callback func(message *Message, walker *walker.Walker), entryPoints MessageIDs, revisitElements ...bool) {
	u.WalkMessageIDs(func(messageID MessageID, walker *walker.Walker) {
		callback(u.tangle.Storage.Message(messageID), walker)
	}, entryPoints, revisitElements...)
}

func (u *Utils) WalkMessagesAndMetadata(callback func(message *Message, messageMetadata *MessageMetadata, walker *walker.Walker), entryPoints MessageIDs, revisitElements ...bool) {
	u.WalkMessageIDs(func(messageID MessageID, walker *walker.Walker) {
		callback(u.tangle.Storage.Message(messageID), u.tangle.Storage.MessageMetadata(messageID), walker)
	}, entryPoints, revisitElements...)
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
