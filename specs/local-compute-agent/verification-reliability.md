# Local execution reliability verification

2026-10-30, Apple M5 Pro, installed Codex CLI 0.165.0 and rune commit
`5d23dda`. Work continues on the existing dirty `local-compute` branch; no
commit was created. This record covers direct `orbit local` execution.

## Reproduced failures and changes

- A real-rune ledger with twenty-one independent tasks stopped after exactly
  twenty fake worker sessions. The default is now `--max-iterations 0`, which
  derives a finite budget from initial unfinished tasks, including descendants.
  The unchanged twenty-one-task regression completes. Explicit positive caps
  still stop with an error when work remains; negative values fail preflight.
- Rune full-ledger and phase-summary subprocesses had no deadline, while
  selection ignored the run's cancellation context. All rune queries now use
  the shared executor with a thirty-second deadline, process-group cleanup and
  the owning run's shutdown context. Tests cover six query paths and a child
  process that would leave a marker if it survived cancellation.
- Rune exit zero with `success: false`, empty stdout or a missing task payload
  could imply completion. Explicit failures and malformed list payloads now
  fail closed. Valid `Tasks: []`, `Tasks: null` and rune's explicit
  `success: true, count: 0, data: []` empty response remain accepted.
- The host `codex --version` probe hung indefinitely. The adapter now gives it
  a five-second deadline and process-group cleanup. Local preflight performs
  that probe before logs or inference, including dry runs. Fake executable
  regressions assert success, nonzero exit and stalled startup behavior.

Integration coverage also exercises absolute and relative nested custom ledger
filenames, spaces, quotes and an external symlink invoked outside the target
Git repository. The ledger itself resolves inside the target Git root; this
does not remove that existing requirement.

## Host checks

`make build test lint` and `make test-local` pass. The final command was
installed with `make install-local-next-task`. Existing source and spec edits
were preserved. Tests use real rune with disposable fake workers; the live
model cases below establish inference separately.

## Live model execution

Artifacts are retained at
`.orbit/local-reliability/20261030-031916/` in the Orbit checkout. The generator
and runner are retained at `.orbit/local-reliability/verify.py`.

The two models were already loaded; no model weights, templates, context
loading or server settings were changed:

| Identifier | Loaded identity | Server context | Parallel slots |
| --- | --- | ---: | ---: |
| `8-bit` | Qwen3.8-27B-Uncensored-MLX, eight-bit, 29.53 GB | 262144 | 4 |
| `bonsai-local` | prism-ml/bonsai-27b, two-bit, 8.52 GB | 32768 | 2 |

Each disposable repository contains two initially unimplemented functions,
eight unchanged unittest assertions, an approved design, Makefile checks and a
rune-created ledger at `specs/word utilities/review's queue.plan`. Task two
depends on task one. A requirement specified only in `AGENTS.md` is checked by
the supplied tests. Orbit is invoked from outside the repository; Bonsai uses
an external symlink to the ledger. Both cases use fresh sessions with
`--phase --context-window 16384 --instructions-file` pointing at the compact
worker policy and a ten-minute per-worker timeout. The automatic iteration
budget is two. Dependencies require no downloads.

Both cases run concurrently, so elapsed times describe contention. The host
independently reruns `make test`, checks that the supplied tests are unchanged,
reads rune statuses and retains source, prompts, JSONL, summaries and console
output. Machine-readable results, runtime metadata and final source/build
hashes are retained beside the cases.

| Model | Elapsed seconds | Orbit exit | Completed tasks | Independent tests |
| --- | ---: | ---: | ---: | --- |
| `8-bit` | 452.775 | 0 | 2/2 | All eight pass, supplied tests unchanged |
| `bonsai-local` | 503.738 | 0 | 2/2 | All eight pass, supplied tests unchanged |

Both summaries end in `success`. Bonsai spent additional tool turns verifying
work after updating its ledger, but ended successfully within its deadline.
This is observable model overhead; the runner does not treat a status update
alone as a completed Codex session.

## Final installed build: timeout and recovery

The final installed binary uses normal PATH lookup, including the repaired
Codex launcher. Each fresh recovery repository starts with independently
accepted task one retained verbatim and task two reopened with rune and its
function restored to `NotImplementedError`. Initial host tests fail for that
missing behavior. Timeout probes run sequentially with both models initially
idle, so competing inference cannot be mistaken for an orphan.

