# Peer review (revision 1): claude-code

Reviewed `requirements.r1.md` and the revision 1 critique. No external model was reachable (Codex was not enabled for the reviewer and Kiro was unavailable), so this review came from two further Claude Code subagents, not from a second harness. AC numbers refer to revision 1.

## On the critique

- **C1:** agree; the proposed fix fails because streamed responses are mostly not logged. Later data overrides the proposed prefix chain: Codex requests carry `client_metadata.thread_id` (see `specs/apsis-lmstudio-logs/handoff.md`), and `previous_response_id` never appears.
- **C2:** agree.
- **C3:** agree, top issue; word it as "opens under the scope it was listed in". Identifiers must pass `SanitizeSessionID` (no `/`, at most 256 characters), so path-based identifiers do not work; use `thread_id` for exec streams.
- **M1, M2, M5, M6, M7, M8, M10:** agree. A killed run leaves `item.started` with no `item.completed`; render it as incomplete.
- **M3:** partly; a cache keyed on path, size and modification time plus a time budget.
- **M4:** partly; containment credits worktree streams to the parent.
- **M9:** minor.

## Recommendations

- LM Studio in its own spec; drop locality; move the prompt-file convention to the work that writes it; add an all-projects mode to serve; assign an exec stream to the nearest ancestor holding `.git`, else all-projects only.

## Missed by the critique

1. `transcript.Entry` carries Claude's wire-format JSON tags and `Model` is `json:"-"`; marshalling it makes Claude internals the public contract. Require a dedicated output layout, checked with golden tests.
2. Exec streams carry no model or provider, so the provider field shows nothing for them.
3. A `codex exec` run without `--ephemeral` also writes a rollout, so it would list under both `codex` and `codex-exec`.
4. LM Studio response bodies have the same attribution problem as notices under parallel slots.
5. A direct `.log` path and stdin are undefined (detection is JSON-line based).
6. Exposure: open sidecars and streams only inside configured directories after resolving symbolic links; all-projects serve raises the stakes.
7. 1.3 departs from the existing convention (rollout parser warns and skips unknown types); choose deliberately.
8. Listing JSON should be an object holding the version and warnings; use one integer schema version.
9. Match endpoints exactly; emit each LM Studio message once; Codex requests carry an `<environment_context>` working directory.

## To verify

Whether `codex exec resume` reuses `thread_id`; whether a rollout's ID equals `thread_id` (since confirmed for one interactive session); whether `turn.completed.usage` is per turn or a running total.
