package events

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

var ErrQueueFull = errors.New("dispatcher queue full")

type AsyncDispatcher struct {
	queue   chan DriverOfferEvent
	workers int
	sender  OfferSender
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewAsyncDispatcher(
	workers int,
	queueSize int,
	sender OfferSender,
) *AsyncDispatcher {

	ctx, cancel := context.WithCancel(context.Background())

	return &AsyncDispatcher{
		queue:   make(chan DriverOfferEvent, queueSize),
		workers: workers,
		sender:  sender,
		ctx:     ctx,
		cancel:  cancel,
	}
}

/*
If queue is full:

Match() slows down
system stays alive

ctx	graceful shutdown
cancel	stop signal
*/

func (d *AsyncDispatcher) EnqueueDriverOffer(driverID, riderID string) error {

	event := DriverOfferEvent{
		DriverID: driverID,
		RiderID:  riderID,
	}

	select {
	case d.queue <- event: //retry scheduled
		return nil
	default:
		//queue full -> backpressure signal
		slog.Warn("retry dropped due to full queue",
			"driver", event.DriverID,
			"rider", event.RiderID,
		)
		return ErrQueueFull
	}
}

func (d *AsyncDispatcher) Start() {
	for i := 0; i < d.workers; i++ {
		go d.worker(i)
	}
}

func (d *AsyncDispatcher) worker(id int) {
	slog.Info("dispatcher worker started", "id", id)

	for {
		select {
		case <-d.ctx.Done():
			return

		case event := <-d.queue:
			slog.Info("[DISPATCHER_WORKER]",
				"worker_id", id,
				"driver", event.DriverID,
				"rider", event.RiderID,
			)
			d.handle(event)
		}
	}

}

func (d *AsyncDispatcher) handle(event DriverOfferEvent) {
	const maxRetries = 3

	err := d.sender.SendDriverOffer(event.DriverID, event.RiderID)
	if err == nil {
		return
	}

	event.Attempts++
	if event.Attempts >= maxRetries {
		slog.Error("offer delivery failed permanently",
			"driver", event.DriverID,
			"rider", event.RiderID,
		)
		return
	}

	time.Sleep(50 * time.Millisecond)

	select {
	case d.queue <- event:
	default:
		slog.Warn("retry dropped due to backpressure",
			"driver", event.DriverID,
			"rider", event.RiderID,
		)
	}
}

func (d *AsyncDispatcher) Stop() {
	d.cancel()
	//close(d.queue)
	//lets workers exit via ctx.Done()
	//Only the sender OR the owner closes a channel — never both  ,Here:  workers are both receivers and senders , therefore channel must NEVER be closed
}

/*
No goroutine leaks
No send-on-closed-channel panic
Backpressure respected
Retry bounded
Clean architecture
Replaceable transport
Testable
Predictable shutdown
*/
