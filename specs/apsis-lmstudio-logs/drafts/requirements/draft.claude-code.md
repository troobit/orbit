# Requirements: Apsis — LM Studio Server Logs

## Introduction

LM Studio writes a server log of the requests its local HTTP server receives, including the request bodies that clients such as Codex send to a local model. This feature adds those logs as an apsis source: apsis groups the logged inference requests into sessions, translates them into the shared representation from [apsis-shared-json](../apsis-shared-json/requirements.md), and renders them as Markdown, HTML and shared JSON. For clients that keep no transcript the log is the only record of local-model work, and for Codex it shows what the server received. The log format is undocumented; these requirements describe LM Studio 0.4.26 logs written with `fileLoggingMode: succinct`.

---

## Terms

- **Log file**: a file `YYYY-MM/YYYY-MM-DD.N.log` in the log directory. LM Studio starts a new file at local midnight and when the current file passes about 10 MiB, raising `N`.
- **Header line**: a line starting `[YYYY-MM-DD HH:MM:SS][LEVEL]`, optionally followed by `[tag]`.
- **Entry**: a header line and the lines after it up to the next header line.
- **Inference request**: an entry `[DEBUG] Received request: POST to <path>` where `<path>` is exactly `/v1/chat/completions`, `/v1/completions`, `/v1/responses`, `/v1/messages`, `/api/v0/chat/completions`, `/api/v0/completions` or `/api/v1/chat`. Its body is the JSON after `with body` in the same entry.
- **Conversation**: the exchange in a request body: `messages` (with `system` for `/v1/messages`), `input` (with `instructions`) for `/v1/responses` and `/api/v1/chat`, or `prompt` for completions. A string counts as one item.
- **Thread ID**: the string `client_metadata.thread_id` that Codex sends in `/v1/responses` bodies.
- **Response body**: an entry `Generated prediction: {` or `Generated synchronous v1/responses response: {`.
- **Follow-up line**: a WARN entry, an ERROR entry or a response body.
- **End line**: a response body, an entry `Finished streaming response`, or an ERROR entry.

---

## Non-Goals

