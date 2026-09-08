# Terminal UX Decision

## Status

Accepted.

## Context

The game core exposes terminal-independent state and input contracts. This
decision defines the presentation contract for a readable, playable terminal
client without moving terminal concerns into `internal/game`, `internal/input`,
or `internal/loop`. It does not change gameplay rules or decide when the game
enters a win or game-over state.

## Terminal layout and sizing

The terminal viewport is measured in character columns and rows. The game
requires at least **40 columns by 18 rows**, including the border and HUD.
This is a presentation minimum, not a change to the logical game arena.

- The renderer maps the complete logical arena into the available playfield
  rectangle while preserving entity ordering and keeping every entity inside
  the visible border.
- At or above the minimum, the playfield uses all available interior space;
  extra columns and rows are distributed as margins rather than changing HUD
  placement or clipping entities.
- A resize is handled at the next render boundary. The renderer recomputes its
  layout from the current terminal size and never panics or writes outside the
  viewport.
- If the terminal is smaller than the minimum in either dimension, normal
  gameplay rendering is suspended and the renderer shows a deterministic,
  centered `Terminal too small` message with the required size (`40 x 18`).
  It must not draw a partial game frame or advance gameplay solely because of
  the size failure. Once the size is sufficient, normal rendering resumes.
- Terminal-size query or output errors are reported to the UI-flow owner. The
  flow must stop safely rather than continuing with an unknown layout.

## Rendering contract

The renderer consumes an immutable game snapshot and presentation state. It
does not mutate game state, apply input, decide transitions, or own the tick
schedule. Rendering output consists of one complete frame:

- A bordered playfield containing every visible entity at its mapped logical
  position. The player, player projectiles, enemies, and any future entity
  types have stable, distinct one-cell glyphs or documented multi-cell
  sprites. The renderer does not infer gameplay from glyphs.
- A HUD outside the playfield showing score and remaining lives. HUD labels and
  values are redrawn from the snapshot each frame; stale values must not be
  retained.
- A short status line identifying the current presentation state when paused,
  won, or game over.

Frames are built in an off-screen character buffer, then written as one
ordered frame to the terminal output. The renderer clears or homes the cursor
before replacing the previous frame, hides the cursor during active drawing,
and restores it on shutdown. It must not depend on terminal scrollback or
incremental cell updates. Redraws occur after state changes and at the chosen
render cadence, not in a busy loop; identical frames may be skipped. A failed
frame write is fatal to the UI flow and must be returned rather than silently
ignored.

## Controls and input mapping

The terminal input adapter translates keys into the existing
`internal/input.Event` values and preserves arrival order. It never emits
terminal-library values across its public boundary.

| Key | Logical result |
| --- | --- |
| `Left Arrow` or `a` | `input.CommandMoveLeft` |
| `Right Arrow` or `d` | `input.CommandMoveRight` |
| `Space` or `f` | `input.CommandFire` |
| `q` or `Esc` | `input.CommandQuit` |
| `p` | UI-flow pause toggle; not a gameplay event |
| `Enter` | Start/restart acknowledgement on start, win, or game-over screens |

Keys not listed above are ignored. Key repeat is treated as repeated key
presses by the adapter; no held-key state is invented. `p` is handled by the
UI flow so pausing cannot accidentally become a new core command. `q` and
`Esc` always request quit, including while paused or on a terminal screen.

## Screen flow and failure behavior

The UI flow owns presentation state and chooses when to render. It delegates
simulation and terminal-state decisions to the existing core contracts.

- **Start:** show the title, controls, and `Press Enter to start`. Do not step
  the game or consume gameplay commands until start is acknowledged.
- **Active play:** render the current snapshot and HUD. Forward mapped
  movement and fire events to the logical input source. `p` changes only the
  presentation flow to paused; while paused, simulation is not advanced and
  gameplay events are not forwarded.
- **Pause:** keep the last snapshot visible, show `Paused` and `Press P to
  resume`, and continue accepting only pause and quit controls. Resuming
  returns to active play without resetting state or adding a simulation tick.
- **Win:** when the core reports its authoritative win state, show the final
  snapshot, score, and `You win` with `Press Enter to play again` and `q`/`Esc`
  to quit. Do not define or re-evaluate the win condition here.
- **Game over:** when the core reports its authoritative game-over state, show
  the final snapshot, score, and `Game over` with `Press Enter to try again`
  and `q`/`Esc` to quit. Do not define or re-evaluate the game-over condition
  here.

Restart acknowledgement creates a new game through the application wiring;
the renderer does not reset state. A terminal screen remains visible until a
valid acknowledgement, quit, or fatal I/O error occurs.

## Ownership and paths

These exact repository-relative paths are reserved for later implementation
packets:

- `internal/render/terminal.go` owns the terminal renderer, layout calculation,
  frame buffer, glyphs, HUD, cursor handling, and too-small/error rendering.
- `internal/input/terminal.go` owns raw terminal setup/teardown, key decoding,
  and mapping terminal keys to `internal/input.Event`. The existing
  terminal-independent contracts in `internal/input/input.go` remain unchanged.
- `internal/ui/flow.go` owns start/pause/active/win/game-over presentation flow,
  render cadence, forwarding of logical events, restart acknowledgement, and
  shutdown coordination.
- `cmd/invaders/main.go` owns composition, concrete terminal construction,
  process signals, and exit status; it does not contain layout or key mapping.

No terminal library or dependency is required by this decision. A later
implementation may justify one explicitly, but it must preserve these public
ownership boundaries and the standard-library-compatible logical contracts.

## Consequences

Future renderer and adapter work can be implemented independently and tested
with snapshots, buffers, and synthetic key streams. Small terminals fail
closed with a stable message, while adequate terminals receive complete,
flicker-resistant frame replacements. Gameplay transition rules remain solely
the responsibility of the core gameplay/state implementation.
