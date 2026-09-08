package game

import "testing"

func TestSelectEnemyShooterSkipsDeadAndUsesLowestInColumn(t *testing.T) {
	enemies := []Enemy{
		{X: 0, Y: 0, Width: 1, Height: 1, Alive: true},
		{X: 0, Y: 2, Width: 1, Height: 1, Alive: false},
		{X: 0, Y: 3, Width: 1, Height: 1, Alive: true},
		{X: 3, Y: 1, Width: 1, Height: 1, Alive: true},
	}
	if got := selectEnemyShooter(enemies, 0); got != 2 {
		t.Fatalf("selected shooter %d, want 2", got)
	}
	if got := selectEnemyShooter(enemies, 1); got != 3 {
		t.Fatalf("rotating selection chose %d, want 3", got)
	}
}

func TestEnemyFireUsesDeterministicEligibleShooter(t *testing.T) {
	makeState := func() State {
		state, err := NewState(7, 12)
		if err != nil {
			t.Fatal(err)
		}
		state.enemies = []Enemy{
			{X: 0, Y: 1, Width: 1, Height: 1, Alive: true},
			{X: 0, Y: 3, Width: 1, Height: 1, Alive: true},
			{X: 3, Y: 2, Width: 1, Height: 1, Alive: true},
		}
		state.enemyFireElapsed = enemyFireInterval - 1
		return state
	}

	first := Step(makeState(), Input{})
	second := Step(makeState(), Input{})
	if len(first.enemyProjectiles) != 1 || len(second.enemyProjectiles) != 1 {
		t.Fatalf("expected one deterministic shot, got %#v and %#v", first.enemyProjectiles, second.enemyProjectiles)
	}
	if first.enemyProjectiles[0] != second.enemyProjectiles[0] {
		t.Fatalf("equivalent states fired different shots: %#v and %#v", first.enemyProjectiles[0], second.enemyProjectiles[0])
	}
	if first.enemyProjectiles[0].X != 0 || first.enemyProjectiles[0].Y != 4 {
		t.Fatalf("expected lowest eligible shooter in column 0, got %#v", first.enemyProjectiles[0])
	}
}

func TestEnemyFireCadenceAndLimit(t *testing.T) {
	state, err := NewState(3, 100)
	if err != nil {
		t.Fatal(err)
	}
	state.enemies = []Enemy{{X: 0, Y: 0, Width: 1, Height: 1, Alive: true}}
	state.enemyFireElapsed = enemyFireInterval - 1
	state = Step(state, Input{})
	if len(state.enemyProjectiles) != 1 || state.enemyProjectiles[0].Y != 1 {
		t.Fatalf("expected one newly created shot, got %#v", state.enemyProjectiles)
	}
	state.enemyFireElapsed = enemyFireInterval - 1
	state.enemyProjectiles = append(state.enemyProjectiles, EnemyProjectile{X: 2, Y: 1})
	state = Step(state, Input{})
	if len(state.enemyProjectiles) != maxEnemyProjectiles {
		t.Fatalf("expected active shot limit, got %d", len(state.enemyProjectiles))
	}
}

func TestEnemyProjectileMovesAndExpires(t *testing.T) {
	state, err := NewState(3, 3)
	if err != nil {
		t.Fatal(err)
	}
	state.enemies = []Enemy{{X: 0, Y: 0, Width: 1, Height: 1, Alive: true}}
	state.enemyProjectiles = []EnemyProjectile{{X: 0, Y: 1}}
	state = Step(state, Input{})
	if len(state.enemyProjectiles) != 1 || state.enemyProjectiles[0].Y != 2 {
		t.Fatalf("expected shot to move down, got %#v", state.enemyProjectiles)
	}
	state = Step(state, Input{})
	if len(state.enemyProjectiles) != 0 {
		t.Fatalf("expected expired shot to be removed, got %#v", state.enemyProjectiles)
	}
}

func TestEnemyProjectileUsesDifficultySpeed(t *testing.T) {
	state, err := NewStateWithDifficulty(3, DifficultyHard, 8)
	if err != nil {
		t.Fatal(err)
	}
	state.enemies = []Enemy{{X: 0, Y: 0, Width: 1, Height: 1, Alive: true}}
	state.enemyProjectiles = []EnemyProjectile{{X: 0, Y: 1}}
	state.advanceEnemyProjectiles()
	if len(state.enemyProjectiles) != 1 || state.enemyProjectiles[0].Y != 3 {
		t.Fatalf("hard projectile speed was not applied: %#v", state.enemyProjectiles)
	}
}

