# Classic Gameplay Contract

This decision defines the deterministic rules for classic/endless arcade play. It
supersedes existing gameplay rules only where a rule below is explicit. All
quantities are integer simulation ticks, cells, or points; no elapsed wall-clock
time or random choice is used.

## Enemy acceleration

## Simulation Constants

The simulation runs at 60 ticks per second. A wave starts with 55 living
invaders (five rows of eleven). `living` is sampled after all removals in the
current tick. Formation horizontal movement uses a per-step interval:

```text
interval(living) = max(floor, base - 2 * floor((55 - living) / 5))
```

The formation moves one horizontal step whenever its movement counter reaches
the interval, then the counter resets to zero. The formation reverses direction
and moves down one row when its next horizontal step would cross the playfield
edge. The values for Normal are `base = 30` ticks and `floor = 3` ticks. Thus a
51-55 living invaders move every 30 ticks, 46-50 living invaders every 28
ticks, 6-10 living invaders every 12 ticks, and 1-5 living invaders every 10
ticks; with zero living invaders the formula still yields the 3-tick floor.
These bands follow directly from the formula and are inclusive.
The floor prevents a zero or negative interval and makes the final invaders
materially faster than the opening formation. Movement counters are preserved
only within a wave and reset for a new wave.

With Normal settings, the opening formation takes 30 ticks per step and the
last five invaders take 10 ticks per step. The latter is three times as often,
while the `floor` guarantees that no valid preset can produce a zero or
negative interval. The formula uses integer floor division, never wall-clock
elapsed time.

## Difficulty presets

The selected preset is part of the initial state and snapshot. Presets change
only the following named deterministic parameters; formation geometry, shield
rules, scoring, lives, and bonus-target rules are unchanged.

| Preset | `base` | `floor` | Enemy-fire interval | Max enemy projectiles | Projectile speed |
| --- | ---: | ---: | ---: | ---: | ---: |
| Easy | 36 | 5 | 72 ticks | 1 | 1 cell/tick |
| Normal | 30 | 3 | 54 ticks | 2 | 1 cell/tick |
| Hard | 24 | 2 | 42 ticks | 3 | 2 cells/tick |

The interval formula above uses the selected `base` and `floor`. Enemy-fire
interval is further adjusted by wave pressure below. Preset selection never
rolls a value or changes during a run.

## Enemy fire

Each wave has a fire counter starting at zero. It increments once in the enemy
fire phase; when it reaches the preset interval, it resets to zero and exactly
one shot is attempted. The fire-event index starts at zero
for each wave and increments after every attempt, including a blocked attempt.
Columns are inspected in circular order from left to right, starting at
`(wave number + fire-event index) modulo 11`. A column's firing lane is the
closed vertical strip from the selected invader's center x coordinate down to
the player loss line, with the projectile's one-cell width included. The lane
is clear exactly when no living invader in that column and no intact shield
cell intersects that strip below the selected invader. The shooter is the
lowest occupied-row living invader in the first inspected column whose lane is
clear. Blocked columns are skipped and inspection continues; if all living
columns are blocked, the attempt creates no projectile. This makes selection a
pure function of wave number, fire-event index, and formation/shield state.

An enemy projectile is created only if the current enemy-projectile count is
below the preset maximum. Enemy projectiles move downward at the preset speed.
There is no spread, random aim, or duplicate shot in one fire event.

## Shields

There are four shields. Let `P` be the player's starting center x, `F` the
formation's starting center x, and `S = 3` the shield width. Shield `i`
(zero-based) has left coordinate `P + floor(((i + 1) * (F - P)) / 5) -
floor(S / 2)`, using integer floor division. Its two rows are immediately
above the player's safe row and its columns are `left`, `left + 1`, and
`left + 2`. Clamp the complete 3-cell rectangle to the playfield if needed.
The formula and starting coordinates are identical on every wave and preset.
Each shield is a fixed 3-by-2 cell grid (six cells), with all
cells initially present at wave start. A cell has durability 2. A projectile
hit removes one durability; a cell is removed at durability zero. A projectile
is consumed by the hit, regardless of direction. Player and enemy projectiles
use identical cell collision geometry and damage, so either direction can
damage or destroy a shield. If a projectile overlaps multiple cells, the
nearest cell along its direction of travel is selected, with lower x then
lower y as the stable tie-break. A projectile cannot damage more than one cell
in one tick.
Shield cells persist through life loss but are restored to six full cells when
the next wave is installed. Shields do not block invaders; invader contact with
the shield area does not alter cells.

## Player, Collisions, and Outcomes

The player has three lives at run start. A player projectile has a maximum
count of one and travels upward at 2 cells/tick. A hit on an invader removes
that invader and awards 10 points. A bonus-target hit uses the rule below.

