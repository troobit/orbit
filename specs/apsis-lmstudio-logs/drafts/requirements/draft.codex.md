# Requirements Draft: Apsis — LM Studio Server Logs

## Introduction

LM Studio server logs can be the only retained record of inference requests from clients that keep no transcript. This feature translates those logs into apsis's shared representation for Markdown, HTML and JSON, while making missing content and uncertain attribution visible. It builds on `apsis-shared-json` and targets the server-log format inspected for LM Studio 0.4.26.

## Dependencies and Portability

- The shared JSON format, schema version, provider metadata and all-projects option come from [apsis-shared-json](../../../apsis-shared-json/requirements.md).
- LM Studio fields extend that schema additively; concrete field and option names belong to design.
- Public API documentation establishes endpoint payloads, not the on-disk log grammar; the supplied handoff establishes the inspected log grammar. Support for later log formats is not assumed.
- Every policy below is proposed for review. Questions at the end identify choices requiring human approval.

## Non-Goals

- Reading LM Studio's saved chat conversations or `lms log stream` output.
- Calling a model server, changing its logging settings or recovering content it did not log.
- Following LM Studio logs live with `-F`.
- Merging LM Studio records with a Codex, Claude Code or other harness transcript.
- Inferring project paths from prompt text, or locality from model or provider names.
- Depending on unmerged `orbit local` work or an orbit-specific file layout.
- Supporting inference endpoints outside the explicit allowlist below.

### 1. Server-Log Input and Discovery

**User Story:** As a reviewer, I want to open and discover LM Studio logs, so that I can inspect recorded inference traffic through apsis.

**Acceptance Criteria:**

