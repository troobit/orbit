# M5 local execution verification

2026-10-29, Apple M5 Pro, 64 GiB unified memory, Codex CLI 0.165.0.
These are actual local inference runs, with independent host checks.

## Runtime and instruction envelope

`lms ps --json` identified the loaded model as
`Qwen3.8-27B-Uncensored-MLX/8-bit`, 29,528,175,221 bytes, served as `8-bit`,
with context 262144 and four parallel slots. Worker contexts were 32768 except
the explicitly labelled 16384 runs. Model identity was checked independently
of the alias.

The installed tokenizer's chat template contained
`reasoning_effort|default('xhigh')`. Its custom model entry exposed no reasoning
configuration: a native API request with `reasoning: off` returned HTTP 400,
`Model '8-bit' does not expose reasoning configuration.` A client-side setting
therefore did not establish the effective template setting.

The operator authorised taking over available compute and repeated experiments.
Only this investigation's workers were stopped before a controlled reload.
Both `tokenizer_config.json` and `chat_template.jinja` were backed up, then the
single fallback was changed from `xhigh` to `low`. Weights were unchanged. The
same model/context/four slots were reloaded; the model index then contained the
`low` template and no `xhigh` fallback. This changes the default, without removing
support for explicit reasoning settings. It does not guarantee a fixed thinking
budget.

Installed files are under
`~/.lmstudio/models/orcarouter/Qwen3.8-27B-Uncensored-MLX/8-bit/`.
The exact originals and before/after SHA-256 values are retained at
`.orbit/local-m5/template-backup-20261029-022838/` in the Orbit checkout.
Restoring both originals and reloading restores the prior default.

