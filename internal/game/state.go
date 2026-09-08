package game

import "errors"

const (
	minArenaWidth      = 1
	minArenaHeight     = 1
	defaultArenaHeight = 1
)

var ErrInvalidArenaWidth = errors.New("arena width must be greater than zero")
var ErrInvalidArenaHeight = errors.New("arena height must be greater than zero")

type Tick int64

type Input struct {
	MoveX int
	Fire  bool
}

type State struct {
	tick               Tick
	arenaWidth         int
	arenaHeight        int
	playerX            int
	playerY            int
	playerProjectile   *Projectile
	bonusTarget        *BonusTarget
	enemies            []Enemy
	enemyDirection     int
	enemyElapsed       Tick
	enemyFireElapsed   Tick
	enemyFireEvent     Tick
	enemyProjectiles   []EnemyProjectile
	shields            []Shield
	enemyLossLine      bool
	score              int
	lives              int
	outcome            Outcome
	waveNumber         int
	difficulty         Difficulty
	waveAdvancePending bool
	waveTick           Tick
}

type Snapshot struct {
	Tick                 Tick
	ArenaWidth           int
	ArenaHeight          int
	Player               PlayerSnapshot
	PlayerProjectile     *ProjectileSnapshot
	BonusTarget          *BonusTargetSnapshot
	EnemyProjectiles     [enemySnapshotCapacity]EnemyProjectileSnapshot
	EnemyProjectileCount int
	Enemies              [enemySnapshotCapacity]EnemySnapshot
	Shields              [shieldCount]ShieldSnapshot
	ShieldCount          int
	EnemyLossLine        bool
	Score                int
	Lives                int
	Outcome              Outcome
	WaveNumber           int
	Difficulty           Difficulty
	WaveTick             Tick
	LivingEnemyCount     int
	WaveAdvancePending   bool
}

type PlayerSnapshot struct {
	X, Y          int
	Width, Height int
}

func NewState(arenaWidth int, heights ...int) (State, error) {
	return newState(arenaWidth, DifficultyNormal, heights...)
}

func NewStateWithDifficulty(arenaWidth int, difficulty Difficulty, heights ...int) (State, error) {
	if _, err := difficulty.Config(); err != nil {
		return State{}, err
	}
	return newState(arenaWidth, difficulty, heights...)
}

func newState(arenaWidth int, difficulty Difficulty, heights ...int) (State, error) {
	if arenaWidth < minArenaWidth {
		return State{}, ErrInvalidArenaWidth
	}
	arenaHeight := defaultArenaHeight
	if len(heights) > 1 || (len(heights) == 1 && heights[0] < minArenaHeight) {
		return State{}, ErrInvalidArenaHeight
	}
	if len(heights) == 1 {
		arenaHeight = heights[0]
	}

	formation, err := newFormation(arenaWidth, arenaHeight)
	if err != nil {
		// Small arenas are useful for focused mechanics tests and legacy callers;
		// the configured startup arena always uses the full formation.
		if err != ErrInvalidEnemyFormation {
			return State{}, err
		}
		formationX := enemyStartX
		if arenaWidth <= 4 {
			formationX = 0
		}
		formationY := 0
		if arenaWidth == 1 {
			formationY = arenaHeight - 2
		}
		formation = []Enemy{{X: formationX, Y: formationY, Width: enemyWidth, Height: enemyHeight, Alive: true}}
	}
	return State{
		tick:        0,
		arenaWidth:  arenaWidth,
		arenaHeight: arenaHeight,
		playerX:     playerStart(arenaWidth, arenaHeight).X,
		playerY:     playerStart(arenaWidth, arenaHeight).Y,
		enemies:     formation, enemyDirection: 1,
		shields:    newShields(arenaWidth, arenaHeight),
		lives:      3,
		waveNumber: 1, difficulty: difficulty,
	}, nil
}

