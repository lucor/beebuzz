package notification

import (
	"context"
	"log/slog"
)

// PushStubEvent is a captured push payload for push-stub mode.
// It carries the exact bytes that would have been sent to the push provider,
// so a test driver can deliver them directly into the service worker via CDP.
type PushStubEvent struct {
	Endpoint string `json:"endpoint"`
	DeviceID string `json:"device_id"`
	// Data is the JS-visible payload (post-transport-decryption).
	// For BeeBuzz this is the JSON envelope or notification payload.
	Data string `json:"data"`
}

// pushStubBufferSize bounds the in-memory queue. The stub flow is single-consumer
// and short-lived; older events are dropped on overflow.
const pushStubBufferSize = 16

// PushStubBroker is a tiny in-memory queue used by push-stub mode to capture
// outbound push payloads instead of dispatching them to a real push provider.
//
// Never enable this in production.
type PushStubBroker struct {
	ch  chan PushStubEvent
	log *slog.Logger
}

// NewPushStubBroker returns a broker with a small bounded buffer. The logger
// may be nil; in that case overflow drops are silent.
func NewPushStubBroker(log *slog.Logger) *PushStubBroker {
	return &PushStubBroker{
		ch:  make(chan PushStubEvent, pushStubBufferSize),
		log: log,
	}
}

// Publish enqueues an event. If the buffer is full, the event is dropped and
// a warning is logged so the operator can spot mismatches in the demo flow.
func (b *PushStubBroker) Publish(ev PushStubEvent) {
	select {
	case b.ch <- ev:
	default:
		if b.log != nil {
			b.log.Warn("push stub broker overflow, dropping event",
				"device_id", ev.DeviceID,
				"buffer_size", pushStubBufferSize,
			)
		}
	}
}

// Next blocks until an event is available or the context is cancelled.
func (b *PushStubBroker) Next(ctx context.Context) (PushStubEvent, error) {
	select {
	case ev := <-b.ch:
		return ev, nil
	case <-ctx.Done():
		return PushStubEvent{}, ctx.Err()
	}
}
