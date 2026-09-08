package game

type Projectile struct {
	X, Y int
}

const projectileStep = 1

type ProjectileSnapshot struct {
	X, Y int
}

func NewPlayerProjectile(x, y int) *Projectile {
	return &Projectile{X: x, Y: y}
}

func (p Projectile) Snapshot() ProjectileSnapshot {
	return ProjectileSnapshot{X: p.X, Y: p.Y}
}

func moveProjectile(p *Projectile, arenaHeight int) bool {
	if p == nil {
		return false
	}
	p.Y -= projectileStep
	return p.Y > 0 && p.Y < arenaHeight
}
