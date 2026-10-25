# Bugfix Report: Local provider warnings

**Date:** 2026-10-25
**Status:** Fixed; remaining upstream protocol limits documented

## Description of the Issue

Local iterations against LM Studio report unsupported `namespace` and
`web_search` tools, ignored `prompt_cache_key` and
`reasoning.encrypted_content`, and developer-to-system role conversions.
Setup guidance did not explain which warnings indicate unnecessary tools and
which describe protocol compatibility.

**Reproduction steps:**

1. Run `orbit local <tasks.md>` with Codex 0.164.0 and LM Studio's `8-bit` model.
2. Inspect the LM Studio server warnings during the iteration.
3. Alternatively, direct `CODEX_OSS_BASE_URL` to a loopback Responses stub and
   capture Codex's outgoing JSON using the same CLI arguments.

**Impact:** Unsupported tools are advertised, adding avoidable overhead and
misrepresenting available capabilities. The warnings also suggest lost context
without explaining the actual compatibility boundary.

## Investigation Summary

- Inspected the local CLI preset, Codex argument construction, local prompts,
  integration tests and existing setup guidance.
- Captured real Codex requests with a disposable `CODEX_HOME` and working
  directory against a loopback HTTP stub; no inference or credentials needed.
- The default request includes a `multi_agent_v1` namespace, `web_search`,
  `reasoning.summary="auto"`, a cache key, two developer messages and the
  encrypted reasoning include.
- Disabling web search and multi-agent support removes both unsupported tool
  types in the isolated profile. `model_reasoning_summary="none"` removes the summary request.
- Tested `model_supports_reasoning_summaries=false`: OSS mode still sends
  the summary and encrypted include. Setting summary to `none` still leaves
  the encrypted include. Neither is a fix for that include warning.
- The dispatch regression fails for both task and phase mode before the fix
  because the local compatibility overrides are absent.

## Discovered Root Cause

**Defect type:** Integration configuration and missing compatibility guidance.

Orbit assumed the native local provider restricted all capabilities to those
LM Studio supports. Ignoring user configuration isolates user settings but
does not disable Codex's built-in web search, delegation or reasoning defaults.
LM Studio drops unsupported tool definitions and metadata; Orbit does not
construct these HTTP requests itself. Developer-role replacement does not by
itself imply instruction text was discarded.

## Resolution for the Issue

**Changes made:**

- The local preset explicitly sets `web_search="disabled"`,
  `features.multi_agent=false`, `features.apps=false` and
  `model_reasoning_summary="none"`.
- The dispatch integration test checks those actual arguments on every task
  and phase invocation, alongside existing progress and log checks.
- Setup documentation explains every warning, how to apply the updated preset,
  and how to diagnose tool failures. The feature spec and agent notes record
  the version-specific compatibility boundary.

**Approach rationale:** Restrict capabilities where Codex settings work, retain
necessary instructions, and explain fields still emitted by Codex. These fields
are generated below Orbit's agent adapter; suppressing their log messages would
hide useful diagnostic information.

**Alternatives considered:**

- Documentation only: rejected because unsupported tool definitions can be
  removed with explicit settings.
- `model_supports_reasoning_summaries=false`: rejected after request capture
  showed it ineffective in this OSS path.
- Rewrite HTTP requests in a proxy or remove instructions: rejected as
  unnecessary protocol ownership and a risk to execution constraints.

## Regression Test

**Test file:** `cmd/orbit/local_test.go`
**Test name:** `TestLocalCommandDispatch`

Verifies explicit compatibility settings reach every worker invocation in both
task and phase mode, while the existing ledger and transcript assertions
continue to pass.

**Run command:** `make test-local`

## Affected Files

| File | Change |
|---|---|
| `cmd/orbit/local.go` | Local capability overrides |
| `cmd/orbit/local_test.go` | Actual-argv regression assertions |
| `docs/runbooks/local-next-task.md` | Warning meanings, limits and recovery |
| `docs/runbooks/local-implementation-quickstart.md` | Reinstall and warning guidance |
| `docs/agent-notes/local-execution.md` | Preserve verified compatibility behavior |
| `specs/local-compute-agent/` | Requirements, design, Decision 14, rune task and verification |

## Verification

- [x] Regression fails before the fix for missing local compatibility settings.
- [x] Regression passes after the fix (`make test-local`).
- [x] Full test suite, build and lint pass (`make build test lint`).
- [x] Installed Codex request capture confirms capability removal and tool use.

**Manual verification:** A rebuilt Orbit drove installed Codex through two
dependent tasks against a loopback Responses stub. Codex executed two real
shell calls, ran the scratch Makefile twice, completed both tasks through rune
and produced two completed-turn transcripts. Four worker requests contained
only function tools and retained an `AGENTS.md` instruction marker. Cache keys,
developer messages and the encrypted include remained. No new model inference
was run for this protocol check; the earlier live-model result is recorded
separately in the feature verification.

`make install-local-next-task` updated the existing installed Orbit binary and
compatibility link. The installed binary matches the checked build, and
`local-next-task --help` succeeds. Changed Markdown links, rune parsing and
`git diff --check` also pass.

## Prevention

Verify local-provider defaults at the HTTP boundary when updating Codex.
Include the normal authenticated profile as well as a clean Codex home.
Distinguish unsupported tools from ignored metadata and preserved instruction
text. Do not add flags solely from their names or hide server warnings.

## Authenticated-profile follow-up

The longer user-started run still reported `namespace` after delegation was
disabled. A request capture using the normal login reproduced seven connected
ChatGPT app namespaces, including Pets, Sites and app management. The original
disposable-CODEX_HOME fixture omitted this authentication-dependent behavior.

Setting `features.apps=false` removed all seven namespace definitions in that
capture while retaining seven function tools and enabled sandbox tool networking.
The dispatch regression first failed on the missing setting before the fix.
The remaining cache, role and encrypted-include warnings are unchanged. This
fix applies to newly started Orbit invocations; it cannot change the arguments
of the already-running parent process.

## Related

- [Local compute design](../../local-compute-agent/design.md)
- [Decision 14](../../local-compute-agent/decision_log.md)
- [Local runner runbook](../../../docs/runbooks/local-next-task.md)
