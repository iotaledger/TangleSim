package multiverse

import (
	"github.com/iotaledger/hive.go/cerrors"
	"github.com/iotaledger/hive.go/events"
	"golang.org/x/xerrors"
)

// region Booker ///////////////////////////////////////////////////////////////////////////////////////////////////////

type Booker struct {
	Events *BookerEvents

	tangle *Tangle
}

func NewBooker(tangle *Tangle) (booker *Booker) {
	return &Booker{
		Events: &BookerEvents{
			MessageBooked:  events.NewEvent(messageIDEventCaller),
			MessageInvalid: events.NewEvent(messageIDEventCaller),
		},

		tangle: tangle,
	}
}

func (b *Booker) Setup() {
	b.tangle.Solidifier.Events.MessageSolid.Attach(events.NewClosure(b.Book))
}

func (b *Booker) Book(messageID MessageID) {
	message := b.tangle.Storage.Message(messageID)
	messageMetadata := b.tangle.Storage.MessageMetadata(messageID)

	inheritedColor, err := b.inheritColor(message)
	if err != nil {
		b.Events.MessageInvalid.Trigger(messageID)
		return
	}

	messageMetadata.SetInheritedColor(inheritedColor)

	b.Events.MessageBooked.Trigger(messageID)
}

// The booked message will inherit the color from its parent
func (b *Booker) inheritColor(message *Message) (inheritedColor Color, err error) {
	inheritedColor = message.Payload
	for _, colorToInherit := range append(make([]Color, 0), b.colorsOfStrongParents(message)...) {
		if colorToInherit == UndefinedColor {
			continue
		}

		if inheritedColor != UndefinedColor && inheritedColor != colorToInherit {
			err = xerrors.Errorf("message with %s tried to combine conflicting perceptions of the ledger state: %w", message.ID, cerrors.ErrFatal)
			return
		}

		inheritedColor = colorToInherit
	}

	return
}

func (b *Booker) colorsOfStrongParents(message *Message) (colorsOfStrongParents []Color) {
	for strongParent := range message.StrongParents {
		if strongParent == Genesis {
			continue
		}

		colorsOfStrongParents = append(colorsOfStrongParents, b.tangle.Storage.MessageMetadata(strongParent).InheritedColor())
	}

	return
}

func (b *Booker) colorsOfWeakParents(message *Message) (colorsOfStrongParents []Color) {
	for weakParent := range message.WeakParents {
		if weakParent == Genesis {
			continue
		}

		colorsOfStrongParents = append(colorsOfStrongParents, b.tangle.Storage.Message(weakParent).Payload)
	}

	return
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region BookerEvents /////////////////////////////////////////////////////////////////////////////////////////////////

type BookerEvents struct {
	MessageInvalid *events.Event
	MessageBooked  *events.Event
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
