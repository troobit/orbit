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
`-c features.multi_agent=false` and `-c model_reasoning_summary="none"` on
every iteration. These disable built-in web search, delegation tools and
reasoning summaries that the local workflow does not use. Shell/file tools
and project instructions remain available.

The installed Codex must support those flags; see its `exec --help` and the
[official CLI reference](https://learn.chatgpt.com/docs/developer-commands?surface=cli).
The [official configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference)
describes the web-search, feature and reasoning settings.
Model calls use the local provider; Orbit does not add comparison, consolidation,
hosted pre/post prompts or shell hooks to this preset. Codex may still load
project instructions and tools supplied by its installation.

## LM Studio compatibility warnings

Codex 0.164.0's native LM Studio integration still sends some Responses API
fields the server does not support. These warnings alone do not establish
that task text or references were discarded:

| Warning | Meaning and action |
|---|---|
| `Ignoring unsupported tool type(s): namespace, web_search` | Codex defaults advertise delegation in a `multi_agent_v1` namespace and built-in web search. The local preset now disables both. If these persist, rebuild/reinstall with `make install-local-next-task` from the Orbit checkout and check that `command -v orbit` / `command -v local-next-task` resolve to the new install. Project or installation tools may need separate investigation if they introduce another namespace. |
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
