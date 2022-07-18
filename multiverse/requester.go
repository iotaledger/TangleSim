package multiverse

import (
	"sync"
	"time"

	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/hive.go/timedexecutor"
)

const retryInterval = 5 * time.Second

// region Requester ////////////////////////////////////////////////////////////////////////////////////////////////////

type Requester struct {
	Events *RequesterEvents

	tangle         *Tangle
	timedExecutor  *timedexecutor.TimedExecutor
	queuedElements map[MessageID]*timedexecutor.ScheduledTask
	mutex          sync.Mutex
}

func NewRequester(tangle *Tangle) (requester *Requester) {
	requester = &Requester{
		Events: &RequesterEvents{
			Request: events.NewEvent(messageIDEventCaller),
		},

		tangle:         tangle,
		timedExecutor:  timedexecutor.New(1),
		queuedElements: make(map[MessageID]*timedexecutor.ScheduledTask),
	}

	return
}

func (r *Requester) Setup() {
	r.tangle.Solidifier.Events.MessageMissing.Attach(events.NewClosure(r.StartRequest))
	r.tangle.Storage.Events.MessageStored.Attach(events.NewClosure(r.StopRequest))
}

func (r *Requester) StartRequest(messageID MessageID) {
	// Comment out this funcion to turn off solidifier

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, requestExists := r.queuedElements[messageID]; requestExists {
		return
	}

	r.triggerRequestAndScheduleRetry(messageID)
}

func (r *Requester) StopRequest(messageID MessageID) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	request, requestExists := r.queuedElements[messageID]
	if !requestExists {
		return
	}

	request.Cancel()
	delete(r.queuedElements, messageID)
}

func (r *Requester) triggerRequestAndScheduleRetry(messageID MessageID) {
	r.Events.Request.Trigger(messageID)

	r.queuedElements[messageID] = r.timedExecutor.ExecuteAfter(func() {
		r.retry(messageID)
	}, retryInterval)
}

func (r *Requester) retry(messageID MessageID) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, requestExists := r.queuedElements[messageID]; !requestExists {
		return
	}

	r.triggerRequestAndScheduleRetry(messageID)
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region RequesterEvents //////////////////////////////////////////////////////////////////////////////////////////////

type RequesterEvents struct {
	Request *events.Event
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
