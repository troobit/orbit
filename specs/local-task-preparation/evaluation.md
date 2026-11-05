# Local task preparation evaluation

The coding ledger implements preparation and shared prepared execution. This
file describes live checks; completing the coding ledger does not claim them.
The user has authorised repeated wt-2 local trials, retaining useful work and
failures. No further blanket approval is required for those trials. Runtime
prerequisites and independent acceptance still apply.

## Implementation comparison

Commit reviewed spec/ledger changes before creating variants: worktrees start
from HEAD. Use configured aliases, an explicit task file and a distinct branch
prefix. Current CLI syntax is:

```sh
orbit run --tasks-file specs/local-task-preparation/tasks.md \
  --variants 2 --variant-agents codex,opus --parallel \
  --branch-prefix orbit-local-preparation
```

This builds the feature using the existing hosted phase workflow. First land
the no-progress guard owned by `local-run-comparison` or supervise with a finite
outer invocation deadline; the new prepared-worker guard cannot protect its
own bootstrap implementation. No manual acceptance/approval tasks belong in
the coding ledger. Check every completed branch independently with
`make test-local`, `make test`, `make lint` and `make build` before adoption.

## Prepared local execution from a fork

Use a binary built from the adopted hosted implementation. Create an isolated
fork/checkout at the recorded target baseline; never let absolute paths point
back into the original checkout. First use the two-behaviour offline fixture
from coding phase 6, then a bounded real-source slice such as Sanarte diagnostics.

1. Prepare/import a plan for the same approved source outcomes. Inspect every
   coverage row and deferred prerequisite, then approve that exact candidate.
   Do not copy completion statuses without their host evidence.
2. Run the candidate with `orbit local <candidate>/tasks.md --model 8-bit`,
   recorded context and positive session/iteration limits. Verify each worker
   packet, host result and host-owned task transition. A supplied test/patch
   template is scaffolding and must be recorded as such.
3. Repeat a failure/resume case: preserve partial edits, deliberately fail one
   host check, restart unchanged work and verify it is rechecked before progress.
   Replanning writes a new candidate, preserving the old plan and evidence.
4. Exercise prepared variants with the configured local Codex alias, for example
   `--variants 2 --variant-agents local-8-bit --parallel --max-parallel 2`.
   Aliases use hyphens; `local/8-bit` is not a supported alias name. Before this
   feature lands, an alias copies worker flags without local runner guarantees.
5. Keep the comparison judge and host result-commit helper explicit. Two local
   variants measure shared-server contention. A status-only report, a warning-only
   post-command or completed source prose does not establish successful judging,
   attributable commits or code acceptance. Use comparison's reviewed helper
   once implemented; do not stage logs or secrets.

Record installed CLIs/rune revision, exact model identity (not only its alias),
loaded context/parallel slots, budgets, task/packet/source hashes, times, retries,
timeouts, independent Makefile results, changed paths and unresolved original
acceptance. Run an uncontended cancellation probe before long trials. Do not
reconfigure/reload the shared model server or interrupt unrelated model users.

## Sanarte evidence: 2026-10-29

Target: `/Users/r/repos/sanarte/.worktrees/wt-2`, branch `wt-2`, baseline
`434a0c7`. Source is `specs/simplify-for-square/tasks.md`, task `v3aumdu`.
The original outcome combines iframe implementation, API diagnostics, browser
checks and provider feasibility. It remains incomplete throughout narrow trials.

| Attempt | Packet | Result | Learning |
| --- | --- | --- | --- |
| Original | Broad booking-fallback task | Interrupted after 1158.893 s; no source edit; only original task status advanced | Repeated whole-document reads do not establish implementation progress |
| First split | Five diagnostic scenarios, separate RED/GREEN tasks | RED timed out after 600.012 s; no source edit or task completion | A smaller title and file scope still leave too much context retrieval/design work |
| Supplied regression | One fixture call and exact test/assertion supplied | Exact RED independently verified; worker timed out before finalising; useful test retained | Code acceptance and completed model turns are different events |
| Supplied implementation | One fixed server-only warning | Worker completed; unchanged RED test and independent test/check/build passed | Bounded scaffolding enabled a real source handoff |
| Locally planned diagnostic | JSON data → reviewed rune RED/GREEN ledger | Both fresh workers completed; all 16 tests/check/build passed independently | Reject truncated output and review semantics before materialisation |

The host baseline `make check`, `make test` and `make build` passed before the
trials. Original and first-split artifacts are retained under the target's
`specs/simplify-for-square/.orbit/local/evaluations/` as `baseline-*` and
`five-scenario-*`. Cancellation stopped the original Orbit/Codex processes;
LM Studio reported `IDLE`. Server context was 262144, worker context 32768,
compaction threshold 20480, model alias `8-bit`, loaded size 29.53 GB and four
server slots. No new model was loaded. These observations are indicative, not
a model-capability benchmark or acceptance of the original Sanarte feature.

## Completion criteria

A passing preparation experiment requires a reviewed source-coverage map,
valid generated rune ledger, actual fresh workers, host-verified results and
retained failures/resume evidence. Full source feature completion additionally
requires its original independent acceptance, including any owner/provider
checks that the local execution candidate explicitly deferred.
