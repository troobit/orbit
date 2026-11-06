<!-- Revision 1 of the requirements, before LM Studio was split out. The r1 reviews refer to these AC numbers. -->

# Requirements: Apsis — Codex Exec Streams and LM Studio Server Logs

## Introduction

Apsis translates each harness's transcript format into one shared representation and renders it as Markdown or HTML. This feature adds two input formats — the Codex CLI `codex exec --json` event stream and LM Studio server logs — and the fields they need in the shared representation: model provider, locality, truncated content and server notices. It also emits the shared representation as versioned JSON and lists sessions across all projects, so other tools can read every supported harness through apsis instead of re-parsing each format.

Formats were inspected against Codex CLI 0.166.0 output and its [non-interactive mode documentation](https://learn.chatgpt.com/docs/non-interactive-mode), and against LM Studio 0.4.26 server logs.

---

## Non-Goals

- Changing `-f json`, which continues to emit raw session data.
- Reading LM Studio chat conversations (`~/.lmstudio/conversations/`).
- Recovering message content that LM Studio truncated in its logs, or changing LM Studio's logging settings.
- Inferring a provider or locality that the source does not record (for example, Claude Code pointed at LM Studio through `ANTHROPIC_BASE_URL`).
- Linking an LM Studio log entry to the Codex or Claude Code session it served; consumers can correlate using the timestamps and models apsis emits.
- OpenCode transcript support.
- Changing how orbit invokes Codex or what it writes to `.orbit/`; tools that save exec streams integrate through the directory and prompt-file conventions in [1.4](#1.4) and [1.5](#1.5).
- Finding exec streams outside the configured directories, for example by scanning every project tree.

---

### 1. Codex Exec Event Streams

**User Story:** As a user of `codex exec --json`, directly or through a tool that saves its output, I want apsis to list and render those event streams, so that non-interactive Codex runs are as reviewable as interactive sessions.

**Acceptance Criteria:**

1. <a name="1.1"></a>Apsis SHALL detect a Codex CLI exec event stream (JSONL whose first event is `thread.started`) without a forced format, and SHALL accept `-a codex-exec` to force it.  
2. <a name="1.2"></a>Apsis SHALL translate exec events into the shared representation: `agent_message` items as assistant messages, `reasoning` items as thinking, `command_execution` items as tool use with command, exit code and output, `file_change`, `mcp_tool_call` and `web_search` items as tool use, plan-update items as plan entries, `error` items and `turn.failed` or `error` events as errors, and `turn.completed` usage as token usage, keeping every usage field the stream provides.  
3. <a name="1.3"></a>WHEN an item or event type is not listed in [1.2](#1.2), apsis SHALL render it as a generic event showing its type, and SHALL NOT drop it.  
4. <a name="1.4"></a>WHERE a file named `<base>.prompt.txt` sits beside a stream named `<base>.jsonl` or `<base>.stdout.jsonl`, apsis SHALL render its content as the first user message; WHERE no such file exists, the session SHALL render without a user message.  
5. <a name="1.5"></a>WHERE exec-stream directories are configured, apsis SHALL list the streams found in them, at any depth, under the source label `codex-exec` in `apsis --list` and `apsis serve`, and SHALL include a stream in a single-project listing only when its path lies inside that project's directory; WHERE none are configured, apsis SHALL list no exec streams.  

---

### 2. LM Studio Server Logs

**User Story:** As a user running models in LM Studio, I want apsis to list and render the API interactions LM Studio logged, so that local-model work served to any client is reviewable.

**Acceptance Criteria:**

1. <a name="2.1"></a>Apsis SHALL read LM Studio server logs from `~/.lmstudio/server-logs/YYYY-MM/YYYY-MM-DD.N.log`, with the directory overridable, and SHALL list them under the source label `lmstudio`, with `-a lmstudio` to force the format.  
2. <a name="2.2"></a>Apsis SHALL translate each logged inference request — `POST` to `/v1/chat/completions`, `/v1/completions`, `/v1/responses`, `/v1/messages` or `/api/v1/chat` — into the shared representation with its timestamp, endpoint, model identifier, request messages and, where logged, the response; model listing, model management, embeddings and other requests SHALL be excluded.  
3. <a name="2.3"></a>Apsis SHALL group requests into sessions by a documented, deterministic rule; IF a request cannot be assigned to a session unambiguously, THEN it SHALL form a session of its own.  
4. <a name="2.4"></a>WHERE LM Studio truncated content (`<Truncated in logs>`), apsis SHALL mark that content as truncated in every output and SHALL NOT present it as complete.  
5. <a name="2.5"></a>Apsis SHALL attach the warnings and errors LM Studio logged for a request (for example unsupported fields, developer-role conversion and server errors) to that request in every output.  
6. <a name="2.6"></a>IF a log contains requests without bodies, THEN apsis SHALL list those requests with the metadata present and SHALL state that the body was not logged.  
7. <a name="2.7"></a>Because the logs record no project, LM Studio sessions SHALL appear when listing across all projects and SHALL NOT appear in a single-project listing.  

---

### 3. Provider and Locality

**User Story:** As a reviewer, I want each session to show which provider served the model, so that I can tell local-model work from cloud work.

**Acceptance Criteria:**

1. <a name="3.1"></a>Apsis SHALL record a session's model provider where the source records it — Codex `session_meta.payload.model_provider`, and `lmstudio` for LM Studio sessions — and SHALL display it beside the model in Markdown and HTML, following the existing message-metadata display.  
2. <a name="3.2"></a>Apsis SHALL mark a session as local when its provider is in a configurable list of local providers, defaulting to `lmstudio` and `ollama`.  
3. <a name="3.3"></a>WHERE the source records no provider, apsis SHALL omit provider and locality silently, as it does for other unavailable metadata.  

---

### 4. Shared JSON Output

**User Story:** As a tool author, I want apsis to emit its shared representation as JSON, so that my tool reads every supported harness through one schema.

**Acceptance Criteria:**

1. <a name="4.1"></a>Apsis SHALL provide a new `-f` value that emits a session's shared representation as JSON: session metadata, then ordered entries with role, content, thinking, tool use and results, errors, timestamps, model, provider, locality, token usage, truncation marks and server notices.  
2. <a name="4.2"></a>The JSON output SHALL carry a schema version; adding fields SHALL keep the major version, and removing a field or changing its meaning SHALL increment it.  
3. <a name="4.3"></a>The schema SHALL be published in the repository as a JSON Schema file, and tests SHALL validate the JSON output for fixtures of every supported format against it.  
4. <a name="4.4"></a>The JSON output SHALL contain content as parsed, without the length limits the Markdown and HTML renderers apply to tool inputs and results.  

---

### 5. Listing Across Projects

**User Story:** As a user or tool author, I want to list sessions from every project at once, so that I can review or process recent work without visiting each project.

**Acceptance Criteria:**

1. <a name="5.1"></a>`apsis --list` SHALL accept an option that lists sessions from every project and source, showing each session's project path where the source records one.  
2. <a name="5.2"></a>Session listings SHALL be available as JSON with source, identifier, creation time, size, project path, model and provider where known, carrying the schema version from [4.2](#4.2).  
3. <a name="5.3"></a>`apsis --list` without the new option SHALL keep its current output.  

---

### 6. Compatibility and Verification

**User Story:** As a maintainer, I want the additions to leave existing behaviour intact and to be proven against real formats, so that they are safe to merge and easy to extend.

**Acceptance Criteria:**

1. <a name="6.1"></a>Markdown and HTML output for existing formats SHALL be unchanged, except for provider metadata where the source records it.  
2. <a name="6.2"></a>IF a line in a new format cannot be parsed, THEN apsis SHALL report it on stderr with its line number and continue, as it does for existing formats.  
3. <a name="6.3"></a>Each setting this feature adds — exec-stream directories, the LM Studio log directory and the local-provider list — SHALL be settable by an environment variable and by a command-line option for both `apsis` and `apsis serve`, with the option taking precedence.  
4. <a name="6.4"></a>Each new parser SHALL be tested against anonymised fixtures captured from the inspected versions, covering truncated bodies, requests without bodies, unknown item types, malformed lines and multi-day LM Studio logs; documented types absent from captures (such as `reasoning`, `mcp_tool_call`, `web_search` and `turn.failed`) SHALL use fixtures built from the documented shape and labelled as such.  
