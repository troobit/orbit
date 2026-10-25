# Local task runner

`local-next-task` and `orbit local` run the same Orbit implementation. They
replace the Python runner in agentic-coding without nesting execution loops.

For the usual `8-bit` workflow with an arbitrary task file, start with the
[implementation quick guide](local-implementation-quickstart.md).

## Setup

Install rune, a recent Codex CLI, and LM Studio. Load a model and start its API
server separately. For the model already installed on this machine:

```bash
lms load 8-bit --identifier 8-bit --context-length 32768 --yes
lms server start --port 1234 --bind 127.0.0.1
lms ps
make install-local-next-task
```

The installer defaults to `~/.local/bin` (`BIN_DIR` can override it). It installs
Orbit and a relative `local-next-task -> orbit` symlink. An existing symlink to
the agentic-coding prototype may be replaced; unrelated files are preserved.
Keep this directory on PATH. No user or project configuration is rewritten.

## Run

```bash
local-next-task ../project/specs/feature/tasks.md --dry-run
local-next-task ../project/specs/feature/tasks.md
orbit local ../project/specs/feature/tasks.md --phase --timeout 1h
```

The ledger determines the target Git root, independent of the caller's current
directory. Paths containing spaces are supported. The default is one task and
its descendants per iteration. Orbit passes the selected work, references and
exact rune progress/completion commands to the worker; it does not complete
tasks on the worker's behalf. Each session starts fresh and leaves edits for
review without committing or pushing.

| Option | Default | Meaning |
|---|---|---|
| `--model` | `8-bit` | Exact LM Studio model identifier |
| `--context-window` | `32768` | Token window actually loaded in LM Studio |
| `--compact-at` | `0` | History compaction threshold; 0 derives five eighths of the window (20480 by default) |
| `--network` | false | Permit network access for sandboxed shell tools, including dependency downloads |
| `--phase` | false | Execute the next phase instead of one task |
| `--selection` | `runner` | `model` delegates retrieval to the installed next-task skill |
| `--timeout` | `30m` | Positive Go duration per worker session |
| `--max-iterations` | `20` | Positive bound per invocation |
| `--dry-run` | false | Print selected work and prompt; no inference or logs |

Model selection mode requires `~/.agents/skills/next-task/SKILL.md`. It retains
the prototype's widening of a selected top-level task to its whole phase.
Phase mode permits dependencies that earlier tasks in the same selection can
satisfy. STOP gates and unmet dependencies outside the selection halt preflight.

Codex runs with `--oss --local-provider lmstudio --ignore-user-config
--ephemeral --json --sandbox workspace-write -c approval_policy="never"`.
The preset also passes `-c web_search="disabled"`,
`-c features.multi_agent=false`, `-c features.apps=false` and
`-c model_reasoning_summary="none"` on
every iteration. These disable built-in web search, delegation tools and
connected ChatGPT app tools, plus unused reasoning summaries. Shell/file tools
and project instructions remain available.

Each worker also receives `model_context_window`,
`model_auto_compact_token_limit`, `tool_output_token_limit=2048` and the chosen
`sandbox_workspace_write.network_access` value. The first two come from the
context flags above. These settings are independent of built-in web search.

