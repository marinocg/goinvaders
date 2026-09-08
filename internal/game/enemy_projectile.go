package game

// EnemyProjectile is a one-cell shot travelling from an invader toward the
// player. It deliberately has a separate type from the player's projectile.
type EnemyProjectile struct {
	X, Y int
}

type EnemyProjectileSnapshot struct {
	X, Y int
}

func (p EnemyProjectile) Snapshot() EnemyProjectileSnapshot {
	return EnemyProjectileSnapshot{X: p.X, Y: p.Y}
}

// moveEnemyProjectile advances one tick and reports whether it remains inside
// the arena. The optional speed keeps the legacy one-cell helper behavior for
// focused mechanics callers while allowing difficulty presets to control shots.
func moveEnemyProjectile(p *EnemyProjectile, arenaHeight int, speeds ...int) bool {
	if p == nil {
		return false
	}
	speed := 1
	if len(speeds) > 0 && speeds[0] > 0 {
		speed = speeds[0]
	}
	p.Y += speed
	return p.Y >= 0 && p.Y < arenaHeight
}

func (s *State) advanceEnemyProjectiles() {
	config, err := s.difficulty.Config()
	if err != nil {
		return
	}
	active := s.enemyProjectiles[:0]
	for i := range s.enemyProjectiles {
		oldY := s.enemyProjectiles[i].Y
		if !moveEnemyProjectile(&s.enemyProjectiles[i], s.arenaHeight, config.EnemyProjectileSpeed) {
			continue
		}
		blocked := false
		for y := oldY + 1; y <= s.enemyProjectiles[i].Y; y++ {
			if s.damageShield(s.enemyProjectiles[i].X, y, 1) {
				blocked = true
				break
			}
		}
		if !blocked {
			active = append(active, s.enemyProjectiles[i])
		}
	}
	s.enemyProjectiles = active
}

// enemyProjectileHit consumes every shot intersecting the player, but reports
// one hit so a tick can cost at most one life.
func (s *State) enemyProjectileHit() bool {
	active := s.enemyProjectiles[:0]
	hit := false
	for _, p := range s.enemyProjectiles {
		if rectanglesOverlap(p.X, p.Y, 1, 1, s.playerX, s.playerY, playerWidth, playerHeight) {
			hit = true
			continue
		}
		active = append(active, p)
	}
	s.enemyProjectiles = active
	return hit
}
