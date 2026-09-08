package game

const (
	shieldCount      = 4
	shieldWidth      = 3
	shieldHeight     = 2
	shieldDurability = 2
)

type Shield struct {
	X, Y  int
	Cells [shieldHeight][shieldWidth]uint8
}

type ShieldSnapshot struct {
	X, Y  int
	Cells [shieldHeight][shieldWidth]uint8
}

func newShields(width, height int) []Shield {
	if width != 40 || height != 18 {
		return nil
	}
	p := playerStart(width, height).X
	f := enemyStartX
	shields := make([]Shield, shieldCount)
	for i := range shields {
		left := p + ((i+1)*(f-p))/5 - shieldWidth/2
		if left < 0 {
			left = 0
		}
		if left+shieldWidth > width {
			left = width - shieldWidth
		}
		shields[i] = Shield{X: left, Y: height - playerHeight - shieldHeight}
		for y := range shields[i].Cells {
			for x := range shields[i].Cells[y] {
				shields[i].Cells[y][x] = shieldDurability
			}
		}
	}
	return shields
}

func (s Shield) Snapshot() ShieldSnapshot { return ShieldSnapshot{X: s.X, Y: s.Y, Cells: s.Cells} }

func (s *State) resetShields() { s.shields = newShields(s.arenaWidth, s.arenaHeight) }

func (s State) shieldCell(x, y int) (int, int, int, bool) {
	for i, shield := range s.shields {
		cx, cy := x-shield.X, y-shield.Y
		if cx >= 0 && cx < shieldWidth && cy >= 0 && cy < shieldHeight && shield.Cells[cy][cx] > 0 {
			return i, cx, cy, true
		}
	}
	return 0, 0, 0, false
}

func (s *State) damageShield(x, y, direction int) bool {
	best, bx, by := -1, 0, 0
	for i, shield := range s.shields {
		cx, cy := x-shield.X, y-shield.Y
		if cx < 0 || cx >= shieldWidth || cy < 0 || cy >= shieldHeight || shield.Cells[cy][cx] == 0 {
			continue
		}
		if best < 0 || (direction < 0 && (cy > by || cy == by && cx < bx)) || (direction > 0 && (cy < by || cy == by && cx < bx)) {
			best, bx, by = i, cx, cy
		}
	}
	if best < 0 {
		return false
	}
	s.shields[best].Cells[by][bx]--
	return true
}