func TestEnemyProjectileDamagesAndConsumesShield(t *testing.T) {
	state, _ := NewState(40, 18)
	x, y := state.shields[0].X+1, state.shields[0].Y-1
	state.enemyProjectiles = []EnemyProjectile{{X: x, Y: y}}
	state.advanceEnemyProjectiles()
	if len(state.enemyProjectiles) != 0 || state.shields[0].Cells[0][1] != shieldDurability-1 {
		t.Fatalf("enemy shot did not consume on shield impact: %+v", state.Snapshot())
	}
}

func TestEnemyProjectileDestroysShieldCellAfterTwoInterceptions(t *testing.T) {
	state, _ := NewState(40, 18)
	x, y := state.shields[0].X+1, state.shields[0].Y-1
	for shield := range state.shields {
		for row := range state.shields[shield].Cells {
			for column := range state.shields[shield].Cells[row] {
				state.shields[shield].Cells[row][column] = 0
			}
		}
	}
	state.shields[0].Cells[0][1] = shieldDurability
	for hit := 0; hit < shieldDurability; hit++ {
		state.enemyProjectiles = []EnemyProjectile{{X: x, Y: y}}
		state.advanceEnemyProjectiles()
	}
	if state.shields[0].Cells[0][1] != 0 {
		t.Fatalf("enemy projectile did not destroy shield cell: %d", state.shields[0].Cells[0][1])
	}
	state.enemyProjectiles = []EnemyProjectile{{X: x, Y: y}}
	state.advanceEnemyProjectiles()
	if len(state.enemyProjectiles) != 1 {
		t.Fatal("destroyed shield cell incorrectly intercepted a later enemy projectile")
	}
}

func TestIdenticalHistoriesProduceIdenticalEnemyFireAndBonusSchedules(t *testing.T) {
	first, _ := NewStateWithDifficulty(40, DifficultyHard, 18)
	second := cloneState(first)
	var firstHistory, secondHistory []string
	for tick := 0; tick < 1600; tick++ {
		first = Step(first, Input{})
		second = Step(second, Input{})
		firstHistory = append(firstHistory, scheduleMarker(first))
		secondHistory = append(secondHistory, scheduleMarker(second))
	}
	for i := range firstHistory {
		if firstHistory[i] != secondHistory[i] {
			t.Fatalf("schedule diverged at tick %d: %q/%q", i, firstHistory[i], secondHistory[i])
		}
	}
}

func cloneState(state State) State {
	clone := state
	clone.enemies = append([]Enemy(nil), state.enemies...)
	clone.enemyProjectiles = append([]EnemyProjectile(nil), state.enemyProjectiles...)
	clone.shields = append([]Shield(nil), state.shields...)
	if state.playerProjectile != nil {
		projectile := *state.playerProjectile
		clone.playerProjectile = &projectile
	}
	if state.bonusTarget != nil {
		bonusTarget := *state.bonusTarget
		clone.bonusTarget = &bonusTarget
	}
	return clone
}

func scheduleMarker(state State) string {
	marker := ""
	if state.bonusTarget != nil {
		marker += "B"
	}
	for _, projectile := range state.enemyProjectiles {
		marker += string(rune('a' + projectile.X))
	}
	return marker
}

func TestEnemyProjectileHitCostsOneLifeAndRespawns(t *testing.T) {
	state, err := NewState(3, 4)
	if err != nil {
		t.Fatal(err)
	}
	state.enemies = []Enemy{{X: 0, Y: 0, Width: 1, Height: 1, Alive: true}}
	state.enemyProjectiles = []EnemyProjectile{{X: state.playerX, Y: state.playerY - 1}}
	state = Step(state, Input{})
	if state.lives != 2 || state.outcome != OutcomePlaying || len(state.enemyProjectiles) != 0 {
		t.Fatalf("expected one life loss and cleared projectiles, got %#v", state.Snapshot())
	}
}

func TestEnemyProjectileHitCausesGameOverWithoutRespawn(t *testing.T) {
	state, err := NewState(3, 4)
	if err != nil {
		t.Fatal(err)
	}
	state.lives = 1
	state.enemies = []Enemy{{X: 0, Y: 0, Width: 1, Height: 1, Alive: true}}
	state.playerProjectile = NewPlayerProjectile(state.playerX, state.playerY-1)
	state.enemyProjectiles = []EnemyProjectile{{X: state.playerX, Y: state.playerY - 1}}
	state = Step(state, Input{})
	if state.lives != 0 || state.outcome != OutcomeGameOver || state.playerProjectile != nil || len(state.enemyProjectiles) != 0 {
		t.Fatalf("expected game over, got %#v", state.Snapshot())
	}
}

