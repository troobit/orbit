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

## Protocol compatibility check

Date: 2026-10-25. Driver: `codex-cli 0.164.0`, macOS arm64. This check uses
the installed Codex against a loopback HTTP stub, **not model inference** or a
new LM Studio coding-quality test. The stub implements model listing and
Responses SSE events. `CODEX_OSS_BASE_URL` points to its `/v1` endpoint;
`CODEX_HOME` and the Git repository are disposable. The actual worker request
is distinguished from Codex's startup probe by its `tools` field.

| Request field | Previous local preset | Updated local preset |
|---|---|---|
| Tool types | `function`, `namespace`, `web_search` | `function` only |
| Delegation tools | `multi_agent_v1` namespace | Absent |
| Shell tools | `exec_command`, `write_stdin` | Both retained |
| `reasoning` | `{"summary":"auto"}` | `{}` |
| `include` | `["reasoning.encrypted_content"]` | Unchanged |
| `prompt_cache_key` | Present | Present |
| Developer instruction roles | Present | Present |

Also tested `model_supports_reasoning_summaries=false` independently with the
OSS flags: the outgoing summary and encrypted include remained. The preset
uses `model_reasoning_summary="none"`, which removes the summary request;
the encrypted include still remains. General config documentation alone is
insufficient to establish this version's local-provider behavior.

End-to-end command: `./orbit local <scratch>/specs/protocol/tasks.md
--timeout 30s --max-iterations 2`, with the loopback endpoint override. The
stub supplied two `exec_command` tool calls in separate iterations. Actual
Codex executed `make test` and `rune complete` for each selected task, returning
tool outputs to the stub before receiving its final assistant message.

Assertions passed: two real shell calls, two Makefile validation runs, both
dependent ledger tasks completed, four worker requests containing only function
tools, and two retained JSONL streams with `turn.completed`. A marker from the
scratch repository's `AGENTS.md` remained in every worker request. Cache keys,
developer messages and the encrypted include were still present as documented.
This verifies removal of unsupported tool definitions and continued shell
execution; it does not establish full LM Studio protocol compatibility.

`TestLocalCommandDispatch` first failed for both task and phase mode because
the overrides were missing, then passed after the preset change.
`make test-local` and `make build test lint` passed. The runbooks now explain
each server warning and the need to rebuild/reinstall before starting a new
iteration. See [the bugfix report](../bugfixes/local-provider-warnings/report.md).

`make install-local-next-task` updated `~/.local/bin/orbit` and its existing
`local-next-task -> orbit` link. The installed binary matches the validated
build, and the installed compatibility command's `--help` succeeds.

## Context and dependency handling check

Date: 2026-10-25. The reported real run ended with no selected task completed;
its artifacts were present. LM Studio's model listing showed 32768 loaded
tokens despite a 262144 advertised maximum. The previous preset had no
explicit context budget. The run's summary counted a failed phase as completed,
which `SaveSession` did unconditionally.

New regression assertions failed before implementation: default budgets were
absent, custom budget flags were unknown, and both failed-turn and nonzero-exit
results increased completed-phase counts. They now pass, including execution
errors, preflight validation and explicit network permission in the worker prompt.
`make build test lint` and a subsequent `make test lint` passed.

An installed-Codex 0.164.0 loopback check used explicit 32768/20480 budgets and
reported 22000 input tokens after a real shell call. The next request was a
local summary request with zero tools; the following request restored seven
function tools. The process returned `turn.completed`. This verifies automatic
compaction activation with the configured limits without calling a hosted
compaction service. It does not measure local-model summary quality.

The rebuilt Orbit also passed the earlier two-task real-Codex protocol fixture
with the new defaults. A second run with `--network` completed both tasks,
executed two Makefile checks and made two HTTP requests from sandboxed shell
tools to the loopback server. No dependencies were installed in another project.
No model reload or server overflow-policy change was made. These fixtures did
not test long-context live inference; see the follow-up below. Tokenizer
differences and oversized individual reads can still cause overflow despite
earlier compaction.

Terminal-only activity display was excluded after clarification because events
are already retained in JSONL. Smaller top-level tasks and an OpenCode comparison
remain documented options; neither task ledgers nor the driver were replaced.
See [the report](../bugfixes/local-context-progress/report.md) and Decision 15.

