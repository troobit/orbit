# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Groundwork for LM Studio server logs in Apsis: the shared transcript representation gains per-request boundary metadata (`LMStudioRequest`) and request/attribution fields on entries, and a `TZ` resolver interprets logged wall times. It distinguishes unset `TZ` (host zone) from empty `TZ` (UTC), accepts IANA names and absolute TZif file paths with an optional leading `:`, warns instead of substituting a zone when an override fails to load, and gives no timestamp for nonexistent or ambiguous wall times. Binaries that link the transcript package embed the IANA zone database (`time/tzdata`).
- An LM Studio server-log parser in the transcript package. Input is auto-detected only when the first 64 KiB hold a request to one of the seven inference endpoints. A log with headers but no such request asks for `-a lmstudio` instead of falling back to the Claude parser. Requests to the chat completions, completions, Messages, Responses and native chat endpoints are translated with their roles. Their remaining body fields are kept as parameters in source order, and `client_metadata` supplies the thread, parent thread and request kind. Logged responses, WARN and ERROR records stay unattributed. Unsupported POST paths produce one warning per path with a count. Missing, malformed, unfinished and oversized (over 10 MiB) bodies keep the request's metadata with a state. Content cut by `<Truncated in logs>` is marked, and each threaded request is marked as extending its predecessor's history or as a context discontinuity. The `-a lmstudio` flag itself and discovery come in later phases.
- Apsis Markdown and HTML render LM Studio server logs. Each request shows its endpoint, source lines, assumed time and zone, model, thread, parent thread, request kind and body state, and whether it extends the thread's previous request or breaks its context. History repeated from the previous request is referenced rather than shown again, and request parameters sit in a collapsed block that summarises the stream setting and offered tool names. Logged responses, warnings and errors appear as collapsed unattributed blocks, and one notice per transcript states the assumed time zone and that request attribution is unknown. Tool calls in request history show with their results under the supplied-context assistant message, and `<Truncated in logs>` stays visible in rendered Markdown.
- Apsis shared JSON carries LM Studio server logs. Session documents gain a `requests` array. Each request lists its ID, logical session, source lines, endpoint, thread, parent thread, request kind, model, logged wall time, assumed timestamp with offset, body state, parameters as logged, history-prefix result, and `response_attribution: "unknown"`. Entries gain `request_id`, `supplied_context`, `unattributed`, `source_line` and `wall_time`. Blocks gain `truncated`, and session metadata gains `time_zone`. Logical-session views can carry an `unattributed_summary` of body-free per-file event counts and lines. LM Studio listing rows report `request_count` instead of `size_bytes`. The published schema describes these fields and four LM Studio warning codes. A logged response's `usage` block records basis `response`. A session's `created_at` comes only from its first request's own time.
- The Apsis session lister can discover LM Studio server logs (`YYYY-MM/YYYY-MM-DD.N.log` under one root) as source `lmstudio`, in all-projects scope only. Project-scope listings read nothing from the root. Requests that share a client thread ID form one logical session across files; every other request is its own session. Identifiers are derived from the root-relative file path and start line, or from the thread ID, so they stay the same when files are appended, added or removed. Listing rows give the request count, the first request's assumed time, the first recorded model and the thread ID. File symlinks are followed only within the root, and symlinked month directories are not descended. On Unix, a file whose size, times, inode and device are unchanged is not read again within one process. A missing root is reported only when it was chosen explicitly. The `--lmstudio-log-dir` option and opening these sessions come in later phases.
- `orbit local` accepts `--context-window`, `--compact-at`, and opt-in `--network` controls for local model budgets and sandboxed dependency downloads.
- `orbit local <tasks.md>` implements an explicit rune task file through Codex and LM Studio, using the `8-bit` model by default with fresh sessions, task or phase selection, and configurable timeout and iteration limits.
- Local runs retain prompts and live stdout/stderr, report elapsed time, and stop on halt signals, changed task graphs, worker failures, or lack of completed work. Unix cancellation terminates worker child processes.
- `make install-local-next-task` installs Orbit and a compatible `local-next-task` command, replacing the known agentic-coding prototype link while preserving unrelated files.
- An `8-bit` implementation quick guide, local-runner runbook, and recorded live-model verification document setup, execution, diagnostics, and recovery.
- `orbit finalize` now displays the variant's agent (alias, type, model) in its preamble, falling back to "Agent: unknown" when none of the fields are populated.
- `orbit finalize` now reads the spec's `consolidation-log.json` and prints a warning before the confirmation prompt when the requested `--variant` differs from the most recent consolidation entry. The warning fires under `--force` as well so it remains visible in CI output.
- `apsis -f shared-json` outputs a session as versioned JSON with the same layout for every agent: source, identifier, recorded creation time, project, model, provider, origin and cost, followed by entries with their recorded identifiers and structured warnings. The layout is published as a JSON Schema in `docs/schemas/apsis-shared-json.v1.schema.json`.
- Apsis Markdown and HTML assistant headers show the model provider recorded by Codex rollouts and Kiro IDE chats beside the model.
- `apsis --list -f shared-json` outputs a versioned listing document: each session's source, identifier, size, recorded creation time or file modification time, project, model, provider and origin where recorded, plus listing warnings with a code.
- `apsis --all-projects` lists, opens and picks the latest session from every project; text listings gain a project column. `apsis serve --all-projects` shows sessions from every project with their project.
- `apsis -s`/`--source <source>` opens the argument as a session ID of that source, so identical IDs in different agents can be told apart.
- Apsis reads event streams saved from Codex CLI `codex exec --json`: a stream file or stdin is detected from its `thread.started` event, and `-a codex-exec` forces parsing of a stream that failed before its thread started. Markdown, HTML and shared JSON show each turn's messages, reasoning, commands with exit codes, file changes, MCP and web-search calls, plans, errors, unrecognised events and token usage (reported thread totals at turn completion), and mark items that never completed.
- `apsis --codex-exec-dir <dir>` (repeatable) and `APSIS_CODEX_EXEC_DIRS` (separated like `PATH`) list exec streams under the given directories as source `codex-exec`, for `apsis --list`, `apsis latest` and `apsis serve`. Each stream belongs to its containing git repository or worktree, is ordered by its file modification time (marked `file modified`), and is hidden when a Codex rollout records the same thread. Nothing is searched by default.

