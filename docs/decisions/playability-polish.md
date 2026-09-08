# Playability and Terminal Presentation Decision

## Status

Accepted.

## Context

The game uses deterministic integer simulation ticks, but the current startup
configuration creates a one-cell-high arena and a one-enemy formation. The
terminal flow also redraws while waiting for start, pause, or a terminal
outcome. In a real terminal those choices present as start flicker, make Enter
appear ineffective, and make the game look like a debug screen rather than a
playable game.

This decision supplies concrete defaults and presentation rules for the
existing mechanics. It does not add mechanics, graphical UI, audio, networking,
or a new timing model.

## Playable runtime defaults

The default logical arena is **40 columns by 18 rows**, with integer
coordinates beginning at `(0, 0)` in the top-left. These are logical gameplay
cells, not terminal rows and columns. The fixed size keeps movement, projectile
travel, formation descent, and the loss line stable when the terminal resizes.

- The player occupies one cell and starts centered at `(19, 17)`.
- A player shot occupies one cell, starts immediately above the player, and
  moves upward one cell per simulation tick. It therefore has 16 logical rows
  of playable travel before leaving the arena.
- The loss line is the player's row, `y = 17`, as defined by the gameplay
  rules. An enemy reaching that line triggers the existing life-loss behavior.
- Terminal resizing changes presentation only. Collision, movement, projectile
  travel, and loss-line checks always use the complete 40-by-18 logical arena.

## Initial wave and cadence

The initial wave is a recognizable **4-row by 8-column** formation (32 living
enemies), ordered row-major. Enemies are one cell wide and one cell high, with
two empty logical cells between columns and one empty cell between rows. The
formation starts at `(9, 0)`, occupying `x = 9..30` and `y = 0, 2, 4, 6`.

- Every movement step moves all living enemies one logical cell horizontally.
- At a horizontal boundary, the formation reverses direction and descends one
  logical cell; the next due step is horizontal in the new direction.
- One movement step is due every **30 simulation ticks**. At the normal 60 Hz
  application tick this is one horizontal cell every 500 ms, or 2 cells per
  second. A 22-cell traverse therefore takes about 11 seconds, making movement
  and edge descent human-readable rather than a rapid sweep. The elapsed
  counter is an integer tick counter and may apply multiple due steps when a
  caller supplies multiple ticks. No fractional or wall-clock progress is
  stored.
- The normal application tick remains **1/60 second**; only the movement
  interval changes. Gameplay remains deterministic at the game-state level.
- Existing player movement and one-active-projectile rules remain unchanged.
  No enemy firing, randomness, automatic waves, or other gameplay system is
  introduced.

## Input and idle pacing

Input remains nonblocking and is drained in arrival order. The UI flow owns
state-specific pacing:

- **Start, paused, win, and game-over:** render once on entry, then wait on a
  cancellable **50 ms** poll deadline. Drain input and check terminal size at
  each deadline. Render only after a meaningful input/state transition or a
  size transition. Do not request simulation ticks in these states; with no
  change, do not repaint continuously.
- **Active play:** use the existing fixed 60 Hz tick source, drain input once
  per tick boundary, and render after a visible state-changing tick. A frame
  byte-for-byte identical to the last successfully rendered frame may be
  suppressed.
- **Resize:** check size at active render boundaries and at the 50 ms idle poll.
  A resize does not advance gameplay. It causes a render only when dimensions
  or too-small status changes.
- **Cancellation, quit, and I/O errors:** stop promptly, restore terminal
  settings and cursor visibility, and return the triggering error. Every wait
  is cancellable through the existing context.

Enter is accepted in start, win, and game-over states and causes one visible
transition. It is not a gameplay command. Pause toggles only active and paused
play and does not add a simulation tick on resume.

## Rendering and flicker policy

Frames are assembled off-screen as fixed-width ASCII/ANSI text. The renderer
must minimize visible full-screen clears:

- Full clear plus cursor-home (`ESC [2J ESC [H`) is allowed only for the first
  frame, a transition between normal and too-small presentation, or a resize
  that changes frame dimensions.
- Same-sized normal frames home the cursor and overwrite the complete frame
  (`ESC [H`), including spaces that erase old content. Identical frames emit
  nothing. The renderer must not rely on scrollback or leave stale HUD/status
  characters.
- Hide the cursor while the renderer owns the screen. On shutdown and on any
  render, write, or size error, attempt to show the cursor and reset ANSI
  attributes; return the triggering error even if cleanup also fails.
