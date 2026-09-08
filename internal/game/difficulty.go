package game

import "errors"

type Difficulty uint8

const (
	DifficultyEasy Difficulty = iota
	DifficultyNormal
	DifficultyHard
)

type DifficultyConfig struct {
	BaseMovementInterval Tick
	MovementFloor        Tick
	EnemyFireInterval    Tick
	MaxEnemyProjectiles  int
	EnemyProjectileSpeed int
}

var ErrInvalidDifficulty = errors.New("invalid difficulty")

func (d Difficulty) Config() (DifficultyConfig, error) {
	switch d {
	case DifficultyEasy:
		return DifficultyConfig{36, 5, 72, 1, 1}, nil
	case DifficultyNormal:
		return DifficultyConfig{30, 3, 54, 2, 1}, nil
	case DifficultyHard:
		return DifficultyConfig{24, 2, 42, 3, 2}, nil
	}
	return DifficultyConfig{}, ErrInvalidDifficulty
}
