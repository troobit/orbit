# Spec Drive: apsis-shared-json and apsis-lmstudio-logs

This file records the working process. Any agent can pick it up cold: read the state, perform useful work, verify it, and log the result. The human authorised continuous process improvement, spec refinement and local implementation on 2026-11-05. The continuous-work rules below override conflicting older rules. Delete this file and every `specs/*/drafts/` folder before opening an upstream PR.

## Continuous refinement and build

- Continue through useful spec, task, implementation and verification work without stopping after one step. Improve this process when an observed failure justifies a concrete change; record why. Do not manufacture work or waits to consume tokens or attempt to control token resets.
- Use the referenced skills' delegation workflows. Fresh-context subagents may draft, review or implement disjoint files in this worktree. Report same-model reviews as fresh-context reviews, not as independent-model evidence. Prefer a different harness/model when available, but their absence does not block local progress.
- One grounded draft plus a fresh-context review can advance a phase. Preserve additional drafts when available. An author cannot supply its own independent review; the coordinator may revise its work using another agent's review.
- Preserve explicit human decisions. Routine reversible technical decisions may be admitted for local implementation after review and recorded as agent decisions. This is not human approval. Ask only for material user-owned behaviour changes; continue independent work while an answer is pending. Publication and git commits/pushes remain outside this authorisation.
- Maintain requirements, design and rune tasks before implementing each slice. Local implementation can precede formal design/tasks approval when the coordinator records the admission and its review evidence. Reconcile documents with tested behaviour and keep unfinished tasks open.
- Step file lists limit concurrent ownership, not the entire workflow. Claimed implementation work may edit the named source/tests/schema files and relevant notes. Keep all writes in this worktree, use Makefile checks, and retain append-only status logs.
- A claim with no completion, no output files and no recorded progress for 30 minutes may be recovered after rechecking the log and outputs. Append a release explaining the recovery, preserve every existing artifact, then claim the work. Recheck before writing; retain concurrent contributions instead of overwriting them. Claims with active output changes are not recoverable on elapsed time alone.
- Build/test commands may fetch declared repository dependencies. Machine-wide installations, other checkouts, branch changes, pushes and external messages remain prohibited; commits are limited to rule 4's orbit exception. Use synthetic fixtures only; never persist private machine data.
- Stop at a meaningful verified checkpoint only when further useful work requires user input, unavailable external state or a fresh review that cannot currently be scheduled. Report completed work and the precise next action; never claim an unperformed review or test.

## Where things are

- Worktree: `~/repos/orbit/.worktrees/apsis-codex-exec-lmstudio`, branch `feature/apsis-shared-json`, based on upstream `ArjenSchwarz/orbit` main. All paths below are relative to it.
- Specs, in dependency order:
  1. `specs/apsis-shared-json/`: shared JSON output, Codex exec streams, model provider, listing across projects.
  2. `specs/apsis-lmstudio-logs/`: LM Studio server logs as an input. Builds on spec 1.
- Each spec has `handoff.md` (verified facts and direction; read it first), its documents (`requirements.md`, `design.md`, `tasks.md`, `decision_log.md`) and `drafts/` (working files and `drafts/status.md`, the log).

## What is settled

The aim is the best decisions and requirements, reached by independent agents. Everything in these specs so far came from one long Claude Code (Opus 5.5) session, the subagents it started, and one Codex draft. Treat all of it as a proposal to challenge, except what the human decided:

- `apsis-shared-json`: LM Studio moves to its own spec (Decision 5); apsis reports the provider only, with no local flag (Decision 6); apsis reads no prompt file beside exec streams (Decision 4); the spec is named `apsis-shared-json` (Q10).
- The direction listed in each `handoff.md` under "Direction from the user".
- From 2026-11-05, orbit drives `apsis-shared-json` implementation, one phase per upstream PR slice, with local commits and no pushes. Coordinated implementation steps for that spec are paused; review moves to each PR slice before it goes upstream. Spec revisions for both specs continue under these rules.