func Step(state State, input Input) State {
	if !validState(state) || !validInput(input) || state.outcome.Terminal() {
		return state
	}
	if state.waveAdvancePending {
		state.waveNumber++
		if !state.installWave() {
			return state
		}
	}
	// State transitions are value-based, so do not mutate the caller's slice.
	state.enemies = append([]Enemy(nil), state.enemies...)
	state.shields = append([]Shield(nil), state.shields...)

	state.tick = nextTick(state.tick)
	state.waveTick = nextTick(state.waveTick)
	// A kill can shorten the cadence, but it must not make the same elapsed
	// tick newly due. Account for this tick against the pre-collision cadence.
	config, _ := state.difficulty.Config()
	movementInterval := enemyMovementIntervalFor(state.enemiesAlive(), config, state.waveNumber)
	state.playerX = clamp(state.playerX+input.MoveX*playerStep, 0, state.arenaWidth-playerWidth)
	if input.Fire && state.playerProjectile == nil && state.playerY > 0 {
		state.playerProjectile = NewPlayerProjectile(state.playerX+playerWidth/2, state.playerY-1)
	}
	var collision CollisionResult
	state, collision = ResolveProjectileCollisions(state)
	state.score += collision.ScoreDelta
	state.advanceBonusTarget()
	if state.enemiesAlive() == 0 {
		// Wave completion is a transition, not a terminal outcome. Installing
		// the next wave on the following step makes the transition observable.
		state.waveAdvancePending = true
		return state
	}
	state.advanceEnemyProjectiles()
	state.enemyFireElapsed++
	if state.enemyFireElapsed >= waveFireInterval(config, state.waveNumber) && len(state.enemyProjectiles) < config.MaxEnemyProjectiles {
		state.enemyFireElapsed = 0
		if shooter := selectEnemyShooter(state.enemies, state.enemyFireEvent); shooter >= 0 {
			enemy := state.enemies[shooter]
			state.enemyProjectiles = append(state.enemyProjectiles, EnemyProjectile{X: enemy.X + enemy.Width/2, Y: enemy.Y + enemy.Height})
		}
		state.enemyFireEvent++
	} else if state.enemyFireElapsed >= enemyFireInterval {
		state.enemyFireElapsed = 0
		state.enemyFireEvent++
	}
	state.enemyElapsed++
	for interval := movementInterval; state.enemyElapsed >= interval; interval = enemyMovementIntervalFor(state.enemiesAlive(), config, state.waveNumber) {
		state.enemyElapsed -= interval
		if moveFormation(&state.enemies, state.arenaWidth, state.playerY, &state.enemyDirection) {
			state.enemyLossLine = true
		}
	}
	if state.enemyLossLine && state.arenaHeight > 2 {
		state.loseLife()
	}
	if state.outcome == OutcomePlaying && state.enemyProjectileHit() {
		state.loseLife()
	}

	return state
}

func (s *State) loseLife() {
	s.lives--
	s.playerProjectile = nil
	s.bonusTarget = nil
	s.enemyProjectiles = nil
	if s.lives == 0 {
		s.outcome = OutcomeGameOver
		return
	}
	s.respawn()
}

func StepMany(state State, inputs []Input) State {
	for _, input := range inputs {
		state = Step(state, input)
	}

	return state
}

func (s State) Snapshot() Snapshot {
	result := Snapshot{
		Tick:        s.tick,
		ArenaWidth:  s.arenaWidth,
		ArenaHeight: s.arenaHeight,
		Player: PlayerSnapshot{
			X:      s.playerX,
			Y:      s.playerY,
			Width:  playerWidth,
			Height: playerHeight,
		},
	}
	if s.playerProjectile != nil {
		projectile := s.playerProjectile.Snapshot()
		result.PlayerProjectile = &projectile
	}
	if s.bonusTarget != nil {
		target := s.bonusTarget.Snapshot()
		result.BonusTarget = &target
	}
	result.EnemyProjectileCount = len(s.enemyProjectiles)
	for i, projectile := range s.enemyProjectiles {
		result.EnemyProjectiles[i] = projectile.Snapshot()
	}
	for i, enemy := range s.enemies {
		result.Enemies[i] = EnemySnapshot{enemy.X, enemy.Y, enemy.Width, enemy.Height, enemy.Alive}
	}
	result.ShieldCount = len(s.shields)
	for i, shield := range s.shields {
		result.Shields[i] = shield.Snapshot()
	}
	result.EnemyLossLine = s.enemyLossLine
	result.Score, result.Lives, result.Outcome = s.score, s.lives, s.outcome
	result.WaveNumber, result.Difficulty, result.WaveTick = s.waveNumber, s.difficulty, s.waveTick
	result.LivingEnemyCount, result.WaveAdvancePending = s.enemiesAlive(), s.waveAdvancePending
	return result
}

func (s State) enemiesAlive() int {
	count := 0
	for _, enemy := range s.enemies {
		if enemy.Alive {
			count++
		}
	}
	return count
}

