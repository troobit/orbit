# LM Studio server log fixture provenance

All fixtures in this directory are synthetic. None was copied from a real
log: content, identifiers, paths and model names (`example-model-a`,
`thread-alpha` and so on) are invented. Two kinds of evidence shaped them, and
they are kept apart below (requirement 8.4).

## Reported log grammar (LM Studio 0.4.26)

LM Studio publishes no specification for its disk server log. The record
structure below comes from earlier structural inspections reported in the
spec's handoff. They are observations, not a documented contract.

| Structure | Used in |
|---|---|
| Header line `[YYYY-MM-DD HH:MM:SS][LEVEL][tag] message`, local time without a zone, zero or more bracketed tags | every fixture |
| `Received request: POST to <path> with body {` followed by pretty-printed JSON up to the next header | every request record |
| `Generated prediction:` and `Generated synchronous v1/responses response:` introduce a logged response body | `endpoints.log`, `classification.log`, `threads.log`, `tools.log` |
| `Returning {` records carry model lists, not inference output | `classification.log`, `notice_only.log` |
| Model tags on follow-up records, e.g. `[example-model-a]` | responses, warnings and errors |
| Long strings cut to head and tail as `... <Truncated in logs> ...` | `tools.log` (user message), `threads.log` (thread ID and turn-metadata blob) |
| Codex `/v1/responses` bodies carry `prompt_cache_key` and a flat string `client_metadata` map | `threads.log` |
| Startup notices listing endpoints, `GET` requests | `notice_only.log`, `classification.log` |

Inventions that no report covers, labelled so they are not mistaken for
observations:

- `classification.log`: the `/v2/future/endpoint` path, the WARN-level
  request header and the WARN-level `/v1/responses/input_tokens` record
  exercise classification precedence, not observed traffic. The WARN and
  ERROR message texts are invented.
- `recovery.log`: the text before the first header, the header without a
  level, the unterminated and trailing-comma bodies, the array body and the
  body unfinished at end of file are deliberate damage.
- `kiro_marker.log`: the `metadata` object holds the keys apsis's Kiro
  heuristics look for (`conversation_id`, `history`, `executionId`, `chat`),
  to check that a valid LM Studio signature wins.
- `notice_only.log` models a rotated file holding no included request.
- `tools.log`: the reused `call_1` across two requests and a response, and the
  unmatched `toolu_missing` result, model colliding tool identifiers.

## Documented endpoint payloads

Request and response body shapes follow public documentation. The disk-log
form of `/v1/messages` and `/api/v0/completions` traffic was not established
by the reported inspections; those records apply the reported request-line
grammar to the documented payloads.

| Endpoint or shape | Source | Used in |
|---|---|---|
| `/v1/chat/completions`, `/v1/completions` requests and responses | [OpenAI-compatible endpoints](https://lmstudio.ai/docs/developer/openai-compat) | `endpoints.log`, `classification.log`, `tools.log` |
| `/v1/responses` requests (`instructions`, `input`, `function_call`, `function_call_output`, `reasoning`) and synchronous responses | [Responses](https://lmstudio.ai/docs/developer/openai-compat/responses) | `endpoints.log`, `threads.log`, `tools.log` |
| `custom_tool_call`, `custom_tool_call_output`, `web_search_call`, `input_image` items | OpenAI Responses item types as sent by Codex; LM Studio documents no specialised handling | `tools.log` |
| `/v1/messages` requests (`system`, `messages`, `thinking`, `tool_use`, `tool_result`, `image` blocks) and Messages responses | [Anthropic-compatible endpoints](https://lmstudio.ai/docs/developer/anthropic-compat) | `endpoints.log`, `tools.log` |
| `/api/v0/chat/completions`, `/api/v0/completions` (string and array `prompt`) | [REST API v0](https://lmstudio.ai/docs/developer/rest/endpoints) | `endpoints.log` |
| `/api/v1/chat` string `input`, typed `text`/`image` input items, `system_prompt`, `previous_response_id`; response `output` (`message`, `reasoning`, `tool_call`, `invalid_tool_call`) and `stats` | [Native chat](https://lmstudio.ai/docs/developer/rest/chat), [stateful chats](https://lmstudio.ai/docs/developer/rest/stateful-chats) | `endpoints.log`, `tools.log` |
| `/v1/decisions` | [Decisions](https://lmstudio.ai/docs/developer/openai-compat/decisions) | `classification.log` |
| `/v1/embeddings`, `/v1/responses/input_tokens`, `/v1/messages/count_tokens`, `/api/v1/models/unload` (excluded paths) | OpenAI-compatible, Anthropic-compatible and [REST API](https://lmstudio.ai/docs/developer/rest) endpoint lists | `classification.log` |
| `client_metadata` keys `thread_id`, `session_id`, `turn_id`, `x-codex-installation-id`, `x-codex-window-id`, `x-codex-parent-thread-id`, and `request_kind`/`thread_id` inside the `x-codex-turn-metadata` JSON string | Codex pinned [`responses_metadata.rs`](https://raw.githubusercontent.com/openai/codex/rust-v0.166.0/codex-rs/core/src/responses_metadata.rs) and [`client.rs`](https://raw.githubusercontent.com/openai/codex/rust-v0.166.0/codex-rs/core/src/client.rs) header names | `threads.log` |