At commit `cdc95c6`, the longer user-driven run was in progress and its result
was pending. Its observed first-iteration outcome is recorded below.

## Authenticated live-run follow-up

The user supplied `phase-1-87438257-97a0-4883-a6cb-cb3b6dc88814.stdout.jsonl`
as a live test. It began at 17:15:44 on 2026-10-25. Its active Codex arguments
confirmed the 32768/20480 budgets, disabled delegation/web search, 2048-token
tool history and disabled shell networking. Raw events and stderr were present.
The user subsequently authorized and preferred networking for future invocations;
the active process's sandbox was left unchanged.

A normal-login loopback capture reproduced seven connected-app namespaces
despite ignored user configuration. `features.apps=false` removed them while
retaining seven function tools. The isolated-home fixture alone had missed this.
A rebuilt-Orbit fixture using normal authentication and `--network` completed
two tasks, ran two Makefile checks and made two HTTP requests from sandboxed
tools. The dispatch regression failed before the app setting and passed after it.

The live run also exposed stale restart state: status was running alongside
the previous invocation's error and completion timestamp. The existing-summary
test reproduced that combination. New starts now clear those terminal fields
while preserving history and run numbering. `make build test lint` passed for
both follow-up fixes. The active summary was not edited externally.

At 17:26:04, the actual LM Studio request had zero tools and the explicit
`CONTEXT CHECKPOINT COMPACTION` marker. The preceding ordinary request had
seven connected-app namespaces; server cache accounting was 19712 cached plus
531 uncached tokens. The compaction prefill reported 18280 uncached tokens.
The worker had recovered from a failing CLI test and subsequently reported
three passing tests.

At 17:30:34 the worker resumed a normal request with tools. Conversation items
fell from 55 to five. Server cache accounting for the resumed request was 8192
cached plus 2755 uncached tokens (10947), down from 20243 before compaction.
No TruncateMiddle events occurred during the observed first iteration.

After resumption, the worker passed Ruff lint, Ruff format checking and pytest
(three tests), wrote an implementation note and completed task 1 with rune.
It returned `turn.completed`; Orbit accepted the ledger changes, saved a
successful session entry and began iteration 2 with a fresh session. The
first session ran from 17:15:44.394588 to 17:34:03.386180: 1098.955 seconds,
about 18 minutes 19 seconds. Reported aggregate usage was 287160 input tokens,
241152 cached input tokens and 7038 output tokens, including 5339 reasoning
tokens; aggregate usage is not the maximum prompt size.

This is a successful longer live iteration and a real compaction/resumption
check. The overall multi-task run is still running; later iterations were not
included in this result. The worker still used some pipelines/trailing echoes
despite the prompt's exit-status guidance, although it noticed the actual test
failure and the final validators reported success. Prompt instructions remain
advisory and do not enforce shell semantics.

The authenticated namespace exclusion and restart-state fix were installed for
new invocations. The active parent retains its original app and network flags;
it was not restarted. Subsequent invocations of this user run should include
the authorized `--network` flag. No target-project files or active summaries
were changed by the observer.

## Iterative efficacy check and leaf dispatch

Date: 2026-10-25. The supplied `../atk/redteam/.orbit/local/` logs were read
without changing the main ledger or its active runner. They show repeated
context retrieval and dependency-cache retries; the 1800-second attempt wrote
a configuration test and reached its expected missing-module failure without
completing the selected parent. Shorter interrupted attempts have no completed
turn, so their logs alone do not establish a model failure.

Added opt-in `--leaf`, retaining bundled dispatch as the default. Real-rune
integration tests cover separate child sessions, review/resume between leaves,
automatic ancestor completion, an in-progress ready sibling behind a blocked
sibling, blocked ancestors, STOP gates in later children, unrelated advancement,
and incompatible CLI flags before log creation. The initial tests failed while
`--leaf` was unknown. `make test-local`, then `make build test lint`, passed.
The updated command was installed with `make install-local-next-task`; installed
`local-next-task --help` lists `--leaf`. A dry run selected only leaf `1.1` and
retained ancestor details without the sibling task.

Live checks use separate standalone Git roots under:

```text
/Users/r/repos/atk/redteam/.orbit/local/evaluations/bundled
/Users/r/repos/atk/redteam/.orbit/local/evaluations/leaf
```

Both start with the same standard-library Python utility stub, design, Makefile
and rune ledger at `specs/word-utils/tasks.md`. A parent contains two dependent
children: first write three expected failing normalization tests without editing
the stub, then implement normalization and pass those unchanged tests. The
cases cover trimming, whitespace collapse, lowercase conversion and empty input.
No dependencies need downloading. Each fixture has its own Git root and logs;
the main target project's source and active summary remain outside that root.

Driver/model: Codex 0.164.0, LM Studio `8-bit`, loaded context 32768, compaction
20480, workspace-write with `--network`, 12-minute session timeout. The existing
main-project worker was also using the server; timings include contention and
are observations, not a controlled throughput comparison.

The original preset ran the bundled fixture with:

```bash
orbit local specs/word-utils/tasks.md --network --timeout 12m --max-iterations 1
```

Session `73dc9bd3-f3d0-46b8-8e1d-0f5c05726036` completed the parent and both
children in 378.773 seconds (6m19s). It retained a completed turn, and Orbit
reported success. Independent `run_silent make test` passed all three tests.

The updated preset runs the leaf fixture with:

```bash
orbit local specs/word-utils/tasks.md --leaf --network --timeout 12m --max-iterations 1
```

The first leaf produced three failures specifically from the original stub's
`NotImplementedError`. Independent validation confirmed `make test` exits 2,
reports exactly three errors, and the implementation stub is byte-for-byte
unchanged. The raw independent output is retained in the fixture's
`red-validation.txt`. Workers still reread the ledger and used trailing echoes
despite guidance; prompts cannot enforce shell exit-status handling.

Session `f16afaa2-dffd-4245-b000-020297ff746b` finished in 396.267 seconds
(6m36s). Only `1.1` was completed; the parent and implementation task remained
unfinished and the session was recorded without an error. The invocation exited
1 solely because the requested one-iteration limit was reached with work left.
After reviewing that checkpoint, the installed `~/.local/bin/orbit` was invoked
again with the same options to select `1.2` in a fresh session.

The independent red-output file was written after the worker had observed its
test failure; the worker later read it during its final checks. This is a
supervised workflow check rather than a blind model evaluation. The red leaf
alone took longer than the bundled baseline, so no speed improvement is claimed.

Session `2d88ab9b-a2ff-44dd-ab0c-8a68827520ff` completed `1.2` in 359.476
seconds (5m59s). Rune automatically completed the parent. Orbit accepted the
progress, recorded a completed turn and ended the resumed invocation with
success. Both leaf sessions remain in the summary and neither has an error.
The generated test file's SHA-256 is unchanged across the implementation
session. Independent `make test` passed all three tests, along with additional
empty-input and mixed-whitespace assertions. Its output is retained as
`specs/word-utils/.orbit/local/independent-green-validation.txt` in the fixture.

| Mode | Worker sessions | Worker time | Observed result |
|---|---|---|---|
| Original bundled preset | 1 | 378.773 s (6m19s) | Parent and both children complete; three tests pass |
| Leaf tests-first checkpoint | 1 | 396.267 s (6m36s) | Only `1.1` complete; three expected stub failures; implementation unchanged |
| Installed leaf restart | 1 | 359.476 s (5m59s) | `1.2` and parent complete; unchanged tests pass |
| Leaf combined | 2 | 755.743 s (12m36s) | Successful reviewed handoff across fresh sessions |

The evaluation directory's `results.json` retains durations, session IDs,
completed command counts and limitations. Both fixtures retain their own
prompts, JSONL, stderr and standard Orbit summaries. The resumed leaf recovered
from one incorrect reference lookup; it did not edit the tests or commit work.

This establishes opt-in smaller dispatch, expected-red acceptance and a real
installed-command restart. It does not establish improved throughput, reliable
validation shell semantics, arbitrary-project completion or working native
delegation. Leaf mode doubled worker time on this tiny supervised sample;
use it for reviewable or otherwise oversized work units, rather than assume
that more sessions are faster. The supplied main ledger was not driven end to
end; the user's existing runner and source changes were left untouched.
