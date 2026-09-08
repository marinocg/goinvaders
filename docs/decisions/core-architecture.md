# Core Architecture Decision

## Status

Accepted.

## Context

The repository needs a minimal, explicit architecture contract for a deterministic terminal Space Invaders skeleton. Later implementation issues should be able to work in narrow, non-overlapping path scopes without inventing package boundaries, timing semantics, or terminal coupling assumptions.

This decision defines only core architecture and contracts. It does not define gameplay rules, rendering implementation, keyboard bindings, or terminal library choices.

## Decisions

## Game state and update model

## Entry point and ownership boundaries

## Package layout

The package layout targets Go 1.23.

Use the following repository-relative package paths:

- `internal/game` - core game domain state types and pure update logic.
- `internal/input` - input abstraction contract consumed by the loop.
- `internal/loop` - deterministic loop orchestration and update scheduling.
- `cmd/invaders` - executable entry point and wiring/composition.

No additional architecture layers are required for the initial skeleton.

### 2) Ownership boundaries

#### `internal/game`

Owns:

- Canonical game state structure and related value types.
- Update API that advances state using deterministic timing input.
- Rendering-facing snapshot data derived from state, if needed, as plain Go data types.

Does not own:

- Terminal rendering implementation details.
- Terminal input adapters or key bindings.
- Wall-clock reads, timers, sleeping, goroutine scheduling.

Constraint:

- Logic in this package should be deterministic and testable as pure state transitions for given inputs.

#### `internal/input`

Owns:

- Logical command/event types consumed by game loop/core update logic.
- Interface contract for a source that yields input events.

Does not own:

- Terminal library dependencies.
- Concrete keyboard-to-command mapping policy (owned by adapters wired in `cmd/invaders` unless later split into a dedicated adapter package).

Constraint:

- Contracts here must be terminal-agnostic and safe for unit tests with fakes/stubs.

#### `internal/loop`

Owns:

- Tick progression orchestration.
- Collection of input events per tick boundary.
- Invocation of `internal/game` update with deterministic timing data.
- Loop stop conditions and cancellation handling (through context) as orchestration concerns.

Does not own:

- Gameplay rules.
- Rendering behavior or frame drawing.
- Terminal event polling implementation details.

Constraint:

- Loop orchestration must be testable with injected tick/input sources (no hard dependency on `time.Now`/`time.Sleep` in core scheduling logic).

#### `cmd/invaders`

Owns:

- Program startup and dependency wiring.
- Selection/initialization of concrete terminal adapters.
- Process-level concerns (signals, context lifecycle, exit codes).

Does not own:

- Core gameplay update rules.
- Reusable domain abstractions.

## Timing model

The loop uses a fixed-step tick model for deterministic updates.

- A tick duration constant is configured at startup (for example, `time.Second / 60`), but the specific rate is not fixed by this decision.
- Core update logic receives elapsed simulation time as an integer tick delta, not floating-point wall-clock time.
- For each loop iteration, orchestrator computes how many ticks to process based on an injected time/tick source.
- Update calls are performed as a sequence of discrete tick applications (`N` single ticks), preserving deterministic order with input events.
- Tests can drive loop progression by supplying synthetic tick pulses and asserting resulting state transitions exactly.

Rationale:

- Fixed-step integer tick progression avoids drift and non-determinism from variable frame deltas.
- It also keeps core logic independent from terminal refresh rate and wall-clock jitter.

## Input contract

The core loop consumes logical input events, not terminal keys.

Contract requirements for `internal/input`:

- Define a closed set of logical commands/events needed by the skeleton (for example: move-left intent, move-right intent, fire intent, quit intent).
- Represent events as plain Go values suitable for deterministic replay in tests.
- Provide an interface for polling/draining events that does not mention terminal libraries.
- Preserve event order within a loop iteration.

Non-requirements in this decision:

- Exact keyboard mappings (e.g., `a`/`d` vs arrow keys).
- Key repeat behavior.
- Terminal raw-mode handling.

### 5) Rendering-facing data boundary

- `internal/game` may expose read-only snapshot/state-view data as plain structs for renderers to consume.
- Renderer implementations (terminal or otherwise) must not mutate game state directly.
- Renderer-specific formatting, rune/color choices, and draw buffering are out of scope for core packages.

## Package ownership for later implementation issues

To enable narrow `allowed_paths` in child issues:

- Core state/update implementation issues may own `internal/game/**`.
- Input contract and fake-input test fixture issues may own `internal/input/**`.
- Loop scheduling/orchestration issues may own `internal/loop/**`.
- CLI wiring/runtime bootstrap issues may own `cmd/invaders/**`.

Cross-cutting changes across these paths should be treated as explicit integration tasks rather than default behavior in a single child issue.

## Explicitly unresolved ambiguities

The following are intentionally not decided here and must be resolved by later product/gameplay decisions:

- Concrete gameplay rules (enemy behavior, projectile motion, collisions, scoring, lives, win/lose conditions).
- Initial tick rate value and any runtime configurability.
- Whether input commands are level-triggered (held) or edge-triggered (press/release), beyond preserving event order.
- Whether rendering runs once per processed tick or on a separate cadence.
- Pause, slow-motion, catch-up cap, and max-ticks-per-iteration policies.

## Consequences

- Future tasks can implement deterministic tests by injecting tick and input sources without terminal dependencies.
- Terminal adapter work can proceed independently of core game/update work.
- Package boundaries reduce overlap and simplify ownership-scoped issue packets.
