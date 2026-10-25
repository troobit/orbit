# Decision Log: Local Compute Agent

## Decision 1: Drive Local Models Through the Existing OpenCode Agent

**Date**: 2026-08-02
**Status**: superseded by Decision 13

### Context

The feature request asked to enable models running on baseCompute, referencing `internal/agents/agent.go` and its subdirectories as the pattern to follow — which implies a new agent package under `internal/agents/`.

Investigation showed baseRT is an inference runtime, not a coding agent. It loads model weights and serves an OpenAI-compatible API, but has no agentic tool-calling loop, no file editing, and no handling of orbit's `/next-task --phase` prompt contract. Every existing orbit agent supplies that missing layer itself.

### Decision

Use the existing OpenCode agent as the agent loop, configured to send its model requests to a local baseRT endpoint. Do not create a new agent package.

### Rationale

The layers are separable: baseRT remains the model runtime and every token still originates from the local model on local hardware, regardless of which component drives it. The only question is who executes the tool calls and edits files. OpenCode already does this, and is the only orbit-supported agent that accepts an arbitrary OpenAI-compatible `baseURL` — the other four are wired to their vendors' APIs. Reusing it reduces the change from roughly 800–1500 lines to roughly 150–350, and avoids re-solving tool-call recovery, which matters more than usual because a locally-sized model produces malformed tool calls relatively often.

### Alternatives Considered

- **Native `internal/agents/basert/` package**: Implement `agents.Agent` directly against baseRT's API - Rejected because it requires writing a complete coding agent from scratch — tool-call loop, tool execution, file editing, and a new session transcript format plus an apsis parser. That work is orthogonal to running on local compute, which is the actual goal.
- **Wire orbit directly at baseRT with no agent loop**: Send the phase prompt straight to the inference API - Rejected because baseRT would return prose describing the work rather than performing it: disk unchanged, no commit, empty variant diff. It would satisfy none of the acceptance criteria while appearing to run.

### Consequences

**Positive:**
- Reuses a proven agent loop, including its recovery from malformed tool calls
- Variant targeting needs no new code: `AssignVariantAgents` already cycles aliases, and `config.go:652` already carries `model` into `AgentConfig.Options`
- Substantially smaller change surface, so the live acceptance run is reachable sooner

**Negative:**
- Adds a dependency on the OpenCode CLI being installed and maintained
- Prompting and tool behaviour are bounded by OpenCode's design rather than orbit's
- Inherits OpenCode's lack of apsis transcript support (see Decision 6)

### Impact

`internal/agents/opencode/`, `.orbit.yaml` configuration, and documentation.

**Correction (2026-08-02):** this entry originally claimed "no changes to the agent registry or variant packages". Review established that is false. `ResolveAliases` validates `type` against `agents.List()` (`internal/config/config.go:690-693,717`), which is populated only by `agents.Register` from package `init()` (`internal/agents/registry.go:29-33`). The distinct agent type in Decision 9 therefore requires a new registration, and so a new package. "No new agent package" in this decision means no new *agent loop* — the driver remains OpenCode — not zero new files.

---

## Decision 2: Do Not Manage or Health-Check the Local Server

**Date**: 2026-08-02
**Status**: superseded by Decision 13

### Context

The local model server must be running before a phase can execute. Orbit could start it, verify it, or ignore it entirely.

### Decision

Orbit neither starts, stops, nor health-checks the local server. The developer runs it themselves, and a run against an absent server fails on the resulting error.

### Rationale

Orbit has never owned a background process, and doing so introduces orphaned-process and port-contention problems — the latter sharply in parallel variant runs, where several variants would target one server. Leaving the server outside orbit's lifecycle keeps that complexity out of the codebase entirely, and one manually-started server naturally serves all parallel variants.

### Alternatives Considered

- **Orbit starts and stops the server via pre-command/post-command hooks**: Use the existing agent shell hooks to bring the server up and tear it down - Rejected because orbit would own a background process it cannot reliably clean up if killed mid-run, and parallel variants would race on the same port.
- **Readiness check without lifecycle management**: Verify the endpoint responds and the model is loaded before phase 1, failing fast otherwise - Rejected because it duplicates diagnostics the first request already provides *for a refused connection*. See the narrowing below: this rationale does not extend to a server that accepts connections but is not ready, and that gap is covered by other decisions rather than by a readiness check.

