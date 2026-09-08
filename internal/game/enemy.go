package game

import "errors"

const (
	formationRows, formationColumns      = 4, 8
	enemyWidth, enemyHeight              = 1, 1
	enemySpacingX, enemySpacingY         = 2, 1
	enemyStartX, enemyStartY             = 9, 0
	enemyStep, enemyDescentStep          = 1, 1
	enemyMovementBaseInterval       Tick = 30
	enemyMovementFloor              Tick = 3
	// Retained for callers that construct timing fixtures; live state uses the
	// cadence derived from its current living-enemy count.
	enemyMovementInterval Tick = 18
	enemyFireInterval     Tick = 54
	maxEnemyProjectiles        = 2
)

var (
	ErrInvalidEnemyFormation = errors.New("enemy formation does not fit arena")
	ErrInvalidEnemyMovement  = errors.New("enemy movement configuration is invalid")
)

type Enemy struct {
	X, Y, Width, Height int
	Alive               bool
}

func selectEnemyShooter(enemies []Enemy, event Tick) int {
	for offset := 0; offset < 11; offset++ {
		column := (int(event) + offset) % 11
		selected := -1
		for i, enemy := range enemies {
			enemyColumn := enemy.X / (enemyWidth + enemySpacingX)
			if enemy.X >= enemyStartX {
				enemyColumn = (enemy.X - enemyStartX) / (enemyWidth + enemySpacingX)
			}
			if enemy.Alive && enemyColumn == column && (selected < 0 || enemy.Y > enemies[selected].Y) {
				selected = i
			}
		}
		if selected >= 0 {
			return selected
		}
	}
	return -1
}

type EnemySnapshot struct {
	X, Y, Width, Height int
	Alive               bool
}

const enemyCount = formationRows * formationColumns

// Snapshot storage remains comparable for existing callers while allowing
// package-level tests and future formations to represent multiple enemies.
const enemySnapshotCapacity = 64

func newFormation(width, height int) ([]Enemy, error) {
	if width <= 0 || height <= 0 || formationRows <= 0 || formationColumns <= 0 || enemyWidth <= 0 || enemyHeight <= 0 || enemySpacingX < 0 || enemySpacingY < 0 || enemyStep <= 0 || enemyDescentStep <= 0 || enemyMovementBaseInterval <= 0 || enemyMovementFloor <= 0 {
		return nil, ErrInvalidEnemyMovement
	}
	if enemyStartX < 0 || enemyStartY < 0 || enemyStartX+(formationColumns-1)*(enemyWidth+enemySpacingX)+enemyWidth > width || enemyStartY+(formationRows-1)*(enemyHeight+enemySpacingY)+enemyHeight > height {
		return nil, ErrInvalidEnemyFormation
	}
	result := make([]Enemy, 0, formationRows*formationColumns)
	for row := 0; row < formationRows; row++ {
		for column := 0; column < formationColumns; column++ {
			result = append(result, Enemy{enemyStartX + column*(enemyWidth+enemySpacingX), enemyStartY + row*(enemyHeight+enemySpacingY), enemyWidth, enemyHeight, true})
		}
	}
	return result, nil
}

func currentEnemyMovementInterval(living int) Tick {
	return enemyMovementIntervalFor(living, DifficultyConfig{BaseMovementInterval: enemyMovementBaseInterval, MovementFloor: enemyMovementFloor})
}

func enemyMovementIntervalFor(living int, config DifficultyConfig, waves ...int) Tick {
	wave := 1
	if len(waves) > 0 {
		wave = waves[0]
	}
	if living <= 0 {
		return config.MovementFloor
	}
	interval := config.BaseMovementInterval - 2*Tick((enemyCount-living)/5) - Tick(wavePressure(wave))
	if interval < config.MovementFloor {
		return config.MovementFloor
	}
	return interval
}

func formationFromSlice(enemies []Enemy) []Enemy {
	return append([]Enemy(nil), enemies...)
}

func moveFormation(enemies *[]Enemy, arenaWidth, lossLine int, direction *int) bool {
	if direction == nil || (*direction != -1 && *direction != 1) {
		return false
	}
	minX, maxX := arenaWidth, -1
	for _, enemy := range *enemies {
		if enemy.Alive {
			if enemy.X < minX {
				minX = enemy.X
			}
			if enemy.X+enemy.Width > maxX {
				maxX = enemy.X + enemy.Width
			}
		}
	}
	if maxX >= 0 && maxX <= arenaWidth && ((*direction < 0 && minX-enemyStep < 0) || (*direction > 0 && maxX+enemyStep > arenaWidth)) {
		*direction = -*direction
		for i := range *enemies {
			if (*enemies)[i].Alive {
				(*enemies)[i].Y += enemyDescentStep
			}
		}
	} else {
		for i := range *enemies {
			if (*enemies)[i].Alive {
				(*enemies)[i].X += *direction * enemyStep
			}
		}
	}
	for _, enemy := range *enemies {
		if enemy.Alive && enemy.Y+enemy.Height >= lossLine {
			return true
		}
	}
	return false
}
