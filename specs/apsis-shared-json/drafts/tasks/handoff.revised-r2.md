# Handoff: apsis-shared-json

Context for whoever works on this spec next. Delete this file before opening the upstream PR. The process and current state are in `SPEC-DRIVE.md` at the worktree root and in `drafts/status.md`.

## Where things are

- Worktree: `~/repos/orbit/.worktrees/apsis-codex-exec-lmstudio` (the folder kept its first name), branch `feature/apsis-shared-json`, created from upstream main at `7259a6c`. Upstream tracking was removed on purpose so a stray `git push` cannot target ArjenSchwarz/orbit. Nothing is committed yet.
- `.worktrees/` was added to `.gitignore` in the main checkout (`local-compute` branch), uncommitted. Leave that checkout alone; it holds other uncommitted work.
- Files: `requirements.md` revision 3 is human-approved (Q25–Q28). `design.md` incorporates completed non-author reviews and the tasks-merge technical corrections (Q63–Q79, Decisions 25–28); neither design nor tasks has human approval. `tasks.md` now merges both drafts into four PR phases, sixteen parents and fifteen adjacent red/green pairs. Original numbered/stable IDs 1–11 and every status/owner survive; task 1 is completed under its original `codex-gpt-6` owner, supported by the separate 22:28:08Z verification record.
- Both task drafts and their non-author cross-reviews remain under `drafts/tasks/`. The byte-exact pre-merge canonical file is `tasks.canonical-before-merge-r1.md`, with scope relocations in `merge-map.codex-gpt-6.1-sol.md`. Historical draft metadata is not an execution lock. Exact earlier design/decision/question snapshots remain under `drafts/design/`.
- Sibling spec `specs/apsis-lmstudio-logs/` builds on this one.

## Direction from the user

- Frame the work as expanding apsis's translation: harness formats in, one shared representation out, rendered as Markdown, HTML and now JSON. New sources and fields must render in Markdown and HTML like existing ones.
- Name mechanisms explicitly (Codex `exec --json` streams). Vague naming was rejected.
- Code must be defensible, toolset-agnostic and good enough for upstream maintainers to merge. Keep personal and machine-specific details out of the spec, code and fixtures; the repo is public.
- The feature must stand on upstream main alone. Unmerged local-execution work in a fork (`orbit local`, which writes `<base>.stdout.jsonl` and `<base>.prompt.txt` under `.orbit/local/`) must be able to build on it, not be required by it.
- No agent or harness is the hero; the process is.
- Upstream conventions: small PRs, changelog entries under `[Unreleased]` (Keep a Changelog), ADR-format decision log, CLAUDE.md testing rules (no assertion-free tests), `make test` and `make lint`.

## Suggested PR slices, in order

1. Shared JSON output: the new `-f` value, a layout independent of any harness's wire format, an integer schema version, the JSON Schema file, and the Codex provider field.
2. Listing across projects: the option for `apsis --list` and `apsis serve`, opening listed sessions from any directory, and JSON listings with warnings.
3. Codex exec streams: the parser, generic events, configured-directory discovery, project assignment by containing repository, and rollout de-duplication.
4. Unchanged-refresh cache: platform metadata adapters and the Lister cache. Native Windows evidence gates only this slice; requirement 5.9 remains unfinished until it passes.

`apsis-lmstudio-logs` follows as its own PRs.

## Verified facts (2026-11-04/05)

