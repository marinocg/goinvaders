package game

const (
	playerWidth  = 1
	playerHeight = 1
	playerStep   = 1
)

// Player is the player's top-left integer position and configured bounds.
type Player struct {
	X, Y          int
	Width, Height int
}

func playerStart(arenaWidth, arenaHeight int) Player {
	return Player{
		X:      (arenaWidth - playerWidth) / 2,
		Y:      arenaHeight - playerHeight,
		Width:  playerWidth,
		Height: playerHeight,
	}
}
