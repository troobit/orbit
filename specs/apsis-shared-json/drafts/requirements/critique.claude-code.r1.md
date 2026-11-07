# Critique (revision 1): claude-code

Reviewed `requirements.r1.md` (before the LM Studio split) with the design-critic method. Code claims were checked on upstream main. AC numbers refer to revision 1.

## Critical

- **C1 (2.3, 2.2, 2.5).** No definition of an LM Studio session: "documented, deterministic rule" and "unambiguously" are circular, so 2.3 is untestable. Missing: a stable session identifier, a rule for history every request repeats, and an outcome for warnings and errors when up to 4 parallel slots share one model tag. Fix: state the rule as observable behaviour, show each message once, mark notices that cannot be tied to one request as unattributed, and test that two interleaved conversations list as two sessions.
- **C2 (1.5, 2.1, 2.7).** `apsis serve` always scopes to one project: an empty `--project` resolves to the current directory (`cmd/apsis/main.go` `resolveProjectPath`). LM Studio sessions and exec streams outside it never appear in serve. Fix: add an all-projects mode for serve, or state these sessions are CLI-only.
- **C3 (sections 1, 2, 5).** No requirement that listed sessions can be opened. Exec-stream file names collide across directories, and Claude lookup by ID searches only the current project when a project path is set (`internal/sessions/resolver.go` `findClaudeSessionPath`). Fix: every listed session has a unique, stable identifier that opens through `apsis <id>` and `/sessions/{source}/{id}`.

## Major

- **M1 (section 2).** LM Studio times are local without a zone; correlation by timestamp breaks if the zone is wrong. Fix: state the zone rule, emit an offset, define the repeated daylight-saving hour.
- **M2 (section 2).** Sessions spanning daily or `.N` files, and a body cut off at the end of a file, are undefined.
- **M3 (listing).** LM Studio listing parses about 130 MB a month; the serve list refreshes every 15 seconds (`internal/apsisweb/templates/sessions.html`); 1.5 searches directories at any depth. Fix: opt-in, or a measurable target.
- **M4 (1.5).** Path containment differs from Codex rollouts' exact cwd match; worktrees under `.worktrees/` would be credited to the parent; repository fixtures starting with `thread.started` would list. Decision 3 rejects scanning project trees, then suggests pointing at a parent of all projects. Fix: pick a rule, define candidate files, restate the consequence honestly.
- **M5 (1.3).** 1.2 omits the lifecycle events `thread.started`, `turn.started`, `item.started`, `item.updated`, so 1.3 would render them generically and duplicate every item. Fix: consume them silently; 1.3 covers unknown types only.
- **M6 (introduction, 1.2, 4.1, 6.1).** New fields are under-specified: token usage, plan entries, generic events, endpoint. The shared representation has no usage field and the Codex rollout parser skips `token_count`. Fix: list every field, say how usage renders, decide whether rollouts get usage.
- **M7 (3.2).** Locality is inferred from a provider ID, which Q3 and the non-goals rule out; `lmstudio` can be a remote host. Fix: drop locality and its setting.
- **M8 (raw output).** The raw `-f json` path reads JSONL, so a `.log` input warns on nearly every line. Fix: define `-f json` and `-F` for LM Studio input.
- **M9 (PR slices).** 4.1 lists fields only later slices deliver; the handoff's slice (c) still names orbit's local-run streams; its counts are stale. Fix: 4.1 becomes "every field the shared representation carries", extended additively per slice.
- **M10 (section 2).** The LM Studio log is undocumented free text and grouping is heuristic; "line" is undefined for multi-line bodies. Fix: tie to the inspected version, list lines ignored silently, consider a separate spec.

## Minor

- 5.1, 5.2: model and provider in listings force content reads; Claude project folder names cannot be decoded back to paths.
- 4.3 prescribes a validator mechanism; go.mod has no JSON Schema validator. State the outcome.
- 4.2 is policy; the testable part is that the output version equals the schema file's version.
- 2.4: the truncation mark must differ from the renderer's own `... (truncated)`.
- 1.4: `.stdout.jsonl` and `.prompt.txt` come from unmerged work; justify generically or move. Define stdin.
- 2.2: `/v1/messages` and `/v1/completions` are absent from captures.
- 5.3 duplicates 6.1.
- Decisions 2 and 3 cite unmerged orbit names.
- Exec events carry no timestamps; say where creation time comes from.

## Questions raised

LM Studio session unit; time-zone rule; LM Studio listing default; serve all-projects mode; keep locality; keep 1.4; separate LM Studio spec; names for the new `-f` value and the all-projects option; exec-stream project rule.