- A too-small terminal shows one stable ASCII message,
  `Terminal too small (required: 40 x 18)`, and suspends gameplay rendering
  and simulation until sufficient size is observed. The message is centered
  when its complete text fits. If the measured width is narrower than the
  message, render the longest prefix that fits the viewport, centered in that
  width; do not wrap it, append an ellipsis, or write past either edge. Thus a
  one-column terminal renders only `T`, a two-column terminal renders `Te`,
  and so on. Place the message on the vertically centered available row when
  there is one; for a zero-row or zero-column measurement, emit no characters
  and wait for the next size check. This clipped form is the sole fallback and
  remains stable until the dimensions or sufficient/too-small status changes.
  No ANSI styling or cursor movement may address a column or row outside the
  measured viewport. It is not repeatedly cleared or repainted.
- Returning to a sufficient terminal, or changing sufficient dimensions, gets
  one full clear and complete frame. Layout code must never write outside the
  currently measured viewport, including during rapid resize.

The existing minimum remains **40 columns by 18 rows**, measured as the whole
viewport. At that minimum the layout is explicitly non-overlapping:

```
rows 1-11   bordered playfield, 40 columns total and 38 interior columns
row 12      blank separator
row 13      `INVADERS` title, centered
row 14      `Score: <n>   Lives: <n>` HUD
row 15      compact controls: `A/D move|Space/F fire|P pause|Q quit`
row 16      state status, including `Press Enter to start`
rows 17-18  blank padding
```

The controls string is 36 columns wide including its separators, so it fits
the 40-column minimum with four columns of spare margin. It is the canonical
single-line form at the minimum size; do not substitute the longer prose form
or allow controls to wrap, clip, or overwrite the border. At larger widths the
same string may be centered, but its wording and ordering remain stable.

The playfield projection maps logical `(x, y)` to interior display cell
`(floor(x * 38 / 40), floor(y * 9 / 18))`, offset by one column and one row
for the border. This is presentation scaling only. If multiple logical cells
map to one display cell, resolve overlap in priority order: player `A`,
projectile `|`, enemy `W`, blank. Larger sufficient terminals may provide more
playfield rows and margins, but must preserve the ordering and never change
the logical arena.

## Terminal viewport fit and autowrap safety

`Size` reports raw terminal dimensions. The usable render viewport is
`max(width-1, 0)` by `max(height-1, 0)`: the final raw column and row are
reserved so printable content never depends on an autowrap-sensitive edge or
bottom-right cell. The required safe layout is therefore 39 by 17 usable
cells, corresponding to the supported raw minimum of 40 by 18.

Too-small detection uses this usable viewport rather than raw dimensions. The
playfield border, title, HUD, controls, status, and fallback message are all
bounded by it and remain single-line. On resize, layout is recomputed and a
full clear removes stale edge content; unchanged frames remain suppressed and
same-sized changes still use cursor-home overwrite instead of unconditional
full-screen clears.

## Centered viewport geometry and projection stability

The renderer treats the safe area as a positioned rectangle inside the raw
terminal. Its origin, width, and height are computed together from the
autowrap-safe dimensions, with spare cells split symmetrically and an odd
remainder retained consistently on the right and bottom. The playfield, title,
HUD, controls, and status all use that same rectangle as their coordinate
basis.

Logical arena coordinates are projected with monotonic nearest-cell integer
interpolation between the rectangle's interior playfield extrema. Deterministic
nearest-cell rounding avoids the persistent left/top bias of floor division. Thus
logical `(0, 0)` and
`(39, 17)` always reach the corresponding left/top and right/bottom interior
cells, while resize recalculates the rectangle from scratch without cumulative
drift or stale edge content. This preserves the final-row and final-column
autowrap reservation and the existing unchanged-frame and resize-clear redraw
policy.

The safe viewport is a **rectangle inside the raw terminal**, not a replacement
for the raw terminal coordinate system anchored unconditionally at `(0, 0)`.
Layout must therefore carry an explicit origin `(originX, originY)` together
with content width and height.

- Position the usable content rectangle within the raw terminal with margins
  distributed as evenly as integer cells allow. When an odd spare cell makes
  perfect symmetry impossible, keep the extra cell on the right or bottom so
  the visible playfield does not jump left/up between adjacent sizes.
- Keep the title, playfield, HUD, controls, and status centered relative to the
  same content rectangle. Do not center some elements against raw width and
  others against usable width.
- Every presentation row is relative to that rectangle, including the HUD,
  controls, and status rows; none may fall back to raw-terminal coordinates.
- Preserve a stable playfield aspect/projection policy across resize. Extra
  terminal width or height becomes margin or deliberate playfield growth; it
  must not shear the logical 40-by-18 arena or independently stretch one axis
  in a way that makes the formation/player geometry look skewed.
- Logical left/right and top/bottom extrema must map to the corresponding
  interior display extrema. Projection must remain monotonic on both axes and
  preserve relative ordering of enemies, projectile, and player.
- Resize may change integer rounding by at most the expected one-cell amount;
  it must not cause a persistent visual drift toward one edge.
- The 40-by-18 raw minimum remains supported. The renderer may use a smaller
  safe content rectangle inside it, but that rectangle must be positioned
  deliberately and all visual blocks must share its origin and dimensions.