**Codex CLI 0.166.0 `codex exec --json`**
- Across 59 local streams: events `thread.started` (has `thread_id`), `turn.started`, `item.started`, `item.completed`, `turn.completed`. Items seen: `agent_message` (`text`), `command_execution` (`command`, `exit_code`, `aggregated_output`, `status`), `file_change`, `error` (`message`). Events carry no timestamps.
- `turn.completed.usage` includes `input_tokens`, `cached_input_tokens`, `cache_write_input_tokens`, `output_tokens` and `reasoning_output_tokens`. The pinned [0.166.0 JSONL emitter](https://raw.githubusercontent.com/openai/codex/rust-v0.166.0/codex-rs/exec/src/event_processor_with_jsonl_output.rs) copies reported `ThreadTokenUsage.total` counters rather than calculating per-turn deltas. Preserve the snapshots and do not sum them as independent turn usage.
- The non-interactive guide lists item families without defining every payload. The pinned [0.166.0 exec definitions](https://raw.githubusercontent.com/openai/codex/rust-v0.166.0/codex-rs/exec/src/exec_events.rs) establish specialised fixture shapes; label synthetic fixtures by their definition source rather than claiming they were captured.
- The stream holds no user prompt, model, provider or `session_meta`. With `--ephemeral` no rollout is written to `~/.codex/sessions`.
- `codex exec --local-provider` accepts `lmstudio` or `ollama`.
- The same Codex thread ID appears as the exec stream's `thread_id`, the rollout's `session_meta.payload.id`, and `client_metadata.thread_id` in requests Codex sends to a model server. The pinned [0.166.0 resume path](https://raw.githubusercontent.com/openai/codex/rust-v0.166.0/codex-rs/exec/src/lib.rs) reuses the resumed thread ID; the JSONL emitter emits it again. A thread ID alone therefore does not identify a stream file.
- Interactive Codex rollouts record `session_meta.payload.model_provider` (`openai` or `lmstudio`) and `originator`; apsis keeps the model but not the provider today.
- Upstream orbit runs `codex exec` without `--json`, so upstream orbit writes no exec streams.

**apsis code (upstream main)**
- Sources live in `internal/sessions/types.go` (`claude`, `codex`, `copilot`, `kiro-cli`, `kiro-ide`). `SessionInfo` has no project path.
- Format detection is in `internal/transcript/parser.go` (`DetectFormat`, `detectFormatFromLine`). `-a` values are `claude-code`, `codex`, `kiro`, `kiro-ide` and `copilot`. `-f json` is raw session data and must not change.
- `transcript.Entry` carries Claude's wire-format JSON tags and `Model` is `json:"-"`, so the JSON output needs its own layout.
- `apsis serve` always scopes to one project: an empty `--project` resolves to the current directory (`cmd/apsis/main.go` `resolveProjectPath`). Internally an empty project path lists everything (`Lister.ListAll("")`), but no flag exposes it.
- Opening by ID searches all Claude projects only when no project path is set; the CLI always sets one.
- Follow mode (`-F`) supports Markdown only and unmarshals each line as a Claude entry.
- apsis has no config file; `apsis serve` reads `APSIS_SERVE_*` environment variables.
- Current `convertCopilotToEntries` preserves user/assistant-turn timestamps and tracks `session.model_change`; `convertKiroUserMessage` and `convertKiroAssistantMessage` preserve available timestamps/model; Kiro IDE conversion sets the session start timestamp on the first entry and preserves the assistant model. `docs/agent-notes/transcript-metadata.md` now records that coverage and renderer lookup scope; use current code when planning additional extraction.

## Design questions and local admission

U1 (additional Claude model extraction), U2 (Kiro IDE provider) and U3 (test-only validator) remain unanswered human questions. Keep current omission for U1/U2 under approved scope. U3's maintained local-only validator is a reversible technical admission, not a human answer. The initial schema includes every known v1 branch and an unknown-type extension branch; later producers must validate against the exact original schema. New header-only Codex success and model-change splitting are enabled only by explicit shared-JSON parse options, so legacy provider-less errors/grouping remain protected. `ParseResolved` must forward both flags, preserve explicit CLI forced-format precedence and merge the source-bound Kiro IDE cost path. Kiro warning positions distinguish chat messages from execution actions; only the outer Codex session_meta timestamp supplies parsed creation time.

The checked 19:00Z log note corrects earlier freeze attribution: the dispatching coordinator set an ownership freeze to protect idle unlogged outputs; the human did not request it. SPEC-DRIVE authorises continued reviewed local planning and implementation without commits, pushes or publication. Preserve concurrent owners and explicit human decisions. The fresh task critique found no defects and peer review found one helper-option forwarding gap. That gap, the warning-position correction and model-grouping precision are now revised; a non-author follow-up must assess them before the coordinator admits task 2. No human design/tasks approval is implied.

## Pending verification

- Requirements revision 3 remains approved; earlier requirements questions are historical and must not be asked again.
- The design's 18:26Z reviews were separate non-author contexts of the same model, not independent-model evidence from each other. Their revision checks cleared the earlier reviewed scope. The task drafts now each have a non-author cross-review, including the independent-harness/model review of the Claude draft. The completed task reviews are same-model fresh non-author evidence. The current revision needs a non-author follow-up on the identified contracts; its author must not supply it.
- Baseline repair is separately verified in the status log: the two remaining receives now assert their errors, earlier assertions are retained, and Makefile lint/build/tests pass (full test on unchanged retry after the known web shutdown flake). Its coordinator then completed task 1 at 22:28:08Z while preserving the original owner.
- Native Windows fingerprint execution is unavailable and unverified. Follow `prerequisites.md`; task 15.2 cannot complete and task 16 cannot start without passing native evidence. Cross-compilation is insufficient. PRs 1–3 remain eligible; no CI change or weakened 5.9 is authorised.
- Module writes are serialized: task 15.2 follows task 3.2 for go.mod/go.sum. Fingerprints live in the isolated `internal/fileidentity` package; cache consumers remain in `sessions`.
- The logged monthly-volume claim was corrected to a nine-month sample; Decision 7 retains the human-approved separate LM Studio spec.

## Related work

The consumer is the `dreaming` spec in `~/repos/agentic-coding/specs/dreaming/`; its section 3 depends on this spec (dreaming Q21), and it correlates sources itself.
