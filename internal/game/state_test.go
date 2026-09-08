package game

import (
	"errors"
	"reflect"
	"testing"
)

func TestNewState(t *testing.T) {
	t.Parallel()

	t.Run("rejects invalid arena width", func(t *testing.T) {
		t.Parallel()

		_, err := NewState(0)
		if !errors.Is(err, ErrInvalidArenaWidth) {
			t.Fatalf("expected ErrInvalidArenaWidth, got %v", err)
		}
	})
	t.Run("rejects invalid arena height", func(t *testing.T) {
		t.Parallel()
		_, err := NewState(9, 0)
		if !errors.Is(err, ErrInvalidArenaHeight) {
			t.Fatalf("expected ErrInvalidArenaHeight, got %v", err)
		}
	})

	t.Run("centers player and initializes tick", func(t *testing.T) {
		t.Parallel()

		state, err := NewState(9)
		if err != nil {
			t.Fatalf("NewState returned error: %v", err)
		}

		snap := state.Snapshot()
		if snap.Tick != 0 {
			t.Fatalf("expected tick 0, got %d", snap.Tick)
		}
		if snap.Player.X != 4 || snap.Player.Y != 0 || snap.Player.Width != 1 || snap.Player.Height != 1 {
			t.Fatalf("expected centered player bounds, got %#v", snap.Player)
		}
	})
	t.Run("initializes the full default formation", func(t *testing.T) {
		state, err := NewState(40, 18)
		if err != nil {
			t.Fatal(err)
		}
		if len(state.enemies) != enemyCount || state.enemies[0].X != 9 || state.enemies[31].X != 30 || state.enemies[31].Y != 6 {
			t.Fatalf("unexpected default formation: len=%d first=%+v last=%+v", len(state.enemies), state.enemies[0], state.enemies[31])
		}
	})
}

func TestRespawnRestoresFormationSlots(t *testing.T) {
	state, err := NewState(40, 18)
	if err != nil {
		t.Fatal(err)
	}
	state.enemies[0].X, state.enemies[0].Y = 20, 10
	state.enemies[8].X, state.enemies[8].Y = 20, 10
	state.respawn()
	if state.enemies[0].X != 9 || state.enemies[0].Y != 0 || state.enemies[8].X != 9 || state.enemies[8].Y != 2 {
		t.Fatalf("respawn did not restore row-major slots: first=%+v second-row=%+v", state.enemies[0], state.enemies[8])
	}
}

func TestOrderedMovementAndFire(t *testing.T) {
	state, err := NewState(9, 6)
	if err != nil {
		t.Fatal(err)
	}
	state = StepMany(state, []Input{{MoveX: 1}, {MoveX: -1}, {MoveX: -1, Fire: true}, {Fire: true}})
	snap := state.Snapshot()
	if snap.Player.X != 3 {
		t.Fatalf("expected ordered movement at x=3, got %d", snap.Player.X)
	}
	if snap.PlayerProjectile == nil {
		t.Fatal("expected active player projectile")
	}
	if got := *snap.PlayerProjectile; got != (ProjectileSnapshot{X: 3, Y: 2}) {
		t.Fatalf("unexpected projectile: %#v", got)
	}
}

func TestStepIsDeterministic(t *testing.T) {
	t.Parallel()

	initial, err := NewState(7)
	if err != nil {
		t.Fatalf("NewState returned error: %v", err)
	}

	inputs := []Input{
		{MoveX: 1},
		{MoveX: 1},
		{MoveX: -1},
		{MoveX: -99},
		{MoveX: 99, Fire: true},
		{},
	}

	left := StepMany(initial, inputs).Snapshot()
	right := StepMany(initial, inputs).Snapshot()

	if !reflect.DeepEqual(left, right) {
		t.Fatalf("expected deterministic transition, got %#v and %#v", left, right)
	}
}

func TestEnemyMovementUsesSequentialDueIntervals(t *testing.T) {
	state, err := NewState(5, 5)
	if err != nil {
		t.Fatal(err)
	}
	state.enemyElapsed = enemyMovementInterval - 1
	state = Step(state, Input{})
	if state.enemies[0].X != enemyStartX+enemyStep {
		t.Fatalf("expected one due movement, got x=%d", state.enemies[0].X)
	}
	state.enemyElapsed = enemyMovementInterval * 2
	state = Step(state, Input{})
	if state.enemies[0].X != enemyStartX+3*enemyStep {
		t.Fatalf("expected sequential movement, got x=%d", state.enemies[0].X)
	}
}