- The implementation derives that rectangle from the safe dimensions in one
  step: its origin is the integer half of the spare width and height, with any
  odd remainder retained on the right and bottom. Projection uses inclusive
  logical extrema for both axes, so `(0, 0)` and `(39, 17)` reach opposite
  interior playfield cells without cumulative resize drift.

## Visual hierarchy and screen states

The presentation is sparse, high-contrast, and functional without ANSI color.
ANSI bold, color, and inverse video are optional enhancements.

- **Title/start:** center `INVADERS` on its own title row, keep controls on
  their own row, and make `Press Enter to start` the prominent status. This
  separation makes acknowledgement visible rather than appearing ineffective.
- **Playfield:** use a single-line ASCII border. Keep HUD and status outside it.
  Use stable glyphs: player `A`, enemy `W`, projectile `|`.
- **Active entities:** player and projectile may be bold; enemies use a
  consistent secondary style. Glyph meaning must remain legible as plain ASCII.
- **HUD:** place `Score: <n>   Lives: <n>` directly below the title/arena block
  at the fixed minimum row and regenerate it from the snapshot.
- **Controls/status:** use the compact controls line above. State status takes
  precedence over controls only if a larger layout cannot fit both; at 40x18
  both fit on their separate rows. The status line may be shorter than the
  controls line, but neither line may wrap at the minimum.
- **Paused:** retain the last playfield and HUD; show `Paused - Press P to
  resume`. Do not dim, clear, or animate.
- **Win:** retain the final playfield and score; show `You win - Press Enter to
  play again`, while keeping quit available.
- **Game over:** retain the final playfield and score; show `Game over - Press
  Enter to try again`, while keeping quit available.

All non-active status lines remain stable between input events. They must not
blink, scroll, or be replaced by a blank frame.

## Ownership and consequences

## Full-frame vertical fit and serialization safety

The renderer treats the complete presentation as one bounded rectangle. The
title, bordered playfield, HUD, controls, status, separators, and safety
margins are reserved together before the remaining rows are assigned to the
playfield. If a supported terminal is taller than the minimum, the playfield
grows; if it is constrained, only the playfield projection shrinks. Required
information rows are never clipped, and terminals below the supported minimum
use the existing too-small state.

Frame serialization uses exactly the safe viewport, `width-1` columns by
`height-1` rows, rather than padding through the raw terminal's final column
or row. Each row transition is serialized as carriage return plus line feed,
and the final safe row has no transition, so no printable cell is written in
the autowrap-sensitive final column and no row transition depends on implicit
wrapping. The resulting frame has a
provable top and bottom within the visible viewport and is idempotent across
repeated renders.

Layout and serialization are recalculated from the current terminal size on
every render. This preserves centered responsive geometry while preventing
resize sequences from accumulating cursor movement, scroll, or origin drift.

## Responsive viewport scaling and centering

The fixed-size content behavior described above is superseded where it would
leave available terminal space unused. Once the raw terminal is at least the
supported 40 by 18 size, the content rectangle is derived from the complete
autowrap-safe viewport: `max(width-1, 0)` by `max(height-1, 0)`. It retains the
minimum 39 by 17 content size, and when either safe dimension is larger than
that minimum it uses two fewer cells in that dimension. The resulting spare
cells are split with integer half-margins, keeping the extra cell on the right
or bottom. Thus the presentation grows substantially while remaining centered,
and the reserved final column and row remain untouched.

The bordered playfield occupies the content rectangle's top section and the
title, HUD, controls, and status occupy its compact seven-row information
area. Consequently, additional height is assigned primarily to the playfield,
and additional width expands both the border and its projected logical arena.
All information rows remain centered against the same content rectangle.

The content rectangle and playfield projection are recomputed from raw terminal
dimensions after every resize. Logical gameplay remains fixed at 40 by 18;
only its monotonic presentation projection changes. The first and last logical
coordinates map to the corresponding interior extrema, stable row-major entity
ordering is retained, and frame suppression, resize clearing, cursor cleanup,
too-small behavior, and the final-row/final-column safety reservation remain
unchanged. At the 40 by 18 minimum this produces the existing readable layout;
larger terminals no longer cap the game at 39 by 17.

`internal/game` owns logical dimensions, formation constants, movement interval,
and integer-tick behavior. `cmd/invaders` supplies the default arena and 60 Hz
tick source. `internal/ui` owns state-specific polling and render triggers.
`internal/render` owns layout, frame buffering, glyphs, ANSI cursor/clear policy,
unchanged-frame suppression, and too-small transitions. `internal/input` owns
nonblocking key decoding and raw-mode lifecycle.

Later implementation work must preserve the 40x18 minimum, these defaults, and
integer-tick semantics. It must not add gameplay systems or modify `autodev/**`.
