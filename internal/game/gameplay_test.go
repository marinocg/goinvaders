package game

import "testing"

func TestGameplayPlayerMovementAndSingleProjectile(t *testing.T) {
	state, err := NewState(5, 6)
	if err != nil {
		t.Fatal(err)
	}

	state = StepMany(state, []Input{{MoveX: -1}, {MoveX: -1}, {MoveX: -1}, {MoveX: 1}})
	if got := state.Snapshot().Player.X; got != 1 {
		t.Fatalf("movement was not bounded and ordered, got x=%d", got)
	}
	state = Step(state, Input{Fire: true})
	first := *state.Snapshot().PlayerProjectile
	state = Step(state, Input{MoveX: 1, Fire: true})
	second := state.Snapshot().PlayerProjectile
	if second == nil || second.X != first.X || second.Y != first.Y-projectileStep {
		t.Fatalf("fire replaced or failed to advance active projectile: first=%+v second=%+v", first, second)
	}
}

func TestGameplayFormationBoundaryAndMovementSchedule(t *testing.T) {
	state, err := NewState(2, 8)
	if err != nil {
		t.Fatal(err)
	}
	state = StepMany(state, make([]Input, enemyMovementInterval-1))
	if got := state.enemies[0]; got.X != 0 || got.Y != 0 || state.enemyDirection != 1 {
		t.Fatalf("formation moved before its interval: enemy=%+v direction=%d", got, state.enemyDirection)
	}
	state = Step(state, Input{})
	if got := state.enemies[0]; got.X != 1 || got.Y != 0 {
		t.Fatalf("unexpected scheduled horizontal movement: %+v", got)
	}
	state = StepMany(state, make([]Input, enemyMovementInterval))
	if got := state.enemies[0]; got.X != 1 || got.Y != 1 || state.enemyDirection != -1 {
		t.Fatalf("expected boundary reversal and descent: enemy=%+v direction=%d", got, state.enemyDirection)
	}
}

func TestGameplayCollisionSelectionNearestBeatsFormationOrder(t *testing.T) {
	state, err := NewState(3, 10)
	if err != nil {
		t.Fatal(err)
	}
	// Both enemies intersect the move from y=6 to y=5. The later formation
	// entry is nearest, despite the earlier entry having the lower index.
	state.enemies = formationFromSlice([]Enemy{
		{X: 1, Y: 4, Width: 1, Height: 2, Alive: true},
		{X: 1, Y: 6, Width: 1, Height: 1, Alive: true},
	})
	state.playerProjectile = NewPlayerProjectile(1, 6)
	next := Step(state, Input{})
	if !next.enemies[0].Alive || next.enemies[1].Alive {
		t.Fatalf("expected nearer enemy 1, not enemy 0, to be removed: enemies=%+v", next.enemies)
	}
	if next.playerProjectile != nil || next.score != enemyPointValue {
		t.Fatalf("expected one projectile collision and score, state=%+v", next.Snapshot())
	}
}

func TestGameplayCollisionSelectionTieUsesStableFormationOrder(t *testing.T) { // tie case
	state, err := NewState(3, 10)
	if err != nil {
		t.Fatal(err)
	}
	state.enemies = formationFromSlice([]Enemy{
		{X: 1, Y: 4, Width: 1, Height: 2, Alive: true},
		{X: 1, Y: 4, Width: 1, Height: 2, Alive: true},
	})
	state.playerProjectile = NewPlayerProjectile(1, 6)
	next := Step(state, Input{})
	if next.enemies[0].Alive || !next.enemies[1].Alive {
		t.Fatalf("expected stable formation entry 0, not entry 1, to be removed: enemies=%+v", next.enemies)
	}
}

func TestGameplayProjectileExpiryAndExactlyOneTarget(t *testing.T) {
	state, err := NewState(3, 10)
	if err != nil {
		t.Fatal(err)
	}
	state.enemies = []Enemy{
		{X: 1, Y: 5, Width: 1, Height: 1, Alive: true},
		{X: 1, Y: 4, Width: 1, Height: 1, Alive: true},
	}
	state.playerProjectile = NewPlayerProjectile(1, 6)
	state = Step(state, Input{})
	if state.enemies[0].Alive || !state.enemies[1].Alive || state.playerProjectile != nil {
		t.Fatalf("collision did not remove exactly one target: %+v", state.Snapshot())
	}
	state, _ = NewState(3, 5)
	state = Step(state, Input{Fire: true})
	state = StepMany(state, []Input{{}, {}})
	if state.playerProjectile != nil {
		t.Fatal("projectile did not expire at the top boundary")
	}
}