- LM Studio chat conversations in `~/.lmstudio/conversations/`.
- Recovering content LM Studio cut with `<Truncated in logs>`, or responses it streamed without logging them.
- Grouping requests by identifiers from clients other than Codex, or by `previous_response_id`.
- Merging an LM Studio session into the Codex session it served; both stay listed.
- Linking from Codex session pages back to LM Studio sessions.
- Showing LM Studio sessions in single-project listings.
- Embedding, model listing and model management requests.
- Rendering the tool definitions and sampling settings sent with each request.
- Progress, prompt-cache and inference-engine lines such as `Prompt processing progress`.
- Detecting the time zone a log was written in; apsis uses its own.
- Following logs live (`-F`) and raw `-f json` output for logs.
- Keeping a listing index on disk between apsis runs.
- Changing LM Studio's logging settings.
- Log formats of LM Studio versions other than 0.4.26, beyond the fallbacks in [1.4](#1.4) and [2.3](#2.3).

---

### 1. Log Discovery and Settings

**User Story:** As a user running models in LM Studio, I want apsis to find its server logs, so that local-model sessions are listed without extra steps.

**Acceptance Criteria:**

1. <a name="1.1"></a>WHERE no log directory is configured, apsis SHALL read LM Studio server logs from `~/.lmstudio/server-logs`, taking only files that match `YYYY-MM/YYYY-MM-DD.N.log` and ignoring all other files.  
2. <a name="1.2"></a>The log directory SHALL be settable by an environment variable and by a command-line option for both `apsis` and `apsis serve`, with the option taking precedence and an empty environment variable meaning that apsis reads no LM Studio logs.  
3. <a name="1.3"></a>WHEN opening an LM Studio session by identifier, apsis SHALL read only files that lie inside the log directory after resolving symbolic links.  
4. <a name="1.4"></a>IF a non-empty log file contains no header line, THEN apsis SHALL report a listing warning naming the file and SHALL continue with the other files.  

---

### 2. Request Translation

**User Story:** As a reviewer, I want each logged inference request translated into the shared representation, so that I can read what a client sent the model and what came back.

**Acceptance Criteria:**

1. <a name="2.1"></a>Apsis SHALL translate each inference request into the shared representation with its timestamp, endpoint, model, log file name and header line number, and with its conversation items: messages with the role as sent, tool calls with their name and arguments, tool outputs as tool results linked to their call where the body gives the call ID, and reasoning as thinking; Markdown and HTML SHALL show each request's timestamp, endpoint, model and log location following the existing message-metadata display.  
2. <a name="2.2"></a>WHEN a response body is attached to a request under [4.3](#4.3), apsis SHALL translate it as assistant output with its text, tool calls and token usage.  
3. <a name="2.3"></a>WHEN a conversation item or content part has a type not covered by [2.1](#2.1), apsis SHALL render it as a generic event showing its type, and SHALL NOT drop it.  
4. <a name="2.4"></a>WHERE content contains LM Studio's truncation marker `... <Truncated in logs> ...`, apsis SHALL mark that content as truncated in Markdown, HTML and JSON, and SHALL NOT present it as complete.  
5. <a name="2.5"></a>IF an inference request has no body, THEN apsis SHALL show the request with its timestamp and endpoint and state that the body was not logged.  
6. <a name="2.6"></a>IF a request body is not valid JSON, THEN apsis SHALL show the request with its timestamp and endpoint marked as unreadable, report the log file name and header line number on stderr, and continue.  
7. <a name="2.7"></a>Apsis SHALL ignore silently: entries for other methods and paths (including `/v1/responses/input_tokens`, `/v1/messages/count_tokens` and embeddings) with their bodies, and INFO and DEBUG entries other than response bodies and `Finished streaming response`.  

---

### 3. Session Grouping

**User Story:** As a reviewer, I want logged requests grouped into the sessions they belong to, so that one conversation reads as one transcript.

**Acceptance Criteria:**

1. <a name="3.1"></a>Apsis SHALL group inference requests that carry the same thread ID into one session, across all log files.  
2. <a name="3.2"></a>WHEN a request without a thread ID has a conversation that begins with the complete conversation of the latest request in exactly one earlier session without a thread ID, at the same endpoint and model, and is longer than it, apsis SHALL add the request to that session; otherwise the request SHALL start a new session.  
3. <a name="3.3"></a>Within a session, apsis SHALL show each conversation item once: for each request after the first, only the items after the longest prefix it shares with the previous request in the session, and no assistant item equal to a response body already shown.  
4. <a name="3.4"></a>WHEN a request leaves out or changes items of the previous request's conversation in its session, such as after context compaction, apsis SHALL mark that request as having replaced earlier context and state how many earlier items it no longer includes.  
5. <a name="3.5"></a>Each session SHALL have an identifier that is unique within the `lmstudio` source, differs from every identifier apsis gives Codex sessions, consists of at most 256 ASCII letters, digits, `-`, `_` and `.` without `..`, and stays the same as log files grow and new log files are added.  
6. <a name="3.6"></a>A session's creation time SHALL be the timestamp of its first request, and its size the total bytes of its requests' entries and attached follow-up lines.  

---

### 4. Server Warnings, Errors and Responses

**User Story:** As a reviewer, I want the warnings, errors and responses LM Studio logged shown with the request they belong to, so that I see how the server handled it without apsis guessing between parallel requests.

**Acceptance Criteria:**

1. <a name="4.1"></a>Apsis SHALL attach WARN and ERROR entries that directly follow an inference request, with no other entry between them, to that request.  
2. <a name="4.2"></a>A request SHALL count as in progress from its entry until the first of: an end line attached to it, the next request in its session, the end of its log file, or the point where the end lines for its model, attached or not, equal the requests for that model received since the model last had no request in progress.  
3. <a name="4.3"></a>WHEN a follow-up line not covered by [4.1](#4.1) has a tag naming a model and exactly one request for that model is in progress, or has no model tag and exactly one request of any model is in progress, apsis SHALL attach the line to that request.  
4. <a name="4.4"></a>IF a follow-up line could belong to more than one in-progress request and all of them are in one session, THEN apsis SHALL attach it to that session, marked as not tied to a single request.  
5. <a name="4.5"></a>IF a follow-up line could belong to in-progress requests in more than one session, THEN apsis SHALL attach it to none of them, and each of those sessions SHALL show in Markdown, HTML and JSON how many such lines may belong to it.  
6. <a name="4.6"></a>Apsis SHALL render attached WARN entries as notices and attached ERROR entries as errors, each with its timestamp and message, in Markdown, HTML and JSON.  

---

### 5. Timestamps and Order

**User Story:** As a reviewer, I want correct instants and order for log entries written in local time, so that sessions read and sort correctly across time-zone and daylight-saving changes.

**Acceptance Criteria:**

1. <a name="5.1"></a>Apsis SHALL read log timestamps as local time in the time zone of the apsis process (`TZ` when set, otherwise the system zone), and SHALL emit them in JSON output with an explicit UTC offset.  
2. <a name="5.2"></a>Apsis SHALL order requests and follow-up lines by log file date, then by the file number `N` compared as a number, then by line number, and SHALL NOT reorder them by timestamp.  
3. <a name="5.3"></a>WHEN a timestamp falls in a local hour that occurs twice, apsis SHALL choose the earlier instant unless it precedes the previous timestamp in the same file, in which case it SHALL choose the later one; WHEN a timestamp names a local time that does not exist, apsis SHALL apply the offset in effect before the change.  

---

### 6. Listing, Opening and Linking

**User Story:** As a user or tool author, I want LM Studio sessions listed and opened like other sessions and linked to the Codex session they served, so that I can move between the client's record and the server's.

**Acceptance Criteria:**

1. <a name="6.1"></a>WHEN listing across projects ([apsis-shared-json 4.1](../apsis-shared-json/requirements.md#4.1)), apsis SHALL list each LM Studio session under the source label `lmstudio` with its identifier, creation time, size, the model of its first request, provider `lmstudio` and number of requests, in `apsis --list`, `apsis serve` and JSON listings.  
2. <a name="6.2"></a>LM Studio sessions SHALL NOT appear in single-project listings, and SHALL open by identifier through `apsis` and the `apsis serve` session page only when the all-projects option is given; otherwise apsis SHALL report the session as not found.  
3. <a name="6.3"></a>WHERE a session has a thread ID, apsis SHALL show it in Markdown, HTML and JSON session output and include it in JSON listings.  
4. <a name="6.4"></a>WHERE apsis also lists a `codex` or `codex-exec` session with the same thread ID, the `apsis serve` session page SHALL link to that session, and listings across projects SHALL show its project path for the LM Studio session.  

---

### 7. Direct Input and Unsupported Modes

**User Story:** As a user, I want to convert a log file or excerpt directly and get clear errors for modes that do not apply, so that I can inspect logs outside the listing.

**Acceptance Criteria:**

1. <a name="7.1"></a>Apsis SHALL detect LM Studio log content from a file path or stdin when its first non-empty line is a header line, and SHALL accept `-a lmstudio` to force the format.  
2. <a name="7.2"></a>WHEN given a log file or stdin, apsis SHALL group only the requests in that input, by [3.1](#3.1) and [3.2](#3.2), and render every resulting session in order of its first request, each under its own heading in Markdown and HTML, and as a JSON array of session objects in the shared JSON output.  
3. <a name="7.3"></a>IF follow mode (`-F`) or raw `-f json` is used with LM Studio log input or an LM Studio session, THEN apsis SHALL exit with an error naming the unsupported combination.  

---

### 8. Performance, Exposure and Verification

**User Story:** As a maintainer, I want the source to stay fast on large logs, to warn about what it exposes and to be tested against the real format, so that it is safe to merge.

**Acceptance Criteria:**

1. <a name="8.1"></a>WHEN listing across projects or opening an LM Studio session with 150 MB of logs and nothing cached in the process, apsis SHALL finish within 3 seconds, as measured by a benchmark in the repository on generated logs.  
2. <a name="8.2"></a>WHILE `apsis serve` runs, each refresh of the session list SHALL re-read only log files whose size or modification time changed since the previous refresh.  
3. <a name="8.3"></a>WHEN `apsis serve` prints its network-access warning and LM Studio logs are read, the warning SHALL state that LM Studio server logs can hold requests from every client of the LM Studio server.  
4. <a name="8.4"></a>WHERE no log directory exists or reading is disabled, apsis output SHALL be unchanged, and single-project listings SHALL be unchanged whether or not LM Studio logs exist.  
5. <a name="8.5"></a>The log parser SHALL be tested against synthetic fixtures that reproduce the LM Studio 0.4.26 line format, multi-line bodies, truncation markers, size rotation, interleaved parallel requests, bodies that do not parse and a repeated daylight-saving hour, and that contain no content from real logs; endpoints absent from captures (`/v1/messages` and `/api/v0/completions`) SHALL use fixtures built from LM Studio's documented request shapes and labelled as such.  

---

## Open Questions for the User

1. Apsis reads `~/.lmstudio/server-logs` by default, as it does other standard locations. When LM Studio accepts network connections, its log also holds requests from other machines. Should reading be opt-in through a configured directory instead, as for exec streams?
2. LM Studio's `succinct` file logging keeps only the first and last 50 characters of every long string, so sessions show the shape of a conversation but little of its text. Is that worth building, and can you capture a log in another logging mode so fixtures can cover it?
3. Requests without a thread ID that continue no earlier conversation, such as single-shot batch calls, each become a session, so one batch tool can add thousands of entries to the all-projects listing. Keep that, or group them, for example per model per day?
4. Should an LM Studio session with a thread ID also appear in the single-project listing of the Codex session it served? Every project listing would then read the LM Studio logs.
5. Each CLI listing across projects reads every log file. Is that acceptable at about 3 seconds per 150 MB, or should apsis keep an index on disk so repeated `apsis --list` runs are fast?
6. A log file given directly produces a JSON array of sessions, while `apsis <id>` produces one session object. Is that acceptable, or should a log file render as a single session?