Facts in `handoff.md` were checked once by one agent; `note` lines in the logs record the ones found wrong since.

## Rules for every agent

1. Your agent name is `<harness>-<model>` in lower case, for example `claude-code-opus-5.5` or `codex-gpt-6.1-sol`. Two agents are independent when their harness or their model differs. Where the process asks for independent agents, prefer another harness and model family over another model on the same harness. Log lines written before this rule name the harness only: `claude-code` lines came from Claude Code with Opus 5.5 and its subagents, and the `codex` model was not recorded.
2. Work only inside the worktree. Never modify `~/repos/orbit` itself (it holds uncommitted work on another branch) or any other worktree.
3. Respect claimed file ownership and the continuous-work rules; append status lines without rewriting earlier entries.
4. No pushes or branch changes. Commits happen only through `orbit run` of `specs/apsis-shared-json/tasks.md` on `feature/apsis-shared-json` (human decision, 2026-11-05); agents outside orbit make no commits. No machine-wide package installs. Web requests read public documentation, apart from declared build/test dependency retrieval.
5. The repository is public. Never copy log content, identifiers, home-directory paths or the names of locally served models from this machine into any file.
6. This feature must merge upstream on its own. Unmerged local-execution work (`orbit local`) must be able to build on it later, never be required by it.
7. Decision logs use supersede mode (the repo has no `.agentic.json`).
8. Continue through useful eligible steps under the continuous-work rules, reporting verified progress.

## Reference files

Read the one for your step as plain instructions. Adapt harness-specific tools to native tools. Use fresh-context review/delegation under the continuous-work rules; avoid redundant approval prompts for already authorised local work.

- Requirements format: `~/.claude/skills/starwave-requirements/SKILL.md`
- Design format: `~/.claude/skills/starwave-design/SKILL.md`
- Tasks format: `~/.claude/skills/starwave-tasks/SKILL.md` and `~/.claude/skills/rune/SKILL.md` (tasks must parse with `rune list`)
- Critique method: `~/.claude/skills/design-critic/SKILL.md`
- Decision log format: `~/repos/agentic-coding/claude/rules/references/decision-log-format.md`
- Overview: `~/.claude/skills/specs-overview/SKILL.md`
- Repo conventions: `CLAUDE.md` (build with `make`, tests must assert behaviour)

## Phases and steps

Each spec goes through three phases in order: `requirements`, then `design`, then `tasks`. Each phase runs these steps in order. `<phase>` is the phase name; files go in `drafts/<phase>/`.

| # | Step | Who may do it | Writes | Complete when |
|---|------|---------------|--------|---------------|
| 1 | `draft` | Any agent, at most one draft per agent. Do not read other drafts. | `draft.<agent>.md`, `decisions.<agent>.md` | Two drafts from independent agents are done |
| 2 | `cross-review` | Any agent that did not write the draft it reviews | `cross-review.<agent>.md`: per item, which version to keep and why; conflicts for the user; gaps in all drafts | Every draft has a cross-review by a non-author |
| 3 | `merge` | Any one agent | the spec's `<phase>.md`, `decision_log.md`, `drafts/<phase>/questions.md` | One `done` |
| 4 | `critique` | Any agent | `critique.<agent>.md`, using the critique method | One `done` for the current revision |
| 5 | `peer-review` | An agent independent of the critique's. Do not read the critique. | `peer-review.<agent>.md`: an independent review of the document | One `done` for the current revision |
| 6 | `revise` | Any one agent | the spec's `<phase>.md`, `decision_log.md`, `questions.md` | One `done` |
| 7 | `approve` | Any agent with a human present | the spec's `<phase>.md`, `decision_log.md` | The human approves |

Steps 4 and 5 run at the same time and independently. Reviews are under 900 words, ranked critical, major, minor, each with the item number and a concrete fix. Steps 4 to 7 count only log lines after the phase's latest `merge | done` or `approve | changes`. Review files from an earlier revision are kept by appending `.r<N>` before `.md` (for example `critique.codex.r1.md`).