func TestGameplayScoreLivesRespawnAndFinalLife(t *testing.T) {
	state, err := NewState(3, 8)
	if err != nil {
		t.Fatal(err)
	}
	state.enemies[0] = Enemy{X: 1, Y: 4, Width: 1, Height: 1, Alive: true}
	state.playerProjectile = NewPlayerProjectile(1, 5)
	state = Step(state, Input{})
	if state.score != enemyPointValue || state.lives != 3 || !state.waveAdvancePending || state.outcome != OutcomePlaying {
		t.Fatalf("expected scored wave transition, state=%+v", state.Snapshot())
	}
	state = Step(state, Input{})
	if state.waveNumber != 2 || state.waveAdvancePending || state.enemiesAlive() == 0 {
		t.Fatalf("expected next wave, state=%+v", state.Snapshot())
	}
	state, _ = NewState(3, 6)
	state.enemyElapsed = enemyMovementInterval - 1
	state.enemies[0] = Enemy{X: 1, Y: state.playerY - 2, Width: 1, Height: 2, Alive: true}
	state = Step(state, Input{MoveX: 1})
	if state.lives != 2 || state.playerX != playerStart(state.arenaWidth, state.arenaHeight).X || state.enemyLossLine || state.enemyElapsed != 0 {
		t.Fatalf("expected respawn/reset after life loss, state=%+v", state.Snapshot())
	}
	state.lives = 1
	state.enemyElapsed = enemyMovementInterval - 1
	state.enemies[0].Y = state.playerY - 1
	state = Step(state, Input{})
	if state.lives != 0 || state.outcome != OutcomeGameOver {
		t.Fatalf("expected final-life game over, state=%+v", state.Snapshot())
	}
}

func TestGameplayWinPrecedesLossAndTerminalStateIsFrozen(t *testing.T) {
	state, err := NewState(3, 8)
	if err != nil {
		t.Fatal(err)
	}
	state.enemyElapsed = enemyMovementInterval - 1
	state.enemies[0] = Enemy{X: 1, Y: state.playerY - 2, Width: 1, Height: 2, Alive: true}
	state.playerProjectile = NewPlayerProjectile(1, state.playerY)
	state = Step(state, Input{})
	if state.outcome != OutcomePlaying || !state.waveAdvancePending || state.lives != 3 {
		t.Fatalf("wave clear did not take precedence over loss line: state=%+v", state.Snapshot())
	}
	state = Step(state, Input{})
	if state.waveNumber != 2 {
		t.Fatalf("expected wave 2 after transition: state=%+v", state.Snapshot())
	}
	state = StepMany(state, []Input{{MoveX: -1, Fire: true}, {MoveX: 1, Fire: true}})
	if state.waveNumber != 2 || state.outcome != OutcomePlaying {
		t.Fatalf("next wave did not remain playable: state=%+v", state.Snapshot())
	}

	gameOver, err := NewState(3, 8)
	if err != nil {
		t.Fatal(err)
	}
	gameOver.lives = 1
	gameOver.enemyElapsed = enemyMovementInterval - 1
	gameOver.enemies[0] = Enemy{X: 0, Y: gameOver.playerY - 1, Width: 1, Height: 1, Alive: true}
	gameOver.playerProjectile = NewPlayerProjectile(0, 1)
	gameOver = Step(gameOver, Input{})
	if gameOver.outcome != OutcomeGameOver {
		t.Fatalf("expected game over before freeze check, state=%+v", gameOver.Snapshot())
	}
	gameOverFrozen := gameOver.Snapshot()
	gameOver = StepMany(gameOver, []Input{
		{MoveX: -1, Fire: true},
		{MoveX: 1, Fire: true},
		{MoveX: -1, Fire: true},
		{},
	})
	if got := gameOver.Snapshot(); got != gameOverFrozen {
		t.Fatalf("terminal game over changed after later input: before=%+v after=%+v", gameOverFrozen, got)
	}
}