func TestEnemyProjectileCostsExactlyOneLifePerHit(t *testing.T) {
	state, err := NewState(3, 4)
	if err != nil {
		t.Fatal(err)
	}
	state.enemies = []Enemy{{X: 0, Y: 0, Width: 1, Height: 1, Alive: true}}
	for wantLives := 2; wantLives >= 1; wantLives-- {
		state.enemyProjectiles = []EnemyProjectile{{X: state.playerX, Y: state.playerY - 1}}
		state = Step(state, Input{})
		if state.lives != wantLives || state.outcome != OutcomePlaying {
			t.Fatalf("expected one life lost with respawn, got %#v", state.Snapshot())
		}
	}
	state.enemyProjectiles = []EnemyProjectile{
		{X: state.playerX, Y: state.playerY - 1},
		{X: state.playerX, Y: state.playerY - 1},
	}
	state = Step(state, Input{})
	if state.lives != 0 || state.outcome != OutcomeGameOver {
		t.Fatalf("expected final hit to end game after one life, got %#v", state.Snapshot())
	}
}

func TestEnemyProjectileIsSuppressedAfterPlayerWinsOnSameTick(t *testing.T) {
	state, err := NewState(3, 8)
	if err != nil {
		t.Fatal(err)
	}
	state.enemies = []Enemy{{X: 1, Y: 5, Width: 1, Height: 1, Alive: true}}
	state.playerProjectile = NewPlayerProjectile(1, 6)
	state.enemyProjectiles = []EnemyProjectile{{X: state.playerX, Y: state.playerY - 1}}

	state = Step(state, Input{})
	if state.outcome != OutcomePlaying || !state.waveAdvancePending || state.lives != 3 {
		t.Fatalf("player wave clear must take precedence over enemy hit: %#v", state.Snapshot())
	}
}

func TestEnemyProjectileHitTakesPrecedenceOverNewFireDuringLossTransition(t *testing.T) {
	state, err := NewState(3, 4)
	if err != nil {
		t.Fatal(err)
	}
	state.enemies = []Enemy{{X: 0, Y: 0, Width: 1, Height: 1, Alive: true}}
	state.enemyLossLine = true
	state.enemyProjectiles = []EnemyProjectile{{X: state.playerX, Y: state.playerY - 1}}
	state.enemyFireElapsed = enemyFireInterval - 1

	state = Step(state, Input{})
	if state.lives != 2 || state.outcome != OutcomePlaying || len(state.enemyProjectiles) != 0 {
		t.Fatalf("transition must respawn and clear enemy shots: %#v", state.Snapshot())
	}
}

func TestEnemyFireStopsInTerminalState(t *testing.T) {
	state, err := NewState(3, 8)
	if err != nil {
		t.Fatal(err)
	}
	state.outcome = OutcomeGameOver
	state.enemyFireElapsed = enemyFireInterval - 1

	next := Step(state, Input{})
	if len(next.enemyProjectiles) != 0 || next.tick != state.tick {
		t.Fatalf("terminal state advanced or fired: %#v", next.Snapshot())
	}
}

func TestIdenticalHistoriesProduceIdenticalEnemyFireSchedule(t *testing.T) {
	first, err := NewStateWithDifficulty(40, DifficultyHard, 18)
	if err != nil {
		t.Fatal(err)
	}
	second := cloneState(first)
	var firstShots, secondShots []EnemyProjectileSnapshot
	for tick := 0; tick < 180; tick++ {
		first = Step(first, Input{})
		second = Step(second, Input{})
		firstSnapshot, secondSnapshot := first.Snapshot(), second.Snapshot()
		if firstSnapshot.EnemyProjectileCount != secondSnapshot.EnemyProjectileCount {
			t.Fatalf("tick %d projectile counts differ: %d/%d", tick, firstSnapshot.EnemyProjectileCount, secondSnapshot.EnemyProjectileCount)
		}
		if firstSnapshot.EnemyProjectileCount > 0 {
			firstShots = append(firstShots, firstSnapshot.EnemyProjectiles[0])
			secondShots = append(secondShots, secondSnapshot.EnemyProjectiles[0])
		}
	}
	for i := range firstShots {
		if firstShots[i] != secondShots[i] {
			t.Fatalf("shot %d differs: %#v/%#v", i, firstShots[i], secondShots[i])
		}
	}
}
