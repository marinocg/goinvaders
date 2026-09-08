package game

const enemyPointValue = 10

// CollisionResult describes the single enemy removal caused by a projectile.
type CollisionResult struct {
	Hit        bool
	EnemyIndex int
	ScoreDelta int
	BonusHit   bool
}

// ResolveProjectileCollisions moves no entities other than the selected target
// and projectile. It is also available to the score/terminal-state layer.
func ResolveProjectileCollisions(state State) (State, CollisionResult) {
	if !validState(state) || state.playerProjectile == nil {
		return state, CollisionResult{}
	}
	projectile := *state.playerProjectile
	oldY := projectile.Y
	if !moveProjectile(&projectile, state.arenaHeight) {
		state.playerProjectile = nil
		return state, CollisionResult{}
	}
	// Upward travel enters the new (higher) cell before leaving the old one.
	if state.damageShield(projectile.X, projectile.Y, -1) || state.damageShield(projectile.X, oldY, -1) {
		state.playerProjectile = nil
		return state, CollisionResult{}
	}

	selected := -1
	selectedDistance := 0
	for i, enemy := range state.enemies {
		if !enemy.Alive || projectile.X < enemy.X || projectile.X > enemy.X+enemy.Width-1 ||
			oldY < enemy.Y || projectile.Y > enemy.Y+enemy.Height-1 {
			continue
		}
		distance := oldY - (enemy.Y + enemy.Height - 1)
		if selected < 0 || distance < selectedDistance {
			selected = i
			selectedDistance = distance
		}
	}
	state.playerProjectile = &projectile
	if selected < 0 {
		if state.bonusTarget != nil && (rectanglesOverlap(projectile.X, projectile.Y, 1, 1, state.bonusTarget.X, state.bonusTarget.Y, state.bonusTarget.Width, state.bonusTarget.Height) || rectanglesOverlap(projectile.X, oldY, 1, 1, state.bonusTarget.X, state.bonusTarget.Y, state.bonusTarget.Width, state.bonusTarget.Height)) {
			state.playerProjectile = nil
			state.bonusTarget = nil
			return state, CollisionResult{Hit: true, BonusHit: true, ScoreDelta: bonusTargetScore}
		}
		return state, CollisionResult{}
	}
	state.enemies[selected].Alive = false
	state.playerProjectile = nil
	return state, CollisionResult{Hit: true, EnemyIndex: selected, ScoreDelta: enemyPointValue}
}