- `revise` weighs the critique and peer review against each other and against the code.
- `merge` and `revise` resolve everything the drafts, cross-reviews and reviews agree on. Anything that needs the human goes into `questions.md` with options and a recommendation. Every decision is recorded in `decision_log.md`.
- `merge` and `revise` also apply every `note` line logged since the phase's previous `merge` or `revise`, after checking it, and may correct facts in the spec's `handoff.md`. They list the notes they applied in their `done` line.
- `approve`: ask the human the questions in `questions.md`, one at a time. Record each answer in `decision_log.md` and apply it. Then ask whether the document is approved, offering approve, request changes, or explain it first. An approval logs `approve | done`. Requested changes are applied and logged as `approve | changes`. The phase then returns to step 4, and steps 4 to 7 run again on the new revision.
- A headless agent, or one without a human present, never does `approve`.
- After a spec's `tasks` phase is approved, any agent may do `overview`: regenerate `specs/OVERVIEW.md` and confirm `rune list specs/<spec>/tasks.md` parses.

## The status log

Each spec's `drafts/status.md` ends with a `## Log`. Lines are only ever appended:

```
- <UTC time, e.g. 2026-11-04T13:30Z> | <agent> | <phase> | <step> | <claimed|done|skipped|released|changes|note> | <note or output files>
```

- A `note` line records a fact that a document or `handoff.md` gets wrong, found outside a step that may fix it. It needs no claim. Use `-` as the step and say who found it and whether it was checked.

- Append with one shell append per line, for example `printf '%s\n' '- ... | claimed | ...' >> specs/<spec>/drafts/status.md`. Never open a status file in an editor or rewrite it: several agents append at once, and a rewrite erases their lines.
- Before starting a step, append `claimed`, then read the log again. If an earlier line holds a live claim on the same step (or, for step 1, a claim from your own harness), you lost the race: append `released | lost to <agent>` and choose again.
- When finished, append `done` with the files you wrote.
- A `claimed` line with no matching `done` after 3 hours is stale. Append `released | stale` and take the step.
- `skipped` (with a reason) counts as complete.
- If you stop partway, append `released | <what is left>`.
- `specs/OVERVIEW.md` is shared by both specs. Before an `overview` step, check both logs for a live `overview` claim.
- Do not review a revision you wrote in the same session; leave that step to a fresh agent.

## Choosing your step

1. Read this file, then each spec's `handoff.md` and `drafts/status.md`.
2. List the open steps you are allowed to do, for the current phase of each spec. A step is open when it is incomplete, all earlier steps of the phase are complete, and nobody holds a live claim on it. Step 1 is the exception: it stays open to other harnesses while one draft is claimed. Steps 4 and 5 are both open once step 3 is done or after `approve | changes`.
3. Take the earliest by step number. On a tie, take `apsis-shared-json` first.
4. If the older sequence has no open step, apply the continuous-work rules: schedule a fresh review, recover an abandoned claim where justified, or admit a reviewed local implementation slice. Stop only when no useful authorised action remains.

## Changes to this process

This file changes only when the human asks. Agents note each change here.

- 2026-11-04T13:36Z | claude-code | Status logs are append-only through shell appends; claim races go to the earlier line; one `overview` at a time; no reviewing a revision you wrote in the same session.
- 2026-11-04T13:39Z | claude-code | Added `note` log lines for facts found outside a step that may fix them; `merge` and `revise` apply them and may correct `handoff.md`.
- 2026-11-04T13:42Z | claude-code-opus-5.5 | Agent names are `<harness>-<model>`; independence counts a different harness or model, preferring another harness and model family; added "What is settled" so agents treat everything except the human's decisions as a proposal.
- 2026-11-04T14:58Z | codex-gpt-6 | Human authorised overriding and improving the process while refining and building the specs. Removed one-step stopping and unavailable-model deadlocks; admitted honest fresh-context review, delegation, recoverable abandoned claims and reviewed local implementation, retaining privacy and no-publication boundaries.
