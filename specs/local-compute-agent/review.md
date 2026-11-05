# Local execution readiness

Reviewed 2026-10-29 against Orbit commit `0d57cd3`. The requested
`local-compute-run` folder is named `local-compute-agent` in this repository.

## Ownership and current evidence

| Work | Owner | Readiness |
|---|---|---|
| Codex/LM Studio task, phase, leaf and explicit instruction-file dispatch | This spec | Implemented; twelve top-level tasks complete; real M5 results in [verification-m5.md](verification-m5.md) |
| Selection, immutable ledger checks, bounds, halt and cancellation in `orbit local` | This spec | Implemented; integration and limited live evidence in [verification.md](verification.md) |
| OpenCode event parsing and standard variant completion checks | [local-run-comparison](../local-run-comparison/smolspec.md) | Pending; four coding phases now tracked there |
| Disposable comparison and later Sanarte trial | [local-run-comparison](../local-run-comparison/prerequisites.md) | Opt-in evaluation after coding and runtime prerequisites |
| Broader hosted-run termination and accounting | [run-termination](../run-termination/design.md) | Separate pending proposal; must reconcile its older task plan before dispatch |
| Local agent writes requirements, design and task phases | Future spec | Not implemented or proved by either existing spec |
| Decompose approved work into programmatic local-worker packets | [local-task-preparation](../local-task-preparation/requirements.md) | Planned; six coding phases with host-owned acceptance and shared prepared dispatch |

The live evidence establishes small approved-ledger execution and a supervised
red/green handoff. It does not establish completion of an arbitrary project,
speed gains from leaf dispatch, reliable native delegation or unattended SDD.
Historical observations that a long run was still active are observations from
their recorded date, not a claim about its present state.

## Review findings and repairs

- **SJ-SUP-001:** The alternate-harness suggestion in this spec is now linked
  to its concrete owner. Comparison aliases retain standard variant semantics;
  they must not be described as `orbit local` runs.
- **SJ-MODE-002:** Comparison lacked a task file. Its coding ledger now has
  small phases, explicit acceptance, red/green pairs and dependencies.
- **SJ-SUP-002:** Comparison's count/next-ID fingerprint and timeout promises
  were inconsistent with its requirements and the retry implementation. It also
  conflated a generated status report with a successful judge comparison. Its
  current contract requires an actual selected-task completion and records
  adapter-dependent retries. Earlier decisions are annotated as superseded.
- **SJ-SCOPE-001:** Coding work and environment-dependent evaluation now have
  separate ledgers. Original Sanarte outcomes keep their IDs with their full
  owner acceptance; narrower code tasks get new IDs. No browser or live Square
  requirement is silently satisfied by mocks or a type check.
- **SJ-SUP-001:** `run-termination` proposed a competing variant guard and state.
  The shared variant guard belongs to comparison; the older termination plan
  has a reconciliation gate instead of claiming these changes are delivered.

## Small steps toward local SDD

1. **Repeatable execution fixture — next deliverable.** Complete comparison's
   coding ledger. Its fixture creates rune tasks and two phases from a fixed
   approved spec using the CLI, without downloads or owner services. First
   run that fixture with `orbit local` through both phases and independently
   check its code, task states and logs. Then test the variant aliases. The
   two routes have separate results; comparison success cannot stand in for
   local runner acceptance. Sanarte and a hosted comparison judge are not
   prerequisites for proving local ledger execution.
2. **Prepare one ledger from approved inputs.**
   [Local task preparation](../local-task-preparation/design.md) owns the
   programmatic handoff, source lineage, target limits and host acceptance.
   Give a planning worker a fixed, approved small design and acceptance criteria.
   Its output is a separate `tasks.md`, created through rune, with two bounded
   phases, stable IDs, working references and explicit checks. Validate parsing,
   requirement coverage, dependency existence/cycles and scope before execution.
   Retain the authored file and reviewer corrections. An active implementation
   ledger must remain immutable apart from statuses; the planner's dispatch
   ledger and its generated output are different files. Its requirements,
   design, tasks and review boundary are recorded in that spec.
3. **Execute the reviewed generated ledger.** Run it with runner selection in
   a fresh disposable repository. Require every task complete, independent
   passing checks, a cleanly terminated run and retained prompts/logs. Repeat
   before increasing problem size; failure should identify a bounded unit that
   can be reviewed and resumed.
4. **Extend authoring one stage at a time.** Only then test local requirements
   and design drafting, retaining the SDD approval gates. Human decisions stay
   outside unattended implementation ledgers. Native subagents, automatic
   approval and an autonomous planner/executor loop are not implied here.

## Ready-to-start checks

The next comparison implementation is phase 1 of its `tasks.md`, using OpenCode
fixtures without a running model. Its live-evaluation ledger has STOP gates.
The separate M5 execution follow-up added the instruction-file option, ran
local models, retained Sanarte diagnostic changes and installed the updated
compatibility command; evidence is in [verification-m5.md](verification-m5.md).

## Validation

Validation of this review: rune parses both existing ledgers and both new
comparison ledgers; dependency/reference checks pass; a disposable rune copy
advances all four coding phases and auto-completes their parents. Current-source
dry runs select only the first red leaf and reject the evaluation STOP gate
without logs. `make test-local`, `make lint` and `make build-orbit` pass. The
scoped mechanical audit has no findings; unrelated historical specs were not
repaired. These are plan/baseline checks, not implementation of the pending fixes.

## Questions for the next authoring spec

- What independent requirement-coverage check will reject a parseable but
  incomplete generated ledger before any implementation starts?
- How will the planning worker hand its separate output to a reviewer, retain
  corrections and resume only from an explicitly approved ledger revision?

These questions belong to the next spec; they do not block the four comparison
coding phases or require adding a planner to the current local execution loop.
