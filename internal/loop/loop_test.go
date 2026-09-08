package loop

import (
	"context"
	"errors"
	"testing"

	"github.com/marinocg/goinvaders/internal/game"
	"github.com/marinocg/goinvaders/internal/input"
)

func TestRunProcessesFixedTicksDeterministically(t *testing.T) {
	t.Parallel()

	state, err := game.NewState(5)
	if err != nil {
		t.Fatalf("NewState() error = %v", err)
	}

	ticks := &scriptedTickSource{values: []int{2, 1}, err: context.Canceled}
	src := &scriptedInputSource{events: [][]input.Event{
		{{Command: input.CommandMoveRight}, {Command: input.CommandMoveLeft}, {Command: input.CommandFire}},
		{{Command: input.CommandMoveLeft}},
	}}

	got, err := Run(context.Background(), state, ticks, src)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	snap := got.Snapshot()
	if snap.Tick != 3 {
		t.Fatalf("tick = %d, want 3", snap.Tick)
	}
	if snap.Player.X != 0 {
		t.Fatalf("player x = %d, want 0", snap.Player.X)
	}
}

func TestRunStopsOnQuitAtTickBoundary(t *testing.T) {
	t.Parallel()

	state, err := game.NewState(7)
	if err != nil {
		t.Fatalf("NewState() error = %v", err)
	}

	ticks := &scriptedTickSource{values: []int{1}}
	src := &scriptedInputSource{events: [][]input.Event{{{Command: input.CommandQuit}}}}

	got, err := Run(context.Background(), state, ticks, src)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got.Snapshot().Tick != 0 {
		t.Fatalf("tick = %d, want 0", got.Snapshot().Tick)
	}
}

func TestRunConsumesBoundaryInputOnlyOnceDuringCatchup(t *testing.T) {
	t.Parallel()

	state, err := game.NewState(9)
	if err != nil {
		t.Fatalf("NewState() error = %v", err)
	}

	ticks := &scriptedTickSource{values: []int{3}, err: context.Canceled}
	src := &scriptedInputSource{events: [][]input.Event{{{Command: input.CommandMoveRight}}}}

	got, err := Run(context.Background(), state, ticks, src)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	snap := got.Snapshot()
	if snap.Tick != 3 {
		t.Fatalf("tick = %d, want 3", snap.Tick)
	}
	if snap.Player.X != 5 {
		t.Fatalf("player x = %d, want 5", snap.Player.X)
	}
}

func TestRunReturnsCleanlyOnCanceledContext(t *testing.T) {
	t.Parallel()

	state, err := game.NewState(3)
	if err != nil {
		t.Fatalf("NewState() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := Run(ctx, state, &scriptedTickSource{}, &scriptedInputSource{})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	if got.Snapshot() != state.Snapshot() {
		t.Fatalf("state changed on cancellation")
	}
}

func TestRunPropagatesSourceErrors(t *testing.T) {
	t.Parallel()

	state, err := game.NewState(4)
	if err != nil {
		t.Fatalf("NewState() error = %v", err)
	}

	want := errors.New("boom")

	_, err = Run(context.Background(), state, &scriptedTickSource{err: want}, &scriptedInputSource{})
	if !errors.Is(err, want) {
		t.Fatalf("Run() error = %v, want %v", err, want)
	}
}

func TestRunIgnoresNonPositiveTickBatches(t *testing.T) {
	state, err := game.NewState(4, 4)
	if err != nil {
		t.Fatal(err)
	}
	source := &scriptedInputSource{events: [][]input.Event{{{Command: input.CommandMoveRight}}}}
	ticks := &scriptedTickSource{values: []int{0, -1, 1}, err: context.Canceled}

	got, err := Run(context.Background(), state, ticks, source)
	if err != nil {
		t.Fatal(err)
	}
	if got.Snapshot().Tick != 1 || got.Snapshot().Player.X != state.Snapshot().Player.X+1 {
		t.Fatalf("state after ignored batches = %+v, want one input-bearing tick", got.Snapshot())
	}
	if source.idx != 1 {
		t.Fatalf("input drains = %d, want one after positive tick", source.idx)
	}
}

type scriptedTickSource struct {
	values []int
	err    error
	idx    int
}

func (s *scriptedTickSource) Next(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	if s.idx < len(s.values) {
		value := s.values[s.idx]
		s.idx++
		return value, nil
	}

	if s.err != nil {
		return 0, s.err
	}

	return 0, context.Canceled
}

type scriptedInputSource struct {
	events [][]input.Event
	err    error
	idx    int
}

func (s *scriptedInputSource) Drain(ctx context.Context) ([]input.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if s.idx < len(s.events) {
		events := s.events[s.idx]
		s.idx++
		out := make([]input.Event, len(events))
		copy(out, events)
		return out, nil
	}

	if s.err != nil {
		return nil, s.err
	}

	return nil, nil
}
