# Gameplay Rules Decision

## Status

Accepted.

## Context

The core architecture uses deterministic, discrete tick updates in
`internal/game`, with ordered logical input events supplied by the loop. This
document defines the initial classic Space Invaders rules so gameplay
implementation and tests do not need to invent behavior. It does not define
package structure, terminal rendering, or keyboard bindings.

## Coordinate and tick conventions

- The playfield is a fixed logical rectangle with origin `(0, 0)` at its top-
  left. Positions and dimensions are integer cells (or the equivalent integer
  world units); the concrete rendering size is an integration choice.
- Entity bounds are closed for collision purposes: touching edges counts as a
  collision. A projectile occupies one cell-wide vertical segment for
  collision checks.
- One update call applies exactly one simulation tick, as required by the
  architecture decision. Rules are evaluated in the order below; no wall-clock
  time or random value affects the result.
- A tick delta of zero has no gameplay effect. Negative tick deltas are invalid
  and must be rejected without changing state.

## Player movement and shooting

- The player starts centered horizontally in the bottom safe row, with its
  entire bounding box inside the playfield. The exact sprite dimensions are a
  game-state constant, not a renderer decision.
- A move-left or move-right intent moves the player by one configured player
  step for that tick. Multiple ordered intents in one tick are applied in
  order. Movement is clamped to the playfield; it never wraps or leaves the
  bounds.
- Fire intent creates a player projectile only when no player projectile is
  already active. Additional fire intents while one is active are ignored.
- A projectile is spawned at the player's horizontal center and immediately
  above its top edge. It travels upward at one configured projectile step per
  tick. A shot that reaches or crosses the top boundary is removed after its
  movement and cannot hit anything outside the playfield.
- Player input is ignored after win or game over. A quit event is an
  orchestration concern and does not alter gameplay state.

## Enemy formation and movement

- Each wave contains one formation with a fixed number of rows and columns,
  fixed spacing, and a fixed starting position. The formation is ordered by
  row then column; this stable order is used for deterministic tie-breaking.
- Enemies move horizontally as a formation. Each movement step advances all
  living enemies by one configured enemy step in the current direction.
- Before a horizontal step, if that step would put any living enemy beyond
  either side boundary, the formation does not take that horizontal step.
  Instead, its direction reverses and the formation descends by one configured
  descent step. The next enemy movement step is horizontal in the new
  direction. The formation never wraps.
- The loss line is the top edge of the player's row: a formation reaches the
  loss line when the bottom edge of any living enemy is at or below that line.
  This is a gameplay boundary, not a rendering choice.
- Enemy movement is scheduled at a deterministic tick interval. The interval
  and steps are game-state constants/configuration, and an elapsed tick count
  is accumulated with integer arithmetic. If a tick crosses a movement
  interval, apply each due movement step in sequence; do not combine them into
  one larger move.
- If any living enemy reaches the loss line, resolve one loss-line transition
  after the due formation movement. The transition atomically decrements
  lives by one. If lives remains greater than zero, the player is respawned at
  its starting position and the living enemies are reset to their original
  wave formation positions, preserving destroyed enemies; formation direction
  and movement schedule are also reset to their wave-start values. If the
  decrement makes lives zero, enter game over instead and do not respawn or
  reset the formation. Thus the third loss-line breach consumes the final life
  and enters game over; reaching the loss line never causes an earlier,
  separate immediate game-over transition.
- Enemies do not fire in the initial ruleset. No random enemy behavior is
  permitted.

## Projectiles and collisions

Each tick is processed deterministically as follows:

1. Apply ordered input events, including movement and fire, in event order.
2. Move active projectiles one step and remove projectiles that leave the
   playfield.
3. Resolve projectile collisions. For each projectile, select at most one
   target: the nearest living enemy along its path; ties use formation order.
   Remove both projectile and target, and award the target's configured point
   value. A projectile cannot pass through an enemy or hit more than one.
4. If the player projectile is destroyed by an enemy collision, it cannot be
   recreated by a fire event already processed in that tick.
5. If no enemies remain, enter win immediately. Win takes precedence over
   enemy descent or loss-line checks in the same tick.
6. Advance the enemy movement schedule and apply all due formation steps.
7. If the formation reaches the loss line, apply the single loss-line
   transition defined above, unless the wave was already completed and the game
   entered win in step 5.

An entity removed earlier in a tick cannot participate in later collision or
boundary checks. A projectile that intersects multiple enemies resolves only
the nearest target. Player and enemy entities cannot occupy the same resolved
state through collision handling: if an implementation detects an invalid
overlap not caused by the defined loss-line rule, it must reject the update
without partially applying that tick.

## Score lives and terminal states

- The player starts with three lives and a score of zero.
- Destroying an enemy increases the score by that enemy's fixed point value;
  missed shots and movement award no points.
- The initial game has one wave. Destroying its last enemy is the win
  condition. There is no automatic next wave in this ruleset.
- Losing a life is triggered when the formation reaches the loss line. The
  life decrement, and, when lives remain, respawn and formation reset are one
  atomic transition; score and destroyed enemies are preserved. When the
  decrement reaches zero lives, the same transition enters game over without
  respawning or resetting. A game-over state has no further simulation or
  score changes.
- Win and game over are terminal and mutually exclusive. The first terminal
  transition wins, except that completing the wave during collision resolution
  explicitly takes precedence over a same-tick loss-line check.
- Invalid configuration (for example, non-positive dimensions or movement
  intervals) must be rejected before play begins; it must not be silently
  clamped or defaulted. Invalid input events are ignored as no-op events.

## Explicit non-decisions

Keyboard-to-command mappings, key repeat policy, terminal drawing, sprite
appearance, and the concrete tick rate remain governed by the architecture and
integration decisions rather than this document.