The installed Codex must support those flags; see its `exec --help` and the
[official CLI reference](https://learn.chatgpt.com/docs/developer-commands?surface=cli).
The [official configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference)
describes the web-search, feature and reasoning settings.
Model calls use the local provider; Orbit does not add comparison, consolidation,
hosted pre/post prompts or shell hooks to this preset. Codex may still load
project instructions and tools supplied by its installation.

## Context and dependency blockers

`TruncateMiddle` is actual context loss. A worker can continue editing after
the server drops intermediate tool results or instructions, so continued output
does not establish that the conversation remains intact. Match
`--context-window` to the **loaded** context reported by `lms ps --json`, not
the model's advertised maximum. Orbit does not reload the model for you.
The [LM Studio Codex guide](https://lmstudio.ai/docs/integrations/codex)
recommends more than roughly 25k context because Codex itself uses substantial
context.

For a model loaded at 32768, the default budgets are explicit:

```bash
orbit local /path/to/specs/feature/tasks.md \
  --context-window 32768 --compact-at 20480 --max-iterations 1
```

The compaction margin reserves room for tool results, summary instructions and
generation. Tool outputs stored in history are limited to 2048 tokens; raw
stdout/stderr artifacts remain available. Tokenizers differ, and compaction
is lossy: these budgets reduce risk but cannot guarantee that every request
fits. In LM Studio's applicable model/prediction settings, prefer rejecting or
stopping at the context limit over silently truncating the middle; the exact
control depends on the runtime/API. Its SDK documents the
[overflow policies](https://beta.lmstudio.ai/docs/typescript/api-reference/llm-prediction-config-input).
Do not assume a client flag changes the server's policy. If truncation persists,
lower `--compact-at`, reduce task/context size, or deliberately reload a larger
supported context and then match Orbit's setting. Stop active work before a
reload. A longer timeout does not prevent truncation.

`turn.completed` token usage is cumulative over model calls in that turn.
For example, 476522 input tokens with 452096 cached tokens is not evidence of
a single 476522-token prompt. Use server per-request logs for that diagnosis.

Local inference and offline shell execution are separate choices. The default
workspace-write sandbox blocks tool networking. Prepare dependencies before
starting the worker, or explicitly pass `--network` when downloads are needed.
This retains filesystem restrictions, so caches still need a writable location
inside the workspace or allowed temporary directory. A missing package in an
offline cache cannot be repaired by repeated offline retries. Worker guidance
now asks for a halt on unavailable prerequisites and bounded document reads.

Run validation commands directly. `command | tail; echo "$?"` reports the
pipeline/tail status, which can hide the first command's failure. If piping is
necessary, enable `pipefail` and preserve the command's real status. A printed
`lock-exit=0` after `run_silent: exit 2` is not successful dependency resolution.

Repeated `codex_models_manager` decode errors are a separate catalog-format
compatibility issue. The observed worker continued calling tools despite them;
the explicit budgets avoid relying on inferred context metadata. This change
does not fix catalog decoding. `Reading additional input from stdin...` is
Codex's input notice; Orbit supplies EOF rather than waiting for user input.

## Smaller work units and alternate harnesses

For constrained local models, use one independently verifiable change per
top-level task. Split scaffolding, dependency preparation and the first behavior
test into separate work units where they have separate validation. Avoid
`--phase` and `--selection model` when minimizing session scope. Adding child
tasks alone does not shrink dispatch: the selected parent and all descendants
currently share one worker session. Orbit does not rewrite another project's
ledger automatically.

| Approach | Benefit | Limit or prerequisite |
|---|---|---|
| Smaller tasks with tuned Codex budgets | Fresh context per verified unit; retains the existing tool loop and logs | Prerequisites must still work; a single oversized read can still overflow |
| Larger LM Studio context | More room for instructions and tool history | Requires memory and a deliberate model reload; may increase latency |
| OpenCode against LM Studio | Custom provider/model limits and harness-owned compaction are available | Needs an adapter check: Orbit's current adapter expects one JSON document and does not forward local output writers or process-group cancellation |
| Direct LM Studio API/SDK integration | Full control of context, tools and overflow policy | Requires implementing or adopting a tool-execution loop, recovery and transcript support |

OpenCode is a candidate for a separate measured run, not established as a better
choice by the current evidence. Its [provider settings](https://opencode.ai/v2/docs/providers)
support local compatible endpoints and explicit model limits; its
[compaction settings](https://opencode.ai/v2/docs/compaction) expose summary and
retained-history controls. A fair comparison uses the same small coding task,
loaded context and validation, measuring successful tool calls, truncation,
recovery and completion rather than a text-only response.

## LM Studio compatibility warnings

Codex 0.164.0's native LM Studio integration still sends some Responses API
fields the server does not support. These warnings alone do not establish
that task text or references were discarded:

| Warning | Meaning and action |
|---|---|
| `Ignoring unsupported tool type(s): namespace, web_search` | Codex advertises delegation, connected-app namespaces and built-in web search. The preset disables all three. Connected apps were identified in a follow-up capture using the normal authenticated login; an empty Codex home did not reproduce them. Rebuild/reinstall with `make install-local-next-task`, check command paths and start a new Orbit invocation. An already-running parent keeps its original arguments. Other namespace sources still need investigation if the warning persists. |
| `prompt_cache_key ... ignored` | Codex sends a cache-routing key. This server does not use it; it is request metadata, not task context. This warning can remain. It does not prove that all local caching is disabled. |
| `Developer role detected - replacing it with system role` | LM Studio converts the instruction role. It reports replacement, rather than deletion of the instruction text. Codex's built-in instructions and execution constraints account for developer messages even with user configuration ignored. Keep those instructions; this warning can remain. |
| `include ... reasoning.encrypted_content ... ignored` | Codex requests encrypted reasoning data in the response. LM Studio does not provide that Responses API feature. This warning can remain even with reasoning summaries disabled; it is not a request to add more task context and does not disable the model's own reasoning. |

Request captures with this Codex version confirmed that
`model_supports_reasoning_summaries=false` does **not** remove the encrypted
include in OSS mode. `model_reasoning_summary="none"` removes the summary
request but leaves that include. `--ignore-user-config` isolates user
configuration; it does not remove built-in tools or instructions. There is no
verified preset setting here to remove the remaining fields. Orbit delegates
HTTP serialization to Codex and does not rewrite requests or filter warnings.

Disabling `features.apps` prevents unrelated connected-app schemas (such as
Pets and Sites) from entering the local request. It does not disable shell
networking: `--network` still permits dependency installation, testing and
updates. Authentication can affect tool availability even when user
configuration is ignored, so provider checks should include both an isolated
home and the usual authenticated profile.

After rebuilding, restart the iteration to use the new preset. Review whether
the worker runs shell/file tools, produces `turn.completed`, and completes the
selected ledger work. A failed turn or no progress still stops Orbit. If tools
are missing or work stalls, keep the warnings and the live stdout/stderr for
diagnosis rather than treating a successful text response as proof of tool
compatibility. See the [bugfix report](../../specs/bugfixes/local-provider-warnings/report.md)
and [request-capture results](../../specs/local-compute-agent/verification.md#protocol-compatibility-check).

## Diagnostics and recovery

Orbit writes standard summaries and index files under `.orbit/local/` beside
the task file. `phase-N` identifies the invocation's iteration, so two tasks
from the same source phase retain separate logs. Reinvocations use Orbit's run
numbering. Unique session artifacts retain each prompt, live stdout JSONL and
stderr, including interrupted output. Tail the printed stdout path; elapsed
time is reported every 30 seconds even while the model produces no events.
Ephemeral Codex sessions are not discoverable through `apsis list`; the live
JSONL and Orbit logs are the retained artifacts.

The runner stops after the first failure, timeout, halt signal, changed graph,
reopened completed task, unrelated task advancement or iteration with no
selected task completed. A success message alone is insufficient. Local
failures are not automatically retried because edits may already exist.

Inspect the last prompt/output, diff and ledger before rerunning the same
command. Keep `.orbit/halt` until its reason is resolved, then remove it
explicitly. The next run selects remaining work from the ledger. Unix timeout
and Ctrl+C terminate the worker process group; Windows terminates the direct
worker process.

The Python prototype's numeric `--timeout 1800` becomes `--timeout 30m` (or
`1800s`). The old Make wrapper in agentic-coding remains that repository's
prototype; use this installed command or `orbit local` for Orbit execution.

## Verification

Run `make test-local`, `make test`, and `make lint`. Integration tests use real
rune and a fake Codex in disposable Git repositories; they do not invoke an
inference server. Live-model observations are recorded in
`specs/local-compute-agent/verification.md`.