### Changed

- The `apsis serve` listing warning banner now refreshes with the session list.
- The LM Studio log support spec clarifies truncated thread identifiers, discovery identifier scope, and unsupported POST path handling, and consolidates its decision log.

### Fixed

- The apsisweb end-to-end test no longer times out on shutdown when the HTTP client holds an unused keep-alive connection.
- Local Codex runs exclude connected ChatGPT app namespaces that appear under normal authentication despite ignored user configuration.
- Restarting a run in an existing log directory clears the previous error and completion timestamp while preserving session history.
- Local Codex workers receive explicit context and compaction budgets and bounded tool-output history to reduce prompt truncation, with guidance for focused reads and dependency blockers.
- Failed worker attempts retain diagnostics without increasing the summary's completed-phase count.
- Local Codex runs disable unsupported web search and delegation tools and unused reasoning summaries; setup guidance explains the remaining LM Studio protocol warnings.
- Local completion checks include in-progress tasks so interrupted work is dispatched again when resuming.
- Codex JSONL execution detects failed turns even when the CLI exits successfully and retains the stream in Orbit's standard session artifacts.
- Claude Code session-limit messages using the new `"You've hit your session limit"` wording now wait until the reported reset time while retaining support for the older message.
- Codex usage-limit messages (`"You've hit your usage limit. ... try again at 4:18 PM"`, including the dated `"try again at Sep 5th, 2026 1:39 AM"` form) are now classified as rate-limit waits that sleep until the reported retry time, instead of failing the phase after generic retries.

## [0.9.0] - 2026-04-11

Initial public release of Orbit and Apsis.

### Orbit - AI Coding Agent Orchestrator