| Model | Requested timeout | Observed process exit | Surviving workers | Inference after exit |
| --- | ---: | ---: | ---: | --- |
| `8-bit` | 15 s | 16.246 s, exit 1 | None | Both models idle |
| `bonsai-local` | 15 s | 15.571 s, exit 1 | None | Both models idle |

The observation includes startup and polling overhead. The eight-bit case
retains one pending task; Bonsai retains one in-progress task. Both preserve the
accepted first implementation. The same command then starts fresh recovery
sessions without resetting the code or task ledger. Recovery results are
retained separately under `final-installed/`.

| Model | Recovery elapsed | Worker result | Independent host acceptance |
| --- | ---: | --- | --- |
| `bonsai-local` | 216.645 s | Exit 0, successful summary with previous error cleared | All eight tests pass; tests and accepted first implementation unchanged |
| `8-bit` | 600.156 s | Exit 1 at ten-minute deadline after completing code and both task statuses | All eight tests pass; tests and accepted first implementation unchanged |

The additional eight-bit recovery is a **failed model turn**, not a successful
unattended recovery. It continued rechecking completed work until the deadline;
the runner terminated it and retained its failed summary. After independent
host acceptance, rerunning the same command exited zero in 0.074 seconds with
no inference, unchanged source and a successful summary. No task status or
summary was hand-edited. The original timeout and failed model turn remain in
the history and `resume-result.json`; the separate acceptance/replay evidence
is `host-accepted-finalisation.json`.

Both models finish idle with no queued inference; all experiment workers have
terminated. This fixes the demonstrated unbounded subprocess paths and default
iteration ceiling. It does **not** guarantee that a local model will end every
turn after doing correct work. Worker deadlines and retained diagnostics remain
necessary, and rerunning an already-completed ledger does not itself validate
its code.

## Independent local model reviews

Both models reviewed the actual automatic-budget and strict-list-parser code,
with the existing immutable-graph and per-iteration completion guards supplied
as context. Both returned final text accepting the logic; independent host
reading agreed. These small reviews support the test evidence rather than
replace it. They overlap recovery inference, so their timings also include
contention.

| Model | Protocol and request setting | Elapsed | Final response |
| --- | --- | ---: | --- |
| `bonsai-local` | Native streaming chat, per-request reasoning off | 12.310 s | Completed, accepts both checks |
| `8-bit` | Raw streaming completion with the installed template's explicit closed-think prefix | 15.978 s | `finish_reason=stop`, accepts both checks |

No server or model configuration was changed. Requests, raw streams, final
responses and timing records live in `model-reviews/`, including
`review-summary.json`. The reviews' references to `null` and `[]` mean explicit
`Tasks` field values; bare `null` and arrays are rejected. The budget relies on
the existing progress guard to ensure a completion per iteration.

## Host Codex launcher repair

The Homebrew cask binary stalled before user code in `_dyld_start`, while the
app-distributed Codex 0.165.0 responded immediately. Both binaries had the same
SHA-256 and valid OpenAI Developer ID signatures. Removing the cask quarantine
attribute and replacing its inode did not fix that resolved launch path; these
changes were reversed and the original cask file and attributes were restored.

`/opt/homebrew/bin/codex` now points at the working, byte-identical binary at
`~/.codex/packages/app-server-daemon/releases/0.165.0-aarch64-apple-darwin/bin/codex`.
Normal PATH lookup responds successfully. Original symlink target, attributes
and hash are retained in
`.orbit/local-reliability/codex-launch-quarantine-backup.json`. This repairs the
observed launcher, without establishing the underlying macOS launch-policy
cause. A later Homebrew update may replace the link.

The initial concurrent model cases use a per-run PATH selecting that same app
copy. Their processes started before the final strict rune parser and Codex
startup preflight landed. The final build is checked separately by the host
suite, installed-command checks and isolated cancellation/recovery; source and
binary provenance are retained in `build-provenance.json`.

## Evidence limits

A worker failure can retain useful code and rune status updates. Independently
validate those changes before resuming; this change does not implement the
planned host-owned task completion or `orbit prepare` workflow. Completion of
these benign fixtures does not establish correctness for arbitrary generated
code, unavailable dependencies or human STOP gates.

The separate `local-run-comparison` and `local-task-preparation` implementation
ledgers remain pending. Direct-local results do not satisfy OpenCode aliases,
variant comparison or unrelated feature acceptance. The initial local-model
review API calls both hit a 150-second deadline without final review text and
returned to idle. A subsequent eight-bit chat request ended at its 512-token
limit with reasoning only after 101.055 seconds. These remain unsuccessful
review attempts; only the final completed reviews above count as acceptance.