func TestEnemyKillUsesNewCadenceWithoutLosingElapsedTicks(t *testing.T) {
	state, err := NewState(3, 10)
	if err != nil {
		t.Fatal(err)
	}
	state.enemies = []Enemy{
		{X: 0, Y: 1, Width: 1, Height: 1, Alive: true},
		{X: 2, Y: 1, Width: 1, Height: 1, Alive: true},
	}
	state.enemyElapsed = currentEnemyMovementInterval(2) - 1
	state.playerProjectile = NewPlayerProjectile(0, 2)
	state = Step(state, Input{})
	if state.outcome != OutcomePlaying || state.enemies[0].Alive || state.enemies[1].X != 2 {
		t.Fatalf("kill caused an unexpected movement: state=%+v", state.Snapshot())
	}
	if state.enemyElapsed != 0 {
		t.Fatalf("expected exactly the due interval to be consumed, elapsed=%d", state.enemyElapsed)
	}
	state = Step(state, Input{})
	if state.enemies[1].X != 2 {
		t.Fatalf("remaining enemy moved before the new interval elapsed: x=%d", state.enemies[1].X)
	}
}

func TestLossLineConsumesLifeAndRespawns(t *testing.T) {
	state, err := NewState(3, 4)
	if err != nil {
		t.Fatal(err)
	}
	state.enemies[0].Y = state.playerY - enemyHeight
	state.enemyElapsed = enemyMovementInterval - 1
	state = Step(state, Input{})
	if state.Snapshot().Lives != 2 || state.Snapshot().Outcome != OutcomePlaying {
		t.Fatalf("expected one life lost while playing, got %#v", state.Snapshot())
	}
	if state.enemies[0].X != enemyStartX || state.enemies[0].Y != enemyStartY {
		t.Fatalf("expected living formation reset, got %#v", state.enemies[0])
	}
}

func TestScoreAndAdvanceOnLastEnemy(t *testing.T) {
	state, err := NewState(3, 4)
	if err != nil {
		t.Fatal(err)
	}
	state.enemies[0].Y = 1
	state.playerProjectile = &Projectile{X: 0, Y: 2}
	state = Step(state, Input{})
	snap := state.Snapshot()
	if snap.Score != enemyPointValue || snap.Outcome != OutcomePlaying || snap.WaveNumber != 1 || !snap.WaveAdvancePending {
		t.Fatalf("expected scored pending wave advance, got %#v", snap)
	}
	state = Step(state, Input{})
	if snap = state.Snapshot(); snap.WaveNumber != 2 || snap.LivingEnemyCount != 1 || snap.Score != enemyPointValue {
		t.Fatalf("expected next wave with preserved score, got %#v", snap)
	}
}

func TestGameOverDoesNotRespawnOrReset(t *testing.T) {
	state, err := NewState(3, 4)
	if err != nil {
		t.Fatal(err)
	}
	state.lives = 1
	state.enemies[0].X = 1
	state.enemies[0].Y = 2
	state.enemyDirection = -1
	state.enemyElapsed = enemyMovementInterval - 1
	state = Step(state, Input{})
	if state.outcome != OutcomeGameOver || state.lives != 0 || state.enemies[0].X != 0 || state.enemies[0].Y != 2 {
		t.Fatalf("expected direct game over without reset, got %#v", state)
	}
}

func TestTerminalStateIsNoOp(t *testing.T) {
	state, err := NewState(3, 2)
	if err != nil {
		t.Fatal(err)
	}
	state.outcome = OutcomeWin
	state.score = 42
	before := state
	got := Step(state, Input{MoveX: 1, Fire: true})
	if !reflect.DeepEqual(got, before) {
		t.Fatalf("expected terminal step no-op, got %#v", got)
	}
}

func TestStepIgnoresInvalidInput(t *testing.T) {
	t.Parallel()

	state, err := NewState(3)
	if err != nil {
		t.Fatalf("NewState returned error: %v", err)
	}

	state = Step(state, Input{MoveX: 100})
	if got := state.Snapshot().Player.X; got != 1 {
		t.Fatalf("expected invalid input to be ignored at x=1, got %d", got)
	}

	state = Step(state, Input{MoveX: -100, Fire: true})
	if got := state.Snapshot(); got.Player.X != 1 || got.PlayerProjectile != nil {
		t.Fatalf("expected invalid input to be a no-op, got %#v", got)
	}

	state = Step(state, Input{MoveX: -1})
	if got := state.Snapshot().Player.X; got != 0 {
		t.Fatalf("expected valid movement to reach left edge 0, got %d", got)
	}
}

func TestStepRejectsInvalidStateWithoutRepair(t *testing.T) {
	t.Parallel()

	degenerate := State{tick: -10, arenaWidth: 0, playerX: -20}
	if next := Step(degenerate, Input{}); !reflect.DeepEqual(next, degenerate) {
		t.Fatalf("expected invalid state to remain unchanged, got %#v", next)
	}
}
