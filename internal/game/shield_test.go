package game

import "testing"

func TestDefaultShieldsAreDeterministicAndFull(t *testing.T) {
	state, err := NewState(40, 18)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.shields) != 4 {
		t.Fatalf("expected four shields, got %d", len(state.shields))
	}
	want := []int{16, 14, 12, 10}
	for i, shield := range state.shields {
		if shield.X != want[i] || shield.Y != 15 {
			t.Fatalf("shield %d has position %+v", i, shield)
		}
		for y := range shield.Cells {
			for x := range shield.Cells[y] {
				if shield.Cells[y][x] != shieldDurability {
					t.Fatal("shield is not full")
				}
			}
		}
	}
}

func TestShieldConsumesRepeatedHitsAndSnapshotIsCopy(t *testing.T) {
	state, _ := NewState(40, 18)
	x, y := state.shields[0].X+1, state.shields[0].Y
	state.playerProjectile = NewPlayerProjectile(x, y+1)
	state, _ = ResolveProjectileCollisions(state)
	if state.shields[0].Cells[0][1] != 1 || state.playerProjectile != nil {
		t.Fatal("first hit was not consumed")
	}
	state.playerProjectile = NewPlayerProjectile(x, y+1)
	state, _ = ResolveProjectileCollisions(state)
	if state.shields[0].Cells[0][1] != 0 {
		t.Fatal("second hit did not destroy cell")
	}
	snapshot := state.Snapshot()
	snapshot.Shields[0].Cells[0][0] = 0
	if state.shields[0].Cells[0][0] != shieldDurability {
		t.Fatal("snapshot exposed mutable state")
	}
}

func TestPlayerProjectileDestroysShieldCellAfterTwoInterceptions(t *testing.T) {
	state, _ := NewState(40, 18)
	x, y := state.shields[0].X+1, state.shields[0].Y
	for shield := range state.shields {
		for row := range state.shields[shield].Cells {
			for column := range state.shields[shield].Cells[row] {
				state.shields[shield].Cells[row][column] = 0
			}
		}
	}
	state.shields[0].Cells[0][1] = shieldDurability
	for hit := 0; hit < shieldDurability; hit++ {
		state.playerProjectile = NewPlayerProjectile(x, y+1)
		state, _ = ResolveProjectileCollisions(state)
	}
	if state.shields[0].Cells[0][1] != 0 {
		t.Fatalf("player projectile did not destroy shield cell: %d", state.shields[0].Cells[0][1])
	}
	state.playerProjectile = NewPlayerProjectile(x, y+1)
	state, _ = ResolveProjectileCollisions(state)
	if state.shields[0].Cells[0][1] != 0 || state.playerProjectile == nil {
		t.Fatal("destroyed shield cell incorrectly intercepted a later player projectile")
	}
}

func TestShieldsPersistAcrossRespawn(t *testing.T) {
	state, _ := NewState(40, 18)
	state.shields[0].Cells[0][0] = 0
	state.lives = 2
	state.respawn()
	if state.shields[0].Cells[0][0] != 0 {
		t.Fatal("respawn restored shield unexpectedly")
	}
}

func TestInstallWaveRestoresShieldsAndPreservesRunProgress(t *testing.T) {
	state, _ := NewState(40, 18)
	state.shields[0].Cells[0][0] = 0
	state.score = 70
	state.lives = 2
	state.enemyProjectiles = []EnemyProjectile{{X: 1, Y: 1}}

	if !state.installWave() {
		t.Fatal("expected a valid wave to install")
	}
	if state.shields[0].Cells[0][0] != shieldDurability || state.score != 70 || state.lives != 2 {
		t.Fatalf("wave reset did not restore/preserve state: %+v", state.Snapshot())
	}
	if len(state.enemyProjectiles) != 0 || state.outcome != OutcomePlaying {
		t.Fatalf("wave reset did not clear transient state: %+v", state.Snapshot())
	}
}
