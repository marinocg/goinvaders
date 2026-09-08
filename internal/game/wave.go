package game

const (
	maxWavePressure          = 12
	minimumFireInterval Tick = 18
)

func wavePressure(wave int) int {
	if wave <= 1 {
		return 0
	}
	if wave-1 > maxWavePressure {
		return maxWavePressure
	}
	return wave - 1
}
func waveFireInterval(c DifficultyConfig, wave int) Tick {
	n := c.EnemyFireInterval - Tick(2*wavePressure(wave))
	if n < minimumFireInterval {
		return minimumFireInterval
	}
	return n
}
