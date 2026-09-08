# Terminal Visual Language Decision

## Status

Accepted.

## Context

`W`, `A`, and `|` are functional, but read as debug symbols when the playfield
gets large. This decision defines a stable ASCII-first presentation language
for later renderer work. It changes neither logical coordinates, collisions,
formation dimensions, movement, nor hitboxes.

## Responsive detail tiers

Tier selection uses the autowrap-safe rectangle, not the raw terminal. The
safe rectangle is at most `width-1` columns by `height-1` rows, reserving the
raw final column and row. A tier is selected only when its complete frame,
including labels, border, margins, and sprite gaps, fits. Smaller terminals
use the existing deterministic `Terminal too small (requires 40 x 18)` screen.

| Tier | Safe rectangle requirement | Playfield presentation | Sprites |
| --- | --- | --- | --- |
| Compact | `39 x 17` (raw minimum `40 x 18`) through medium threshold | `39 x 8`, bordered; the logical `40 x 18` arena is scaled into its `37 x 6` interior | 1x1 |
| Medium | at least `60 x 31` | At least `58 x 25`, bordered; its `56 x 23` interior accommodates the full detailed formation | enemy/player 3x2, projectile 1x2 |
| Large | at least `84 x 40` | At least `82 x 33`, bordered; its `80 x 31` interior accommodates the full rich formation | enemy/player 5x3, projectile 1x3 |

The compact mapping is intentional: an 18-row logical arena is not an
18-terminal-row display arena. At the minimum, the complete frame is exactly
17 safe rows: title, HUD, eight playfield rows (top border, six interior rows,
bottom border), controls, status, and four one-row separators/margins. The
logical arena's extrema map to the presentation interior extrema; multiple
logical rows or columns may share a display cell. Entity ordering is retained,
and the renderer resolves a shared cell deterministically without changing
game state. This is the only compact scaling exception; medium and large use
the largest projection that fits. A terminal between `40 x 18` and the medium
threshold therefore remains compact; detailed sprites are enabled only when
the entire 4-column by 8-row formation and its required gaps fit.

All rows are bounded before serialization. No string wraps, no implicit
terminal wrapping is used, and CRLF separates rows with no transition after
the final safe row.

## Entity sprites

## Classic gameplay entity assets

Enemy shots use `v` compact, a two-row `\\/` and `/\\` dart medium, and a
three-row ` /\\  `, `  v  `, ` \\/  ` dart large, keeping them distinct from the
player's `|` shot without color. Shields render only positive-durability cells:
`.`/`#` compact, `+`/`#` at richer tiers for intact/damaged material; zero cells
are transparent. Durability `1` is damaged and durability `2` is intact.
The bonus target uses `B`, `-B-` over `---`, and a three-row saucer silhouette
for compact, medium, and large tiers. All assets are ASCII and anchored to the
snapshot point; selection downgrades at borders. Shields are composited before
actors and projectiles so gameplay sprites have deterministic precedence.

Matrices contain printable ASCII and spaces; spaces are transparent. The
logical entity point is projected first. It is the bottom-center cell for odd
width sprites (the sole cell for compact); the projectile anchor is its bottom
cell. Matrix placement around that anchor never changes logical position,
collision, or hitbox. A complete matrix must fit inside the border interior.
If it cannot, the renderer first selects the next smaller tier, then clips
only as a final edge-safe fallback; it never moves the logical entity.

### Compact (1x1)

```text
player:     A
enemy:      W
projectile: |
```

### Medium (3x2)

Anchor is column 2, row 2 for player/enemy and column 1, row 2 for projectile.
Quoted rows specify exact widths.

```text
player:     " ^ "
            "/A\\"
enemy:      "/W\\"
            "\\_/"
projectile: "|"
            "|"
```

### Large (5x3)

Anchor is column 3, row 3 for player/enemy and column 1, row 3 for projectile.

```text
player:     " /A\\ "
            "<|||>"
            " /A\\ "
enemy:      " /W\\ "
            "<W_W>"
            " \\_/ "
projectile: "|"
            "|"
            "|"
```

These ASCII silhouettes are wider/taller than one cell where space permits and
remain distinguishable without color. ASCII punctuation is preferred over
Unicode whose terminal width is inconsistent.

## Formation spacing and projection

Logical entity coordinates and hitboxes are authoritative and unchanged. A
medium or large projection may increase pitch while preserving formation
ordering and mapping logical extrema to playfield extrema. With detailed
sprites, neighboring living enemies require at least one completely blank
display column between occupied horizontal bounds and one blank display row
between occupied vertical bounds. The renderer increases pitch or falls back
to the smaller tier if this cannot fit; it never merges or hides an enemy.
Compact one-cell sprites need no additional gap.

## Arena and background

The border is always a complete single-cell ASCII frame: `+`, `-`, and `|`.
It is outside the logical playfield and is never confused with a projectile.
An optional star layer uses only sparse `.` cells in empty interior cells,
placed by a fixed seed and stable coordinate hash. It is static,
presentation-only, never occupies an entity or border cell, and is disabled if
it could reduce contrast or spacing. No random or time-dependent decoration is
allowed.

## HUD and state screens

The safe rectangle is the alignment basis. The hierarchy, top to bottom, is:
title (`INVADERS`), separate `Score: <n>` and `Lives: <n>` HUD readouts,
bordered playfield, controls, then state status. Title and status are the
strongest text; controls are secondary. At compact size each is a distinct
single-line row in the exact compact budget above. Larger tiers may add margins
but do not reorder these elements.

- **Start:** title, HUD, controls `A/D or arrows: move  Space/F: fire  P: pause  Q: quit`, then `Press Enter to start`; no gameplay frame is required before acknowledgement.
- **Active:** title/HUD above the complete playfield and controls/status below; no overlay obscures entities.
- **Pause:** retain the last frame and show `PAUSED - Press P to resume` in status.
- **Win:** retain the final frame and score and show `YOU WIN - Press Enter to play again`; `Q`/`Esc` remains available.
- **Game over:** retain the final frame and score and show `GAME OVER - Press Enter to try again`; `Q`/`Esc` remains available.

State text is stable, actionable, and plain ASCII. It does not alter game
state or gameplay timing.

## ANSI styling and fallback

ANSI may add bold title/status/player, a bright or inverse border, and a
consistent enemy style. ANSI is optional, stripped when unsupported, and never
carries meaning. Plain output retains every label, glyph, boundary, and action.
ANSI sequences are non-printing and excluded from width calculations.

Frames are built off-screen as bounded rows and emitted as one ordered frame.
No sprite, decoration, label, or status writes the reserved final cell, causes
autowrap, scrolls the terminal, or depends on implicit wrapping. Layout and
projection are recomputed after resize. These rules preserve the full-frame
viewport-fit and safe-serialization contracts of decisions #89, #92, #95, and
#96.

## Consequences

Compact terminals remain fully playable despite their strict vertical budget;
medium and large terminals gain recognizable classic arcade silhouettes. Stable
fallbacks and deterministic decoration make snapshots and resize behavior
repeatable without introducing gameplay changes.