1. <a name="1.1"></a>Apsis SHALL accept `-a lmstudio` for files and stdin, and SHALL automatically recognise input containing the inspected timestamp-and-level header followed by a `Received request: POST to` record for an endpoint in [2.1](#2.1); a `.log` extension alone SHALL NOT identify the format.  
2. <a name="1.2"></a>Apsis SHALL discover `YYYY-MM/YYYY-MM-DD.N.log` files in LM Studio's standard server-log directory and list logical sessions under source `lmstudio`; the root directory SHALL be overridable for both the CLI and `apsis serve` through a command-line option and an environment variable, with the option taking precedence.  
3. <a name="1.3"></a>WHEN reading a direct file or stdin, apsis SHALL render only the supplied records as one log transcript in source order, with each request's logical-session identifier shown; opening a listed logical session by identifier SHALL instead select that session's requests across the configured tree.  
4. <a name="1.4"></a>WHEN multiple requests share a file, opening one listed session SHALL exclude other sessions' request bodies and attributed responses, while retaining the explicitly unattributed records required by [4.2](#4.2).  
5. <a name="1.5"></a>WHEN resolving a listed identifier, apsis SHALL read only files inside the configured root after resolving symbolic links; a direct file argument SHALL remain an explicit choice to read that file.  

### 2. Request Translation and Missing Content

**User Story:** As a tool author, I want recorded requests translated consistently, so that every output describes the same evidence.

**Acceptance Criteria:**

1. <a name="2.1"></a>Apsis SHALL translate only `POST` requests whose path exactly matches `/v1/responses`, `/v1/chat/completions`, `/v1/completions`, `/v1/messages` or `/api/v1/chat`; it SHALL exclude model listing, model management, embeddings and `/v1/messages/count_tokens`.  
2. <a name="2.2"></a>For each included request, apsis SHALL preserve its endpoint, recorded model, source location and request boundary, set provider `lmstudio`, and translate the recorded `input`, `messages` or `prompt`, including separate system instructions, into ordered shared entries with their supplied roles.  
3. <a name="2.3"></a>WHERE a request or attributable response contains thinking, tool calls or tool results, apsis SHALL preserve those as the equivalent shared entries; unrecognised content blocks SHALL appear as labelled generic events rather than being silently dropped.  
4. <a name="2.4"></a>WHERE a response body is attributable under [4.1](#4.1), apsis SHALL translate its recorded output; WHERE it is absent or unattributable, apsis SHALL label the request's response as not recorded or attribution unknown, respectively, and SHALL NOT infer generated output from a later request's input history.  
5. <a name="2.5"></a>IF an included request has no logged body, THEN apsis SHALL retain the available request metadata and show `body not logged`; IF its body is invalid or unfinished, THEN apsis SHALL retain the request with the corresponding state and the diagnostic required by [7.2](#7.2).  
6. <a name="2.6"></a>WHERE content includes LM Studio's `<Truncated in logs>` marker, apsis SHALL preserve the available content and mark it as truncated in Markdown, standalone HTML, web HTML and shared JSON; JSON SHALL add no renderer length limits to the parsed content.  

### 3. Session Identity and Repeated History

**User Story:** As a reviewer, I want stable sessions with explicit history boundaries, so that repeated context and compaction do not look like new conversations.

**Acceptance Criteria:**

1. <a name="3.1"></a>WHERE a parsed request contains a nonempty string `client_metadata.thread_id`, apsis SHALL group requests with the exact same value within the configured log root into one session, including across daily and numbered files; an endpoint change, model change or shorter input history SHALL NOT split that session.  
2. <a name="3.2"></a>WHERE that thread identifier is absent or invalid, apsis SHALL place each request in its own session; equality of model, prompt text, `prompt_cache_key` or an otherwise unverified identifier SHALL NOT group requests.  
3. <a name="3.3"></a>Every listed session and every request within it SHALL have a deterministic identifier; listed identifiers SHALL be unique within `lmstudio`, contain only ASCII letters, digits, `_` and `-`, be at most 256 characters, and remain unchanged when existing files are appended or later files are added, with the root and existing file names unchanged.  
4. <a name="3.4"></a>Apsis SHALL preserve the original client thread identifier as correlation metadata in every output where present, while keeping LM Studio and harness sessions separately listed even when those identifiers match.  
5. <a name="3.5"></a>WHEN consecutive requests in one session repeat an identical leading sequence of normalised input items, apsis SHALL show that prefix once and refer to it from subsequent request boundaries; JSON SHALL retain enough information to recover each request's complete input sequence, and repeated items outside that common prefix SHALL remain present.  
6. <a name="3.6"></a>WHEN a request's input is not an extension of the preceding request's input, apsis SHALL mark a context discontinuity, preserve the changed suffix, and retain the same thread session; it SHALL NOT treat assistant messages supplied as context as proof that their output was logged.  

For [3.5](#3.5), equality compares role, content and recorded item identifiers after translation; JSON object key order is irrelevant, but array order and string contents are significant. A shorter exact prefix adds no new input items but still produces its request boundary and the discontinuity marker. Request histories never merge across singleton sessions.

### 4. Concurrent Requests and Attribution

**User Story:** As a reviewer, I want uncertainty kept visible, so that parallel requests do not acquire each other's responses or warnings.

**Acceptance Criteria:**

1. <a name="4.1"></a>Apsis SHALL attach a response, warning or error to an individual request only when the record supplies an explicit identifier that uniquely matches that request; proximity, matching model tags and timestamp equality alone SHALL NOT establish attribution.  
2. <a name="4.2"></a>WHEN a logged response, warning or error cannot be uniquely attributed, apsis SHALL retain it as a log-level event labelled `request attribution unknown`, with its source location and timestamp, in a direct transcript and in each logical session containing a request in the same source file; it SHALL NOT label that event as any request's response or failure.  
3. <a name="4.3"></a>Markdown, standalone HTML, web HTML and shared JSON SHALL expose the same attribution state and event identity, so that consumers can recognise a log-level event repeated across session views.  
4. <a name="4.4"></a>Apsis SHALL ignore routine progress, startup and model-management records that carry no included request, response, warning or error; an unrecognised record claiming to be an included request or response SHALL produce a warning rather than silently appearing as successfully translated content.  

### 5. Timestamps, File Order and Partial Records

**User Story:** As a reviewer, I want reproducible ordering and declared time assumptions, so that daylight-saving transitions and rotation do not reorder the evidence.

**Acceptance Criteria:**

1. <a name="5.1"></a>Apsis SHALL preserve each record's logged wall time and interpret it in a selectable IANA time zone for both the CLI and `apsis serve`, with command-line selection taking precedence over the environment and the operating system's local zone as the default; every output SHALL identify the selected zone as an assumption because the log records none.  
2. <a name="5.2"></a>WHERE the wall time has one interpretation in the selected zone, apsis SHALL emit a timestamp with an explicit UTC offset; WHERE it is invalid, nonexistent or ambiguous during a daylight-saving transition, apsis SHALL retain the wall time, omit a claimed absolute timestamp and report the uncertainty.  
3. <a name="5.3"></a>Apsis SHALL preserve physical record order within a file and, for discovery, order files by their filename date and numeric `N` suffix; wall times SHALL NOT reorder records, including through a repeated daylight-saving hour.  
4. <a name="5.4"></a>WHEN a final request body is incomplete at end of file, apsis SHALL mark it incomplete and continue with subsequent files without concatenating their bytes into that body; a later read after the original file grows SHALL replace the incomplete state if that body becomes complete.  

A record starts at a physical line matching `[YYYY-MM-DD HH:MM:SS][LEVEL]` with optional following tags, and includes continuation lines up to the next such header or EOF. Line numbers are physical, one-based and local to each file; warning/error attribution does not follow merely from membership in a multiline log record.

### 6. Listing, Refresh and Exposure

**User Story:** As a user of `apsis serve`, I want predictable discovery without prompt previews, so that large logs can be browsed alongside other sources.

**Acceptance Criteria:**

1. <a name="6.1"></a>LM Studio sessions SHALL appear only with the all-projects listing option supplied by `apsis-shared-json`, in both CLI and web listings, and SHALL open by source and identifier from any working directory; their project path SHALL remain unknown.  
2. <a name="6.2"></a>Listings SHALL show session metadata and source warnings through the existing human-readable and shared JSON listing contracts, and SHALL NOT include request bodies, response bodies or prompt snippets.  
3. <a name="6.3"></a>Within one `apsis serve` process, refreshing an unchanged log tree SHALL read zero log-body bytes after the initial listing; appended or replaced files SHALL be reflected on the next successful listing refresh, without changing unaffected session identifiers.  
4. <a name="6.4"></a>IF the default log directory is absent, THEN apsis SHALL list no LM Studio sessions without a warning; IF an explicitly selected directory is absent or a candidate file cannot be read, THEN apsis SHALL surface a source warning and retain results from other readable files and sources.  
5. <a name="6.5"></a>Help for LM Studio discovery SHALL state that recorded bodies can contain full prompts and sensitive data; `apsis serve` SHALL retain its localhost binding default and its existing network-exposure warning when bound to `0.0.0.0`, with no prompt content in that warning.  

### 7. Compatibility and Verification

**User Story:** As a maintainer, I want explicit unsupported outcomes and meaningful fixtures, so that this parser can ship independently upstream.

**Acceptance Criteria:**

1. <a name="7.1"></a>IF LM Studio input is combined with raw `-f json` or follow mode `-F`, THEN apsis SHALL exit unsuccessfully with an error naming the unsupported combination; the raw JSON error SHALL direct users to the shared JSON format, and existing sources' behaviour SHALL remain unchanged.  
2. <a name="7.2"></a>IF an included record cannot be parsed, THEN apsis SHALL report its file or stdin label and physical start/end line numbers, continue at the next recognised header, and expose that warning on stderr for CLI conversion and in shared JSON or web warnings without copying body content into the diagnostic.  
3. <a name="7.3"></a>LM Studio fixtures SHALL validate the additive shared JSON schema and behaviour in Markdown, standalone HTML and web HTML, including identity across appends, concurrent attribution, repeated history, compaction, missing and truncated bodies, malformed multiline records, rotation, time-zone uncertainty and source warnings.  
4. <a name="7.4"></a>Fixtures SHALL contain synthetic content, identifiers, paths and model names only; fixtures representing the inspected 0.4.26 log grammar SHALL be labelled separately from endpoint payloads constructed from public documentation, especially `/v1/messages` and `/v1/completions`, whose log forms were not established by the inspected samples.  

## Public References

- [OpenAI-compatible endpoints](https://lmstudio.ai/docs/developer/openai-compat) and [Responses](https://lmstudio.ai/docs/developer/openai-compat/responses) document the compatible inference APIs.
- [Anthropic-compatible endpoints](https://lmstudio.ai/docs/developer/anthropic-compat) documents `POST /v1/messages`; [legacy completions](https://lmstudio.ai/docs/developer/openai-compat/completions) documents LM Studio's continuing completions support.
- [Native chat](https://lmstudio.ai/docs/developer/rest/chat) documents `/api/v1/chat`, including stateful requests. Retaining a `previous_response_id` does not itself approve grouping by response chains in this draft.
- [`lms log stream`](https://lmstudio.ai/docs/cli/serve/log-stream) describes a separate live logging interface, not a contract for these disk files.

## Open Questions for the User

1. **Requests without thread identifiers:** keep one request per session (recommended), or add grouping through uniquely logged `previous_response_id` chains? Prefix/time/model heuristics are not recommended because they can merge unrelated clients; documented response chaining is plausible but was not established in the inspected logs.
2. **Repeated history:** approve exact leading-prefix references with discontinuities marked (recommended), or keep full request snapshots visible every time? The recommendation removes routine repetition without deduplicating equal text at unrelated positions.
3. **Unattributed records:** include clearly labelled file-level events in every session touching that file (recommended), or expose them only in direct file views? The first can show unrelated sensitive content in a session view; the second makes important errors invisible when opening a session by ID.
4. **Project assignment and exposure:** keep LM Studio discovery limited to explicit all-projects listings with unknown projects (recommended), or additionally require an LM Studio opt-in? Extracting a working directory from client prompt text is not recommended because the server log has no authoritative project field.
5. **Time-zone uncertainty:** default to the host zone with a declared assumption and omit absolute times for ambiguous/nonexistent wall times (recommended), or require a zone before conversion and select a documented offset for ambiguous times? Neither option can recover the original offset from the log alone.