func (s *State) respawn() {
	s.playerX = playerStart(s.arenaWidth, s.arenaHeight).X
	s.playerY = playerStart(s.arenaWidth, s.arenaHeight).Y
	s.playerProjectile = nil
	s.bonusTarget = nil
	s.enemyProjectiles = nil
	s.enemyFireElapsed = 0
	s.enemyFireEvent = 0
	s.enemyDirection = 1
	s.enemyElapsed = 0
	s.enemyLossLine = false
	formation, err := newFormation(s.arenaWidth, s.arenaHeight)
	if err == nil {
		for i := range s.enemies {
			if s.enemies[i].Alive && i < len(formation) {
				s.enemies[i].X = formation[i].X
				s.enemies[i].Y = formation[i].Y
			}
		}
		return
	}
	for i := range s.enemies {
		if s.enemies[i].Alive {
			s.enemies[i].X = enemyStartX
			if s.arenaWidth <= 1 {
				s.enemies[i].X = 0
				s.enemies[i].Y = enemyStartY
				continue
			}
			s.enemies[i].Y = enemyStartY
		}
	}
}

// installWave replaces the formation for a newly started wave.  Unlike
// respawn, a wave transition restores defensive material while retaining the
// run's score and remaining lives.
func (s *State) installWave() bool {
	formation, err := newFormation(s.arenaWidth, s.arenaHeight)
	if err != nil {
		if err != ErrInvalidEnemyFormation {
			return false
		}
		formation = []Enemy{{X: 0, Y: 0, Width: enemyWidth, Height: enemyHeight, Alive: true}}
	}
	s.enemies = formation
	s.resetShields()
	s.playerProjectile = nil
	s.bonusTarget = nil
	s.enemyProjectiles = nil
	s.enemyDirection = 1
	s.enemyElapsed = 0
	s.enemyFireElapsed = 0
	s.enemyFireEvent = 0
	s.enemyLossLine = false
	s.waveTick = 0
	s.outcome = OutcomePlaying
	s.waveAdvancePending = false
	return true
}

func validState(state State) bool {
	if _, err := state.difficulty.Config(); err != nil {
		return false
	}
	if state.arenaWidth >= minArenaWidth &&
		state.arenaHeight >= minArenaHeight &&
		state.tick >= 0 &&
		state.waveNumber >= 1 &&
		state.playerX >= 0 && state.playerX+playerWidth <= state.arenaWidth &&
		state.playerY >= 0 && state.playerY+playerHeight <= state.arenaHeight {
		return validEntities(state)
	}
	return false
}

func validEntities(state State) bool {
	if len(state.shields) > shieldCount {
		return false
	}
	for _, shield := range state.shields {
		if shield.X < 0 || shield.Y < 0 || shield.X+shieldWidth > state.arenaWidth || shield.Y+shieldHeight > state.arenaHeight {
			return false
		}
		for y := range shield.Cells {
			for x := range shield.Cells[y] {
				if shield.Cells[y][x] > shieldDurability {
					return false
				}
			}
		}
	}
	if state.playerProjectile != nil {
		p := state.playerProjectile
		if p.X < 0 || p.X >= state.arenaWidth || p.Y < 0 || p.Y >= state.arenaHeight {
			return false
		}
	}
	if state.bonusTarget != nil {
		t := state.bonusTarget
		if t.Width <= 0 || t.Height <= 0 || t.X < 0 || t.Y < 0 || t.X+t.Width > state.arenaWidth || t.Y+t.Height > state.arenaHeight {
			return false
		}
	}
	for _, p := range state.enemyProjectiles {
		if p.X < 0 || p.X >= state.arenaWidth || p.Y < 0 || p.Y >= state.arenaHeight {
			return false
		}
	}
	for _, enemy := range state.enemies {
		if enemy.Width <= 0 || enemy.Height <= 0 || enemy.X < 0 || enemy.Y < 0 {
			return false
		}
		if enemy.Alive && rectanglesOverlap(
			state.playerX, state.playerY, playerWidth, playerHeight,
			enemy.X, enemy.Y, enemy.Width, enemy.Height,
		) {
			return false
		}
	}
	return true
}

func rectanglesOverlap(ax, ay, aw, ah, bx, by, bw, bh int) bool {
	return ax <= bx+bw-1 && bx <= ax+aw-1 && ay <= by+bh-1 && by <= ay+ah-1
}

func validInput(input Input) bool {
	return input.MoveX >= -playerStep && input.MoveX <= playerStep
}

func nextTick(tick Tick) Tick {
	if tick == Tick(^uint64(0)>>1) {
		return tick
	}

	return tick + 1
}

func clamp(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}

	return value
}
