package game

import "testing"

func TestPlayerStartsCenteredInBottomRow(t *testing.T) {
	state, err := NewState(8, 5)
	if err != nil {
		t.Fatal(err)
	}

	got := state.Snapshot().Player
	if got != (PlayerSnapshot{X: 3, Y: 4, Width: 1, Height: 1}) {
		t.Fatalf("expected centered bottom-row player, got %#v", got)
	}
}

func TestPlayerMovementClampsToPlayfield(t *testing.T) {
	state, err := NewState(3, 2)
	if err != nil {
		t.Fatal(err)
	}

	state = StepMany(state, []Input{{MoveX: -1}, {MoveX: -1}, {MoveX: -1}})
	if got := state.Snapshot().Player.X; got != 0 {
		t.Fatalf("expected left boundary, got %d", got)
	}
	state = StepMany(state, []Input{{MoveX: 1}, {MoveX: 1}, {MoveX: 1}, {MoveX: 1}})
	if got := state.Snapshot().Player.X; got != 2 {
		t.Fatalf("expected right boundary, got %d", got)
	}
}

func TestPlayerBoundsFitNarrowPlayfield(t *testing.T) {
	state, err := NewState(1, 1)
	if err != nil {
		t.Fatal(err)
	}

	if got := state.Snapshot().Player; got != (PlayerSnapshot{X: 0, Y: 0, Width: 1, Height: 1}) {
		t.Fatalf("expected player bounds inside narrow playfield, got %#v", got)
	}
}

func TestPlayerMovementPreservesInputOrder(t *testing.T) {
	state, err := NewState(9, 4)
	if err != nil {
		t.Fatal(err)
	}

	got := StepMany(state, []Input{{MoveX: -1}, {MoveX: -1}, {MoveX: 1}}).Snapshot().Player.X
	if got != 3 {
		t.Fatalf("expected ordered movement to end at x=3, got %d", got)
	}
}