Separately, the shorter Codex base instructions reduced observed requests from
about 10K input tokens to 5–6K. The initial experiment used a PATH wrapper;
the implemented `--instructions-file` option removes that wrapper. Default local
and hosted instructions remain unchanged unless the option is supplied.
The supported override is documented in the
[official Codex configuration reference](https://developers.openai.com/codex/config-reference).
LM Studio's [reasoning API](https://lmstudio.ai/docs/developer/rest/chat)
documents that supported settings are model dependent.

## Independent two-task runs

Each disposable Git repository begins with two `NotImplementedError` functions,
six supplied unittest cases, Makefile checks and a rune-generated ledger.
Phase 1 implements whitespace/lowercase normalization; phase 2 depends on it and
implements order-preserving deduplication. Each phase uses a fresh Codex session.
The host reruns `make test` and reads rune state after Orbit terminates.

| Configuration | Elapsed seconds | Orbit exit | Completed tasks | Host tests |
| --- | ---: | ---: | ---: | --- |
| Default Codex instructions, 32768 context | 750.216 | 0 | 2/2 | Pass |
| Compact instructions, 32768 context | 594.179 | 0 | 2/2 | Pass |
| Compact instructions plus supplied implementation hints | 531.975 | 0 | 2/2 | Pass |
| Compact instructions, 16384 context | 533.160 | 0 | 2/2 | Pass |
| Repeated compact/16384 run | 404.889 | 0 | 2/2 | Pass |
| Native `--instructions-file`, project guidance checks | 470.269 | 0 | 2/2 | Pass |

The last run has eight tests: two additional assertions require the docstrings
specified only in the repository's `AGENTS.md`. Both passed, showing project
guidance was consumed with the replacement base instructions. The new flag
was used directly; no wrapper was on that run's PATH.
These experiments used parallel `orbit local` processes in isolated repositories.
The standard `orbit run --variants` dispatch policy is unchanged.

These six successful runs contain twelve actual worker sessions. They ran with
overlapping inference and changing contention, so timings are indicative and
do not establish isolated throughput or causal speedups. An earlier three-way
matrix was deliberately interrupted after about 184 seconds to test the runtime
configuration change; those failures and partial ledgers remain retained.

Artifacts are under the checkout's `.orbit/local-m5/`:

- `matrix-20261029-022505/`: interrupted initial comparison.
- `matrix-20261029-022946/`: default, compact, scaffolded and repeated 16K runs.
- `matrix-20261029-023101/`: independent compact/16K run.
- `matrix-20261029-024022/`: direct flag and project-guidance run.

Each case retains source/tests, baseline commit, rune ledger, console log,
prompts, JSONL, summary, host-check output and machine-readable result.

## Real Sanarte source

The authorised target was `~/repos/sanarte/.worktrees/wt-2`, source task
`v3aumdu`. The original broad task was interrupted after 1158.893 seconds with
no source edit. A five-scenario diagnostic RED assignment then timed out after
600.012 seconds without an edit.

A supplied single-test packet produced the exact intended RED result: all 14
existing tests passed and the new diagnostic assertion failed. That worker
timed out after 360.013 seconds before finalising; the host independently
accepted its useful test and retained the failed session. The next worker used
the retained test, implemented the fixed server-only warning and completed its
Orbit session in 560.070 seconds. Host `make test`, `make check` and `make build`
passed, and the RED test's SHA-256 remained unchanged.

This accepts only the diagnostic slice. The original iframe, browser and real
provider outcomes remain incomplete. Their source task was not marked done.
Target artifacts live in `specs/simplify-for-square/.orbit/local/evaluations/`.

### Local planning → rune → implementation workers

A subsequent local planning call returned a RED/GREEN task plan for the
write-disabled diagnostic. Its first response hit the 2048-token output limit
and contained incomplete JSON; it was retained and rejected. A concise schema
with a 4096-token allowance returned a complete object in 44.559 seconds
(429 output tokens, including 175 reasoning tokens).

Structural checks verified source identity, two ordered items, exact change
paths, dependencies and Makefile checks. Semantic review caught one error:
the GREEN detail called `console.warn` with one array, while the behaviour
requires two scalar strings. The reviewed plan corrected that detail, retaining
both versions at `.orbit/local-m5/planning-20261029-025117/`.

Host code materialised the reviewed data using rune and read it back as two
dependent phases in Sanarte's `tasks-local-write-diagnostic.md`. The RED worker
completed in 311.825 seconds. Independent host checking confirmed 15 existing
tests pass and exactly the intended new assertion fails. The GREEN worker
completed in 294.735 seconds, preserving the RED test hash. Independent host
`make test` (all 16 tests), `make check` and `make build` passed; both generated
ledger tasks are complete. The original booking feature remains incomplete.

This is a supervised prototype using explicit input constraints and coordinator
review. It does not implement the planned `orbit prepare` command or establish
autonomous semantic coverage. It does demonstrate local planning data consumed
programmatically by rune and then by separate fresh implementation workers.

## Lower-quantisation and restart trial

The installed `prism-ml/bonsai-27b` two-bit MLX model (8.52 GB) was loaded as
`bonsai-local` with 32768 server context and two slots, alongside the eight-bit
model. With the same compact instructions and project-guidance fixture:

- The 16384-worker-context run completed both tasks in 615.372 seconds, with
  independent tests passing.
- The 32768-worker-context run timed out after 600.223 seconds. It left the
  first function and one completed ledger task, but no accepted Orbit phase.
  Independent `make test-normalize` passed for that retained implementation.
  Raw failure, source/ledger history and output remain at
  `.orbit/local-m5/matrix-20261029-025938/bonsai-32k/`.

The failed case resumed with the successful 16384 context budget and completed
the remaining task in 69.773 seconds. Its eight independent tests passed,
including the project-guidance assertions, with both ledger tasks complete.
The first implementation and failed-session history were retained; a fresh
worker finished the remaining function. The recorded `resume-result.json` and
`retained-normalize-host-check.txt` distinguish recovery from the first attempt.
A model's size and advertised maximum context do
not establish an effective worker profile. Source task state and accepted
Orbit phases can differ after an interrupted worker; independently recheck
retained code before relying on its task completion.

## Code validation

The instruction-file regression failed before implementation. Actual argv,
relative/quoted paths, invalid-file preflight and unchanged task/sandbox/provider
flags pass through real-rune/fake-Codex tests. `make test-local`, `make lint`,
`make build`, `make test` and `make install-local-next-task` passed.

The full suite's existing Copilot version smoke test launched a hanging
`copilot --version`; only that test child was terminated. The test explicitly
accepts a CLI error. No Copilot code was changed. Local tests and lint passed
without that intervention.

The installed compatibility entry point exposes `--instructions-file` and routes
to the same implementation. Programmatic preparation, host-owned completion and
readiness receipts in `local-task-preparation` remain a separate planned feature.