#### Multi-Agent Support

- Support for 5 AI coding agents: Claude Code, OpenAI Codex, AWS Kiro, GitHub Copilot, and OpenCode
- Agent abstraction layer with pluggable agent implementations via registry pattern
- Per-agent error classification with appropriate retry strategies (rate limits, connection errors, session invalid, fatal)
- Shared retry executor with exponential backoff and context-aware interruptible sleep
- Claude Code usage limit detection: automatically waits until reset time and resumes execution
- Agent selection via `--agent` flag, `ORBIT_AGENT` environment variable, or `.orbit.yaml` config

#### Phase-Based Orchestration

- Sequential phase execution driven by rune task files with automatic task detection from git branch
- Session continuation across phases and crash recovery via session ID tracking
- Pre-prompt and post-prompt AI interactions for setup/review around the phase loop
- Agent-level pre-command and post-command shell hooks with configurable timeout
- Execution order: pre-command -> pre-prompt -> phases -> post-prompt -> post-command
- Dry-run mode for previewing execution without running agents

#### Multi-Variant Comparison

- `orbit run --variants N` creates git worktrees and runs parallel or sequential implementations
- `--variant-agents` assigns different agents to variants with cycling support
- `--guidance-file` provides per-variant YAML instructions
- Per-variant model selection via agent aliases in `.orbit.yaml`
- AI-powered comparison with structured JSON output, recommendation, and confidence level
- HTML and Markdown comparison reports with per-file analysis and learnings
- Cross-variant improvement identification for consolidation
- Comparison failure recovery: falls back to written JSON file if agent session fails

#### Consolidation and Finalization

- `orbit consolidate` merges improvements from non-chosen variants into the chosen one
- Auto-consolidation after comparison with `--auto-consolidate` flag
- Rollback support for failed consolidations
- `orbit finalize` adopts a variant via fast-forward merge and cleans up worktrees
- `orbit cleanup` removes all variant worktrees without adopting any
- Variant session recovery when re-running: continue, new run (preserving completed), or cancel

#### Status and Monitoring

- `orbit status` shows variant progress with recent commits, git state, last action, and task progress
- Text and JSON output formats
- `orbit serve` web interface with dashboard, run details, transcript viewer, and HTMX live updates
- Run registry at `~/.orbit/runs/` for tracking runs across repositories
- Centralized structured logging to `~/.orbit/logs/` in JSON Lines format

#### Configuration

- Layered configuration: CLI flags > environment variables > project `.orbit.yaml` > home `~/.orbit.yaml` > defaults
- `orbit init` generates default configuration file
- Per-agent settings: cli-path, auto-approve, timeout, extra-args, model, pre-command, post-command
- Agent aliases for per-variant model selection (e.g., `claude-sonnet`, `claude-opus`)

### Apsis - Session Transcript Viewer

#### Transcript Conversion

- Convert session transcripts from all 5 agent formats to Markdown or HTML
- Auto-detection of transcript format (Claude JSONL, Codex JSONL, Copilot JSONL, Kiro CLI SQLite/JSON, Kiro IDE JSON)
- `apsis latest` to view the most recent session
- Follow mode (`-F`) for live session monitoring (like `tail -f`)
- JSON output (`-f json`) for raw session data inspection
- Force agent format with `-a` flag when auto-detection is insufficient

#### Session Discovery

- Unified session listing across all agent types via `apsis -l`
- Session resolution by ID or file path
- Kiro IDE `.chat` file support with credit cost extraction
- Kiro CLI SQLite session discovery with path normalization and symlink support

#### Web Interface

- `apsis serve` starts a local web server for browsing sessions
- Session filtering by agent type and search by ID
- Dark mode, mobile-responsive design
- HTMX-powered auto-refresh (15s polling)
- Security headers and input validation middleware

### Build and CI

- GitHub Actions CI workflow for testing and linting
- Multi-platform release workflow (linux/windows/darwin x amd64/arm64)
- Build metadata injection: version, build time, and git commit via ldflags
