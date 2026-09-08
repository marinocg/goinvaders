package game

import "testing"

func TestCurrentEnemyMovementIntervalUsesLivingThresholds(t *testing.T) {
	tests := []struct {
		living int
		want   Tick
	}{
		{32, 30}, {27, 28}, {22, 26}, {7, 20}, {2, 18}, {0, 3},
	}
	for _, test := range tests {
		if got := currentEnemyMovementInterval(test.living); got != test.want {
			t.Errorf("living=%d: interval=%d, want %d", test.living, got, test.want)
		}
	}
}

func TestCurrentEnemyMovementIntervalHasMinimum(t *testing.T) {
	if got := currentEnemyMovementInterval(-1); got <= 0 {
		t.Fatalf("negative living count interval=%d, want positive", got)
	}
	if enemyMovementFloor <= 0 {
		t.Fatalf("movement floor=%d, want positive", enemyMovementFloor)
	}
}

func TestCurrentEnemyMovementIntervalIsMonotonic(t *testing.T) {
	previous := currentEnemyMovementInterval(enemyCount)
	for living := enemyCount - 1; living >= 0; living-- {
		current := currentEnemyMovementInterval(living)
		if current > previous {
			t.Fatalf("living=%d: interval=%d increased from %d", living, current, previous)
		}
		previous = current
	}
}

func TestFormationMovementAcceleratesAsEnemiesAreDestroyed(t *testing.T) {
	state, err := NewStateWithDifficulty(40, DifficultyNormal, 18)
	if err != nil {
		t.Fatal(err)
	}
	if got := enemyMovementIntervalFor(state.enemiesAlive(), mustConfig(t, DifficultyNormal), state.waveNumber); got != 30 {
		t.Fatalf("full formation interval=%d, want 30", got)
	}
	for i := 1; i < len(state.enemies); i++ {
		state.enemies[i].Alive = false
	}
	if got := enemyMovementIntervalFor(state.enemiesAlive(), mustConfig(t, DifficultyNormal), state.waveNumber); got != 18 {
		t.Fatalf("last-enemy interval=%d, want 18", got)
	}
	state.enemyElapsed = 17
	before := state.enemies[0]
	state = Step(state, Input{})
	if state.enemies[0].X == before.X && state.enemies[0].Y == before.Y {
		t.Fatal("last enemy did not move on its shortened cadence")
	}
}

func mustConfig(t *testing.T, difficulty Difficulty) DifficultyConfig {
	t.Helper()
	config, err := difficulty.Config()
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func TestNewFormation(t *testing.T) {
	formation, err := newFormation(40, 18)
	if err != nil {
		t.Fatal(err)
	}
	if len(formation) != enemyCount {
		t.Fatalf("expected %d enemy, got %d", enemyCount, len(formation))
	}
	if formation[0] != (Enemy{X: 9, Y: 0, Width: 1, Height: 1, Alive: true}) ||
		formation[7] != (Enemy{X: 30, Y: 0, Width: 1, Height: 1, Alive: true}) ||
		formation[8] != (Enemy{X: 9, Y: 2, Width: 1, Height: 1, Alive: true}) ||
		formation[31] != (Enemy{X: 30, Y: 6, Width: 1, Height: 1, Alive: true}) {
		t.Fatalf("unexpected row-major formation: %+v", formation)
	}
}

func TestFormationRejectsTooSmallArena(t *testing.T) {
	if _, err := newFormation(0, 1); err == nil {
		t.Fatal("expected invalid formation error")
	}
}

func TestMoveFormationPersistsAndPreservesDestroyedEnemies(t *testing.T) {
	enemies := formationFromSlice([]Enemy{{X: 0, Y: 0, Width: 1, Height: 1, Alive: true}})
	enemies[0].Alive = false
	enemies[0].X, enemies[0].Y = 9, 9
	direction := 1
	if moveFormation(&enemies, 4, 4, &direction) {
		t.Fatal("unexpected loss line")
	}
	if enemies[0].X != 9 || enemies[0].Y != 9 {
		t.Fatal("destroyed enemy was moved")
	}
	if enemies[0].Alive {
		t.Fatal("destroyed enemy was resurrected")
	}
	if enemies[0].Width != enemyWidth || enemies[0].Height != enemyHeight {
		t.Fatal("enemy bounds changed")
	}
}

func TestMoveFormationReversesAndDescendsAtBoundary(t *testing.T) {
	enemies := formationFromSlice([]Enemy{{X: 3, Y: 0, Width: 1, Height: 1, Alive: true}})
	direction := 1
	moveFormation(&enemies, 4, 4, &direction)
	if direction != -1 || enemies[0].X != 3 || enemies[0].Y != 1 {
		t.Fatalf("unexpected reversal: direction=%d enemy=%+v", direction, enemies[0])
	}
}

func TestMoveFormationDetectsLossLineOnlyForLivingEnemies(t *testing.T) {
	enemies := formationFromSlice([]Enemy{{X: 0, Y: 3, Width: 1, Height: 1, Alive: true}})
	direction := 1
	if !moveFormation(&enemies, 4, 4, &direction) {
		t.Fatal("expected loss line")
	}
	enemies[0].Alive = false
	if moveFormation(&enemies, 4, 4, &direction) {
		t.Fatal("dead enemy reached loss line")
	}
}