### Narrowing

Review established that the original rationale claimed more than it supports. A refused connection does fail fast and is diagnosed by requirement [3.1](requirements.md#3.1). A server that accepts the connection while still loading weights, holding the wrong model, or swapping under memory pressure does not fail fast — it simply responds slowly or not at all.

That residual risk is not addressed by this decision. It is bounded by Decision 7 (a 2-hour ceiling terminates the run) and made observable by Decision 11 (output is written incrementally, so the run can be tailed). Without both of those, this decision would leave a run that hangs silently and indefinitely.

### Consequences

**Positive:**
- No process supervision, no port contention, no orphan cleanup in orbit
- A single developer-managed server serves all parallel variants

**Negative:**
- The developer must remember to start the server; forgetting produces a failed run rather than a warning
- Failure diagnostics become load-bearing — this is why requirement [3.3](requirements.md#3.3) classifies unreachable-endpoint errors as fatal rather than retryable, so a down server fails immediately instead of after five backoff attempts
- A reachable-but-not-ready server is diagnosed only by timeout and by reading the incremental output, not by a purpose-built check

---

## Decision 3: Local Runs Execute Without a Timeout

**Date**: 2026-08-02
**Status**: superseded by Decision 7

### Context

The feature request stated local inference should proceed "with no regard for how slow the model will be." Orbit's retry and timeout defaults were tuned for hosted APIs, where a long silence usually indicates a real failure. On local hardware it usually indicates the model is simply working.

### Decision

Phases executed by a locally-backed alias run with no execution timeout. An explicitly configured timeout on the alias is still honoured.

### Rationale

Any fixed ceiling requires guessing a number above "slow" but below "stuck", and a wrong guess kills healthy work — the exact failure the request ruled out. Removing the bound satisfies the requirement directly and needs no new mechanism, since `RunOptions.Timeout = 0` already means no timeout. The explicit-override path remains as the only way to bound a genuinely hung run.

### Alternatives Considered

- **Large explicit timeout (e.g. 8h)**: Set a high per-alias ceiling that normal runs never approach - Rejected because the correct ceiling is unknowable in advance and varies by model and phase size; a wrong value silently kills healthy runs.
- **Idle-based timeout**: Leave total runtime unbounded but fail after a period with no output, distinguishing slow from stuck - Rejected for this iteration because it requires observing output as it streams rather than at process exit, which the current executor does not do. It remains the better long-term answer if hung runs prove to be a real problem.

### Consequences

**Positive:**
- Directly honours the stated requirement with no new mechanism
- No risk of a healthy long run being killed or retried

**Negative:**
- A genuinely hung run blocks until interrupted manually
- Distinguishing slow from stuck is left entirely to the developer watching the run

---

## Decision 4: Report Tokens Without a Currency Figure

**Date**: 2026-08-02
**Status**: superseded by Decision 13

### Context

Local inference has no monetary cost, but orbit's cost reporting assumes a meaningful unit (USD, credits, premium requests). Comparison reports place variants side by side, so a local variant's cost sits directly beside a hosted variant's real spend.

### Decision

Local runs report input and output token counts and display no monetary figure. Comparison reports render the local variant's cost as absent rather than as a zero amount.

### Rationale

`$0.00` is ambiguous in exactly the place it matters most: it reads equally as "this was free" and "this failed before spending anything". Omitting the figure while keeping token counts preserves the useful signal — how much work the model did — without asserting a false equivalence between a free local run and a zero-cost error.

### Alternatives Considered

- **Display `$0.00`**: Treat local runs as zero-cost USD - Rejected because it is ambiguous against a failed run, and places a misleading zero next to real spend in comparison reports.
- **Substitute wall-clock time as the cost unit**: Report elapsed time, since time is the scarce resource locally - Rejected for this iteration as it introduces a fourth cost unit into `internal/cost/`; worth revisiting if local runs become routine.

### Consequences

**Positive:**
- No false equivalence between free and failed runs
- Token counts still convey the work performed

**Negative:**
- Comparison reports carry a gap in the cost column, which needs deliberate rendering
- The real cost of a local run — hours of wall-clock time — is not surfaced as a cost

---

## Decision 5: Spike Before Design

**Date**: 2026-08-02
**Status**: superseded by Decision 8

### Context

Whether a model sized for 64 GB of unified memory can drive an agent loop well enough to complete an orbit phase is unproven. Tool-calling reliability degrades on smaller models, and orbit's contract depends on the model producing usable tool calls consistently enough to edit files and commit. This risk is independent of the driver chosen in Decision 1.

### Decision

Run a throwaway spike before the design phase: pull a model, point the agent loop at the local endpoint, and attempt one trivial task in a scratch repository. Design proceeds only after observing whether files actually changed.

### Rationale

This is the largest unknown in the feature and it can invalidate design assumptions about phase completion, prompting, and error handling. It is cheap to answer — roughly an hour, most of it model download — and answering it late means discovering it during implementation, after the design has committed to assumptions it cannot support.

### Alternatives Considered

- **Write the full spec now, no spike**: Let the first implementation task serve as the de facto spike - Rejected because a negative result would invalidate design decisions already made, forcing rework of design and tasks.
- **Make the spike task 1 of the task list**: Keep the spike but sequence it inside the spec - Rejected because spike findings would then feed into implementation rather than into the design, which is where assumptions about tool-calling reliability actually get made.

### Consequences

**Positive:**
- The riskiest assumption is tested before the design depends on it
- A negative result reshapes the design cheaply, while it is still a document

**Negative:**
- Delays the design phase by the length of the spike, dominated by model download time
- Requires baseRT installed and a multi-gigabyte model pulled before design can start

---

## Decision 6: apsis Transcript Support Is Out of Scope

**Date**: 2026-08-02
**Status**: superseded by Decision 13

### Context

`internal/sessions/` and `internal/transcript/` contain no OpenCode handling, so apsis cannot read OpenCode session transcripts today. Because this feature drives local models through OpenCode (Decision 1), local runs inherit that gap: they produce no viewable transcript in apsis or the `orbit serve` transcript views.

### Decision

Transcript support is out of scope for this feature. The limitation is recorded in the requirements Non-Goals and stated in user-facing documentation.

### Rationale

The gap is pre-existing and belongs to OpenCode support generally, not to local compute — closing it would mean writing an OpenCode transcript parser, a separate piece of work benefiting every OpenCode run. Folding it in would roughly double the feature's scope and delay the live acceptance run, which is the actual completion gate.

### Alternatives Considered

- **Include OpenCode transcript support in this spec**: Write the parser as part of this feature - Rejected because it roughly doubles scope for a benefit unrelated to local compute, and delays the acceptance run.
- **Leave it undocumented**: Ship without mentioning the gap - Rejected because a developer would reasonably expect `apsis` to show their run, and silence turns a known limitation into a bug report.

### Consequences

**Positive:**
- Keeps the feature scoped to local compute
- The limitation is written down rather than rediscovered

**Negative:**
- Local runs produce no viewable transcript, reducing debuggability precisely where runs are slowest and most opaque
- Comparison of a local variant relies on the diff alone, without transcript context

---

## Decision 7: Bound Local Runs at Two Hours Rather Than Leaving Them Unbounded

**Date**: 2026-08-02
**Status**: superseded by Decision 13 (previously superseded Decision 3)

### Context

Decision 3 removed the execution timeout for local runs, reasoning that any fixed ceiling risks killing healthy work. Review then established that three accepted decisions composed badly: with no timeout (Decision 3), no readiness check (Decision 2), and no transcript (Decision 6), every failure mode except a refused TCP connection produces an indefinite hang with no diagnostic and no termination path other than manual interruption.

Separately, `internal/orbit/single.go:771-800` loops on pending tasks with no no-progress detection, so an agent that reports success without doing work causes orbit to re-enter the same phase indefinitely. Unbounded execution turns that from a bug into an unbounded resource drain.

### Decision

A phase run under the local agent type has a default execution timeout of 2 hours, overridable per alias. Comparison run under a local alias uses that same effective timeout in place of its own 30-minute default.

**Correction (2026-08-02):** this decision originally said the bound "applies to comparison and consolidation calls, replacing their 30-minute defaults", which conflated two paths that behave differently:

- **Comparison already honours it.** `internal/orbit/comparison.go:51-54` and `cmd/orbit/compare.go:190-193` already prefer `AgentConfig.Timeout` over `comparison.DefaultTimeout`, so this needs no new plumbing. Requirement [2.6](requirements.md#2.6) now states it.
- **Consolidation cannot honour it today.** `consolidation.Config.Timeout` (`internal/consolidation/types.go:26`) is set by no caller — neither `cmd/orbit/consolidate.go:135-144` nor `internal/orbit/comparison.go:134-141` populate it — so `agentTimeout()` always returns the 30-minute default. Extending the bound there needs plumbing no requirement authorises, so consolidation is **excluded** and keeps its 30-minute default.

### Rationale

Two hours is far above any plausible healthy phase on local hardware and far below "forever", which makes it a usable distinction between slow and stuck without requiring a precise estimate. It preserves the original intent — inference slowness must never fail a run — while restoring an automatic termination path, so a stuck run ends on its own rather than waiting for someone to notice. Making it a per-alias override keeps the escape hatch for genuinely long phases.

### Alternatives Considered

- **No timeout at all (Decision 3, superseded)**: Let local phases run until they finish or are interrupted - Rejected after review showed it composes with Decisions 2 and 6 into a run with no failure signal, no diagnostic, and no termination; the developer left to distinguish slow from stuck has no means to do so.
- **Idle-based timeout**: Leave total runtime unbounded but fail after a period with no output - Still the better long-term answer, but deferred: it requires observing output as it streams. Requirement [6.1](requirements.md#6.1) adds exactly that capability, so this becomes available cheaply in a later iteration.

### Consequences

**Positive:**
- Stuck runs terminate automatically instead of hanging indefinitely
- Slow-but-healthy runs are unaffected, since 2 hours is well above expected phase duration
- Comparison and consolidation no longer silently impose a 30-minute ceiling on a local alias

**Negative:**
- A phase legitimately needing more than 2 hours fails unless the alias overrides the default
- The value is a judgement, not a measurement, until the verification run provides real timings

---

## Decision 8: Scope the Spike to Plumbing Feasibility, Not Model Capability

**Date**: 2026-08-02
**Status**: superseded by Decision 13 (previously superseded Decision 5)

### Context

Decision 5 established a spike before design. Review recommended widening it to measure tool-call reliability across turns, behaviour at the context limit, and whether a locally-sized model can complete a real phase unattended — treating model capability as the feature's central risk.

That framing was rejected. Model capability is a property of the model, not of this integration, and it improves independently of anything decided here. A design gated on today's model quality would be gated on a moving target that the design cannot influence.

### Decision

The spike answers only whether the plumbing can work. It confirms that the agent loop accepts a local endpoint, that orbit receives a usable success signal, and what the runtime's failure output actually looks like. It does not assess the quality or reliability of the model's output.

Spike questions, in order:

1. Does OpenCode accept an arbitrary OpenAI-compatible base URL with no API key?
2. Does its `--format json` envelope survive intact against a custom provider — the envelope being orbit's only success signal (`internal/agents/opencode/agent.go:413`)?
3. Does that envelope carry token usage against a custom provider? The answer determines whether requirement [5.2](requirements.md#5.2) or [5.3](requirements.md#5.3) applies.
4. What verbatim error output appears for: server down, server up but model still loading, unknown model name, model too large for available memory? These strings are what make requirements [3.1](requirements.md#3.1) and [3.2](requirements.md#3.2) implementable.
5. Does `opencode run --format json` write stdout progressively while working, or emit a single JSON document at process exit? Does anything useful reach stderr during the run? Requirement [6.1](requirements.md#6.1) and the idle-timeout deferral in Decision 7 both assume progressive output; nobody has verified it.

### Rationale

Questions 1 and 2 are genuine kill criteria — a negative answer to either means this integration cannot work at all, regardless of model. Questions 3 and 4 produce facts that two requirements currently cannot be written without. All four are properties of the software stack, stable across model choice, and answerable in a short session. Model output quality, by contrast, would need re-testing with every model release and could never be settled.

### Kill and Downgrade Branches

Decision 5 stated only that a negative result would "reshape the design", which assumed the feature proceeds. Explicitly:

- **Question 1 or 2 negative**: the feature is blocked on this driver. Reopen Decision 1 and reconsider the rejected alternatives.
- **Question 3 negative**: requirement 5 downgrades to [5.3](requirements.md#5.3); no token reporting is built.
- **Question 4 yields no distinguishable strings**: requirements [3.1](requirements.md#3.1) and [3.2](requirements.md#3.2) narrow to whatever is distinguishable, and the remainder classify as unknown. Recorded as a limitation rather than worked around.
- **Question 5 negative** (output does not stream): requirement [6.1](requirements.md#6.1) reduces to capture-at-exit, [6.3](requirements.md#6.3) becomes the only liveness signal during a phase, and the idle-based timeout named in Decision 7 stays unavailable — building it on non-streaming output would kill every healthy run.

### Alternatives Considered

- **Widen the spike to model capability**: Measure tool-call reliability over turns, context-limit behaviour, and unattended completion of a real phase - Rejected because model quality is outside this feature's control and improves on its own; gating a design on it measures the model, not the integration.
- **No spike; let implementation discover it**: Proceed straight to design - Rejected because requirements [3.1](requirements.md#3.1) and [3.2](requirements.md#3.2) cannot be written without the verbatim error strings, and questions 1 and 2 are cheap to answer but expensive to discover late.

### Consequences

**Positive:**
- The spike answers stable questions with lasting answers, rather than a snapshot of current model quality
- Two kill criteria are named in advance, so the spike can actually stop the feature
- Produces the concrete error strings that two requirements depend on

**Negative:**
- Whether a local model can complete a phase well remains unknown when the design is written; the verification gate in the requirements is where that surfaces
- A poor result at verification time cannot be blamed on an untested assumption, because the assumption was deliberately not tested

---

## Decision 9: Express Locality as a Distinct Agent Type

**Date**: 2026-08-02
**Status**: superseded by Decision 13

### Context

Review identified two blocking defects with a shared root. First, eleven acceptance criteria conditioned on an alias being "locally-backed" while nothing defined the term — `AgentAliasConfig` (`internal/config/config.go:53-62`) has no endpoint or locality field, so a baseRT-pointed alias is indistinguishable from a hosted one. Second, making connection-refused fatal would flip that behaviour for every OpenCode run, contradicting the Non-Goal protecting hosted agents, because classifiers are registered per agent type and `Classify()` receives no alias context.

### Decision

Locally-served models are a distinct agent type, named in an alias's `type` field, separate from the existing `opencode` type.

### Rationale

One change resolves both defects. The type is already an observable field in `.orbit.yaml` and is already validated against the registered agent types (`internal/config/config.go:690-693,717`), so every conditional acceptance criterion gains a concrete antecedent. A separate type carries its own error classifier and its own timeout default without affecting hosted OpenCode runs. Type is also the natural place for the differing default timeout in Decision 7.

**Correction (2026-08-02):** this rationale originally claimed classifier lookup keys on the configured agent type at three call sites, and that no registry change was needed. Review established both claims are wrong, and the second defect is load-bearing:

- Only `internal/orbit/orbit.go:241` resolves the classifier from the config type. `internal/orbit/variants.go:879,981` and `internal/consolidation/consolidator.go:547` resolve it from `agent.Name()`, which is the hardcoded literal `"opencode"` (`internal/agents/opencode/agent.go:52`).
- Consequently, if the local type simply reuses `opencode.New`, then **in variant mode it resolves the OpenCode classifier**, whose connection handling is retryable (`internal/agents/opencode/errors.go:65-67`). That would violate requirement [3.3](requirements.md#3.3) in precisely the mode requirement [1.3](requirements.md#1.3) targets — a down server burning five backoffs instead of failing fast.
- A new registration is required (see the correction on Decision 1).

Requirement [3.3](requirements.md#3.3) now states the single-run and variant modes explicitly so this cannot be satisfied in one mode only. How the local agent reports its name, and whether the local classifier is a copy, a delegating wrapper, or a shared implementation parameterised on locality, are design questions this decision deliberately leaves open.

### Alternatives Considered

- **A `local: true` boolean on the alias**: Add a locality flag to `AgentAliasConfig` - Rejected because classification and timeout selection would still need that flag threaded through interfaces that currently receive only the agent type, a wider change for the same outcome.
- **A `base-url` field on the alias**: Infer locality from the presence of a local endpoint - Rejected because the endpoint genuinely lives in the agent loop's own configuration; duplicating it into orbit creates two sources of truth that can disagree.

### Consequences

**Positive:**
- Gives every conditional acceptance criterion a concrete, observable antecedent
- Separate error classification and timeout defaults with no risk to hosted OpenCode runs
- No change required to the agent registry or retry layer

**Negative:**
- Two agent types share one underlying implementation, which must be kept in step
- A user pointing a plain `opencode` alias at a local endpoint gets hosted defaults, and the mismatch is silent

---

## Decision 10: Stop the Run After Two Consecutive No-Progress Phases

**Date**: 2026-08-02
**Status**: superseded by Decision 13

### Context

The phase loop (`internal/orbit/single.go:771-800`) is gated only on the pending task list. An agent that returns a well-formed response having changed nothing produces no error, so orbit records the phase as successful, the task list is unchanged, the next-phase lookup returns the same phase, and the loop re-enters it. No no-progress detection exists in either the single-run or variant path.

This is orbit control flow, not model quality, so it remains in scope under Decision 8.

### Decision

A phase that completes without error and leaves the set of pending task identifiers unchanged is recorded as making no progress. Two consecutive such phases stop the run with a report naming the phase and the count. Detection applies in both single-run and variant mode, and only to runs using the local agent type with the default task-advancing command.

**Correction (2026-08-02):** the original wording said "pending task list unchanged" with no comparison predicate and no scope, which review showed was both untestable and a regression risk:

- *Predicate.* An agent that completes one task and appends a follow-up leaves the count identical but the identifier set different. The criterion now compares the **set of pending task identifiers**, not the count.
- *Blast radius.* `Command` is user-overridable via `--command` and `ORBIT_COMMAND`. Any command that is not the task-advancing one — a review pass, a docs pass — never mutates the pending list, so every phase would read as no progress and abort the run after two. That would have been a regression for existing hosted users, introduced by a spec about local compute. Requirement [4.4](requirements.md#4.4) now scopes detection to the local agent type and the default command.
- *Variant mode.* The variant phase loop (`internal/orbit/variants.go:616-720`) calls only `GetNextPhase()` and never `ListPending`, so the chosen signal was wired in the single-run path alone — the opposite of where it matters, since variants are where local phases run for hours and `--parallel` multiplies the waste. Requirement [4.3](requirements.md#4.3) now requires both modes.

### Rationale

Two consecutive occurrences distinguishes a genuine stall from a single phase that legitimately had nothing to do. Detection uses the pending task list, which orbit already queries each iteration, so no new signal is required. Without it, a bounded timeout only bounds each individual phase — the loop itself could still repeat indefinitely, consuming two hours per iteration.

### Alternatives Considered

- **Stop after one no-progress phase**: Fail immediately - Rejected because a single phase can legitimately complete with no pending-list change, making false positives likely.
- **No detection; rely on the timeout**: Let each phase hit its 2-hour bound - Rejected because the timeout bounds a phase, not the loop; repeated no-progress phases would each consume the full bound.

### Consequences

**Positive:**
- Closes the unbounded-loop path that the timeout alone does not cover
- Uses a signal orbit already has

**Negative:**
- A workflow that legitimately produces two consecutive no-change phases stops early
- Adds a stopping condition that applies to all agents, not only local ones

---

## Decision 11: Capture Agent Output Incrementally

**Date**: 2026-08-02
**Status**: superseded by Decision 13

### Context

`internal/agents/executor.go:69-71` binds stdout and stderr to in-memory buffers and calls a blocking run; logs are written only after a phase returns (`internal/logs/manager.go:585-603`). `orbit status` last-action supports only claude-code and kiro, and the spinner is single-run TTY only, so variant mode shows nothing. There is no signal between phase start and phase end. With phases now permitted to run for two hours, a working run and a stuck one are indistinguishable for that entire period, and an interrupted run leaves nothing on disk.

### Decision

Agent output is written incrementally to a file during execution, and retained if the run is interrupted.

### Rationale

This is the minimum that makes a long local run observable, and it does not reverse Decision 6: a raw output file is not a rendered transcript — no session discovery, no parser, no apsis viewer. It is also the exact prerequisite that Decision 7 cited when deferring the idle-based timeout, so building it now makes that option available later at no additional cost. Because it touches shared execution code rather than local-specific code, it needs to be a requirement now rather than a discovery during implementation.

### Alternatives Considered

- **Rely on apsis transcripts**: Close the transcript gap instead - Rejected as Decision 6 already scoped that out; it is a larger piece of work benefiting all OpenCode runs.
- **Progress heartbeat only**: Print elapsed time periodically without capturing output - Rejected because it shows that time is passing but not what the agent is doing, and leaves nothing behind after an interruption.

### Consequences

**Positive:**
- A long run can be tailed to distinguish progress from a hang
- Output survives interruption, which is the expected termination path for a stuck run
- Unlocks the idle-based timeout as a cheap future option

**Negative:**
- Changes shared execution code, so it affects all agents and needs regression coverage
- Produces an additional output file per phase, with its own disk-usage and cleanup considerations

---

## Decision 12: Document the Existing Comparison Agent Split Rather Than Change It

**Date**: 2026-08-02
**Status**: superseded by Decision 13

### Context

Comparison and consolidation invoke an agent themselves, sending up to 3000 characters each of requirements, design and tasks, plus commit messages and diff stats. Which agent receives that content determines whether a "local" run is actually local end to end.

An earlier draft of this decision asserted that both always use the configured default agent. Review established that is wrong for the primary workflow, and that two paths behave differently:

- **In-run comparison follows `--agent`.** `internal/orbit/comparison.go:58` passes `o.agent`, which is built from the run's `--agent` flag (`internal/orbit/orbit.go:216-236`). So `orbit run --agent <local-alias> --variants 2` **already** runs comparison on the local agent and already inherits its timeout.
- **Standalone commands follow the configured default.** `cmd/orbit/compare.go:181` and `cmd/orbit/consolidate.go:119` both call `resolveAgent("", appConfig)` (`cmd/orbit/run.go:465-473`), which resolves the default regardless of what the original run used.

### Decision

Neither path changes. The split is documented as it actually behaves, in requirement [7.6](requirements.md#7.6).

### Rationale

The in-run path already does the privacy-preserving thing, so no change is needed there. The standalone path sending content to a hosted default is a genuine gap, but closing it means adding an agent override to commands used by everyone, and its value depends on whether a local model can produce reliable structured JSON — exactly the model-capability question Decision 8 removed from scope. Documenting the real behaviour gives a developer with a strict locality requirement an accurate picture: in-run comparison is safe if the run is local, standalone commands are not.

### Alternatives Considered

- **Add an agent override to the standalone commands**: Let `orbit compare` and `orbit consolidate` target a local alias - Rejected as separate work affecting all users, whose value depends on model capability at structured output that this feature deliberately does not assess.
- **Make the standalone commands infer the agent from the run's recorded metadata**: Read it back from `summary.json` - Rejected because variants may use different agents, so there is no single correct inference, and a stale or absent summary would silently change which agent runs.

### Consequences

**Positive:**
- The documented behaviour matches the code, including for the primary `orbit run --variants` workflow
- No change to commands used by all agents, local or not
- In-run comparison under a local alias is already local, at no cost

**Negative:**
- Standalone `orbit compare` and `orbit consolidate` still send repository content to the configured default agent, which may be hosted
- The locality guarantee is workflow-dependent rather than absolute, which is harder to explain than a flat rule
- A developer needing strict locality must avoid the standalone commands specifically, not comparison generally

---

## Decision 13: Replace the prototype with an Orbit local preset

**ID**: 13
**Date**: 2026-10-25
**Status**: accepted

### Context

The user requested uplifting and testing the recent agentic-coding
`local-next-task` work in Orbit. That prototype already uses Codex's native
LM Studio provider. The earlier draft's assumption that only OpenCode supports
local inference is obsolete. The prototype duplicates Orbit's session loop,
timeouts and logging, but has selection and halt checks Orbit needs to retain.
This decision replaces the earlier unimplemented requirements and Decisions
1–12 for this deliverable. Existing supersession links remain historical.

### Decision

Expose `orbit local` and a `local-next-task` compatibility command as presets
of the existing Orbit loop and Codex agent. Orbit selects one task by default;
phase and model-driven selection are explicit switches. Preserve the
prototype's 30-minute timeout and 20-iteration defaults. Stop on the first
failed or unproductive iteration and leave edits for review.

### Rationale

Orchestration belongs in Orbit. Explicit task dispatch reduces tool turns for
local models. A separate preset isolates these behavioral defaults from hosted
runs and needs no new agent type or inference client.

### Alternatives Considered

- Keep the Python runner and call it from Orbit: rejected because it retains
  two lifecycle implementations and separate logging/recovery behavior.
- Build a new local agent type or direct inference tool loop: rejected because
  Codex already supplies LM Studio integration and tool execution.
- Change every hosted run to this contract: rejected because custom hosted
  prompts and commit workflows have different completion semantics.

### Consequences

Positive: one orchestration loop, deterministic dispatch, retained diagnostics,
and existing Orbit summaries and registry integration.

Negative: recent Codex flags and an installed rune CLI are prerequisites;
local model quality and latency remain variable. Windows only has direct-process
cancellation. Model selection mode requires the installed next-task skill.

---

## Decision 14: Restrict local capabilities and document protocol compatibility

**ID**: 14
**Date**: 2026-10-25
**Status**: accepted

### Context

LM Studio warns that Codex 0.164.0 sends unsupported `namespace` and
`web_search` tools, a cache key, developer roles and an encrypted reasoning
include. Capturing requests against a loopback stub reproduced each field
with the same local CLI flags. The namespace contains delegation tools despite
Orbit's local prompt prohibiting delegation.

### Decision

Disable web search, multi-agent tools and reasoning summaries in the local
preset. Document the remaining cache-key, role-conversion and encrypted-include
warnings as version-specific compatibility limits. Retain project instructions
and the existing Codex tool loop.

### Rationale

Explicit capability settings remove tools the server drops. The remaining
fields originate inside Codex; captures show that disabling summaries does not
remove the encrypted include, and the OSS path ignores a false
`model_supports_reasoning_summaries` override. Request metadata does not by
itself establish that task context is lost.

### Alternatives Considered

- Document every warning without changing the preset: rejected because web
  search and delegation can be disabled and do not belong in this local loop.
- Rewrite requests in a compatibility proxy: rejected because owning streaming,
  tool schemas and instruction-role conversion is unnecessary for this fix.
- Remove Codex/project instructions: rejected because role conversion does not
  establish that their content is useless; they include execution constraints.

### Consequences

Positive: fewer unsupported capabilities, less tool/instruction overhead, and
setup guidance that identifies what the preset can actually control.

Negative: local tasks have no built-in web search or delegation, some server
warnings remain, and future Codex/LM Studio versions need another wire check.

---

## Decision 15: Budget local context and separate environment failures from progress

**ID**: 15
**Date**: 2026-10-25
**Status**: accepted

### Context

LM Studio removed intermediate history from a 33340-token request against a
32768-token loaded context. Code edits continued, but missing dependencies and
sandbox networking blocked validation. Orbit correctly failed the iteration for
no ledger progress, while its summary incorrectly counted the failed phase.
The user asked to consider smaller tasks and alternate harnesses, then clarified
that extra terminal-only tool activity display was unnecessary.

### Decision

Pin configurable context and compaction budgets, bound retained tool output,
offer explicit sandbox network opt-in, improve blocker/validation guidance and
correct failed-attempt accounting. Keep the current Codex driver for this fix;
document smaller top-level work units, larger loaded context and OpenCode as
options to evaluate. Do not rewrite task ledgers or change server loading and
overflow policy automatically. Leave terminal activity presentation unchanged.

### Rationale

A loopback check with installed Codex 0.164.0 and 22000 reported input tokens
triggered a summary request under the 20480 threshold, followed by a regular
tool-enabled request. These controls work without adopting a new harness.
Dependency access is independent of inference routing or task granularity, and
saving diagnostics must not imply that a phase passed validation.

### Alternatives Considered

- Only shrink tasks: useful, but rejected as the sole fix because it cannot
  satisfy missing dependencies or prevent an oversized read from overflowing.
- Increase the loaded context only: retained as an option, but rejected as the
  default because it requires resources, a reload and corresponding client limits.
- Switch immediately to OpenCode: retained for a measured comparison; rejected
  for this fix because the current adapter expects one JSON document and lacks
  local live-output/cancellation forwarding. No equivalent live-model comparison
  has established better task completion.
- Build a direct LM Studio harness: retained as a future option; rejected here
  because it requires owning tool execution, history management and recovery.

### Consequences

Positive: explicit budgets, fewer unbounded reads, clearer dependency blockers
and accurate failed-attempt summaries, with the existing orchestration intact.

Negative: budgets depend on operator-supplied loaded context, compaction remains
lossy and tokenizer estimates can differ. Tool output retained in history is
limited; raw artifacts remain available. Network access is still opt-in and
does not make external caches writable. Alternate harnesses remain unmeasured.