Within collision resolution, the precedence is: shield cells, invaders,
bonus target, then player. A projectile that hits a shield is consumed and
cannot hit a later object. A player hit by an enemy projectile or by an
invader reaching the player line loses one life; simultaneous causes still
remove only one life. On life loss, all projectiles are removed, the player is
reset to its starting position, and play resumes on the next tick. If lives
reach zero, the run enters `game_over` and no later wave or score change is
applied. A player hit does not restore invaders or shields.

## Wave progression

When the last invader is removed, the current tick completes its collision work
and sets `wave_advance_pending = true`; it does not install a formation in the
middle of that tick. At the beginning of the next tick, phase 0 checks this
flag. If it is true and the outcome is still `playing`, it increments the wave
number, installs exactly one fresh 55-invader formation, restores all shield
cells, clears projectiles and counters, clears the flag, and then proceeds with
the new wave's tick. The flag is cleared as part of installation, so a cleared
wave advances exactly once. A zero-enemy state therefore cannot repeatedly
schedule advancement without installation.

Score and remaining lives are preserved across waves. Normal classic play is
endless: clearing a wave advances to another wave and is not a final win.

Wave pressure is capped and deterministic. For wave `w >= 1`, define
`pressure = min(12, w - 1)`. The enemy-fire interval used for that wave is
`max(18, preset_fire_interval - 2 * pressure)` ticks. Formation movement still
uses the preset floor, so every wave remains valid and bounded.

## Bonus target and scoring

The bonus saucer follows a fixed schedule independent of input: it spawns at
tick 600 of each wave, then at 1,500, 2,400, and every 900 ticks thereafter,
with the first spawn
at the left edge moving right. It moves one cell per tick, reverses at either
edge, and despawns after 180 ticks or when hit. Each successful player-projectile
hit awards 100 points and consumes both projectile and saucer. At most one
saucer exists. A saucer does not collide with enemy projectiles, shields, or
the player; its schedule is based on the wave tick and is reset on wave
installation.

At most one saucer exists; a scheduled spawn is skipped if one is already
active. Its schedule uses integer wave ticks and resets on wave installation.

## Tick Contract

Each simulation tick executes these phases in order:

1. **Begin-wave/input:** apply the pending advancement as specified above, then
   read the player input for this tick and update the player position/fire
   request. A newly fired projectile is not moved until the next tick.
2. **Projectiles:** move existing projectiles by their fixed speeds and remove
   those outside the playfield.
3. **Shields:** resolve projectile-versus-shield collisions in projectile ID
   order; each resolved projectile is consumed. This applies to both projectile
   directions before any projectile can reach an invader or player.
4. **Enemy fire:** advance the fire counter and attempt fire when its interval
   is reached, subject to the shooter and count rules. A projectile created here
   is not moved or collided until the next tick.
5. **Collisions:** resolve remaining projectile collisions in ascending
   projectile ID, using shield, invader, saucer, player precedence. Apply each
   object removal and score immediately, but do not reorder already selected
   collisions.
6. **Saucer/formation movement:** advance the saucer and formation counters
   and perform due movement.
7. **Life outcome:** determine whether a loss-line breach occurred after the
   due formation movement and whether an enemy projectile hit the player. If
   the final invader was cleared in phase 5, discard the loss-line cause for
   this tick; an enemy-projectile hit remains a valid cause. Otherwise apply at
   most one life loss for all such causes, clear all projectiles, and reset the
   player position when a life remains. If the decrement makes lives zero,
   enter `game_over` immediately; this terminal transition is recorded before
   phase 8 and cannot be undone by wave completion. A loss-line breach and
   projectile hit in the same tick still consume only one life when the
   loss-line cause has not been suppressed.
8. **Wave completion:** if no invaders remain, inspect the result of phase 7.
   If phase 7 entered `game_over` (including the zero-lives case), game over
   takes precedence: do not set `wave_advance_pending`, do not advance, and do
   not award any later score. Otherwise set `wave_advance_pending = true`.
   Therefore final-invader clearance suppresses a same-tick loss-line breach
   only when at least one life remains; it never rescues a run that lost its
   final life. Installation is deliberately deferred to phase 0 of the next
   tick.

Identical initial state and input sequence produces identical IDs, positions,
collisions, scores, lives, wave numbers, and outcomes.

## Snapshot and UI Fields

The public snapshot exposes `wave_number` (1-based), `difficulty` (`easy`,
`normal`, or `hard`), `wave_tick`, `living_enemy_count`, `wave_advance_pending`,
`score`, `lives`, and `outcome` (`playing` or `game_over`). UI may display these
fields directly; it must not infer difficulty or wave state from timing,
entity count, or renderer state.
