# Local compute verification

Date: 2026-10-25. Platform: macOS arm64.

## Automated and installation checks

- `make build test lint`: passed. The full Go suite includes real-rune local
  dispatch tests. `make test-local` also passed separately.
- Worker tests cover one-task and phase dispatch, model selection, descendants,
  internal/external dependencies, STOP gates, persistent and worker halt
  signals, no progress, changed graphs, reopened tasks, unrelated advancement,
  failed-turn events with exit zero, incomplete turns, process failures,
  timeout diagnostics, iteration limit and resume, and relative paths with
  spaces and shell punctuation.
- Shared execution tests verify output is readable before process completion,
  captured output remains intact, and Unix cancellation kills child processes.
- Installer checks passed for preserving an unrelated file, replacing the known
  prototype symlink, repeated installation, and installed command routing.
- `make install-local-next-task` installed Orbit and replaced this machine's
  `~/.local/bin/local-next-task` link with `local-next-task -> orbit`.
- The installed command successfully dry-ran `../atk/redteam/tasks.md` from the
  Orbit checkout, selecting task 1 and resolving the target Git root correctly.
  That ledger was used only for a preview and was not modified.

The installed validation tools are Go 1.27.3 and golangci-lint 2.15.0. Installing
golangci-lint through Homebrew also updated its Go dependency. The previous
user-bin linter (2.1.6, built with Go 1.26) could not analyze the newer standard
library. Current lint exposed five existing discarded error-channel receives
in transcript tests; they now assert clean shutdown. The full suite also exposed
an HTTP integration shutdown timeout; draining three unread response bodies
fixed connection reuse and cleanup. No server implementation changed.

## Live model

- Driver: `codex-cli 0.164.0`, native LM Studio provider.
- Model API identifier: `8-bit`.
- Installed model: `Qwen3.8-27B-Uncensored-MLX/8-bit`, qwen3_5 architecture,
  8-bit quantization, 29,528,175,221 bytes; context length 32,768.
- LM Studio was open, but its server was stopped and no model loaded at the
  start of verification. The installed model was loaded and the loopback server
  started explicitly for this check. No model download was needed.
- Command: `orbit local <scratch>/tasks.md --timeout 15m --max-iterations 2`.
- Scratch repository:
  `/private/var/folders/mq/ws4rpgbx7v51dg5g76f5wfj80000gn/T/orbit-local-smoke-kkoyggxc`.
  The scaffold contained a Makefile running unittest, a design document and a
  two-task rune ledger. Task 2 depended on task 1; both belonged to one source
  phase. Task 1 added `greet(name)` and tests; task 2 added `farewell(name)` while
  preserving the first function and tests. Each strips surrounding whitespace
  and rejects blank names.

| Iteration | Work | Duration | Result |
|---|---|---|---|
| 1 | Implement greet and four tests | 487.529 s | Task 1 completed; task 2 remained pending |
| 2 | Add farewell and four tests | 358.339 s | Task 2 completed; all eight tests passed |

Total worker time: 845.868 seconds (about 14 minutes 6 seconds). Orbit validated
each ledger change, used two distinct fresh session IDs, and marked the run
`success` after exactly two iterations. Both tasks retained their original IDs
and dependency. An independent `run_silent make test` in the scratch repository
passed all eight tests afterward. The scratch repository still has no HEAD
commit, confirming the worker left its edits uncommitted. Source and generated
notes were inspected as well as the ledger and summary.

The live stream includes recoverable apply_patch errors and Codex model-metadata
refresh warnings. Neither is a failed turn. The first iteration recovered,
wrote the implementation and tests, and returned `turn.completed`. Raw stdout,
stderr and prompts are retained under the scratch ledger's `.orbit/local/`.
This confirms plumbing and a small coding workflow; it does not measure output
quality on larger projects. Model-driven retrieval and phase dispatch were
verified with real rune and fake Codex, rather than a separate inference run.

The live check also exposed empty standard Codex JSON artifacts. The final
adapter now populates RawJSON for JSONL execution; regression tests verify the
standard Orbit session files contain completed-turn events. The live UUID
stdout files already retained the original inference streams throughout.
