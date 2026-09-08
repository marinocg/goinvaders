package input

import "context"

// QueueSource is a deterministic in-memory Source used by loop tests and wiring.
type QueueSource struct {
	queue []Event
}

// NewQueueSource constructs a QueueSource with a defensive copy of queued events.
func NewQueueSource(events []Event) *QueueSource {
	queue := make([]Event, len(events))
	copy(queue, events)

	return &QueueSource{queue: queue}
}

// Drain returns all currently queued events in insertion order and empties the queue.
func (s *QueueSource) Drain(ctx context.Context) ([]Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if len(s.queue) == 0 {
		return nil, nil
	}

	events := make([]Event, len(s.queue))
	for i, event := range s.queue {
		events[i] = event.Normalize()
	}

	s.queue = s.queue[:0]

	return events, nil
}
