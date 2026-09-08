package input

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestCommandNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   Command
		want Command
	}{
		{name: "noop stays noop", in: CommandNoop, want: CommandNoop},
		{name: "left is valid", in: CommandMoveLeft, want: CommandMoveLeft},
		{name: "right is valid", in: CommandMoveRight, want: CommandMoveRight},
		{name: "fire is valid", in: CommandFire, want: CommandFire},
		{name: "quit is valid", in: CommandQuit, want: CommandQuit},
		{name: "pause is valid", in: CommandPause, want: CommandPause},
		{name: "enter is valid", in: CommandEnter, want: CommandEnter},
		{name: "easy difficulty is valid", in: CommandDifficultyEasy, want: CommandDifficultyEasy},
		{name: "normal difficulty is valid", in: CommandDifficultyNormal, want: CommandDifficultyNormal},
		{name: "hard difficulty is valid", in: CommandDifficultyHard, want: CommandDifficultyHard},
		{name: "unsupported high value maps to noop", in: Command(99), want: CommandNoop},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.in.Normalize()
			if got != tt.want {
				t.Fatalf("Normalize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEventNormalize(t *testing.T) {
	t.Parallel()

	event := Event{Command: Command(200)}
	got := event.Normalize()

	if got.Command != CommandNoop {
		t.Fatalf("Normalize() command = %v, want %v", got.Command, CommandNoop)
	}
}

func TestQueueSourceDrainPreservesOrder(t *testing.T) {
	t.Parallel()

	src := NewQueueSource([]Event{
		{Command: CommandMoveRight},
		{Command: Command(255)},
		{Command: CommandFire},
		{Command: CommandMoveLeft},
	})

	got, err := src.Drain(context.Background())
	if err != nil {
		t.Fatalf("Drain() returned error: %v", err)
	}

	want := []Event{
		{Command: CommandMoveRight},
		{Command: CommandNoop},
		{Command: CommandFire},
		{Command: CommandMoveLeft},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Drain() = %#v, want %#v", got, want)
	}
}

func TestQueueSourceDrainConsumesEvents(t *testing.T) {
	t.Parallel()

	src := NewQueueSource([]Event{{Command: CommandQuit}})

	first, err := src.Drain(context.Background())
	if err != nil {
		t.Fatalf("first Drain() returned error: %v", err)
	}
	if len(first) != 1 {
		t.Fatalf("first Drain() length = %d, want 1", len(first))
	}

	second, err := src.Drain(context.Background())
	if err != nil {
		t.Fatalf("second Drain() returned error: %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("second Drain() length = %d, want 0", len(second))
	}
}

func TestQueueSourceDrainRespectsContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	src := NewQueueSource([]Event{{Command: CommandFire}})

	got, err := src.Drain(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Drain() error = %v, want %v", err, context.Canceled)
	}
	if got != nil {
		t.Fatalf("Drain() events = %#v, want nil", got)
	}
}
