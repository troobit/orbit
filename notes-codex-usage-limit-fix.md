# Scratch: Codex usage-limit fix — ready to upstream

Status: committed locally, NOT pushed. Branch `fix/codex-usage-limit-wait`
(commit `adb2ad8`), based directly on `upstream/main` (`7259a6c`), so it can be
pushed to the `troobit/orbit` fork and PR'd against `ArjenSchwarz/orbit` as-is.

## Problem

Codex reports quota exhaustion on stderr as:

    ERROR: You've hit your usage limit. Upgrade to Pro (https://chatgpt.com/explore/pro), visit https://chatgpt.com/codex/settings/usage to purchase more credits or try again at 4:18 PM.

and sometimes with a full date:

    ERROR: You've hit your usage limit. Upgrade to Pro (https://chatgpt.com/explore/pro), visit https://chatgpt.com/codex/settings/usage to purchase more credits or try again at Sep 5th, 2026 1:39 AM.

The codex classifier (`internal/agents/codex/errors.go`) had no pattern for
this, so the error fell through to generic retries and failed the phase.
claude-code already handled its equivalent ("You've hit your limit · resets
3am") by waiting until the reset (`ErrorClassRateLimitWait`).

Observed in the wild twice during a variant run in the spare repo
(specs/ui/review-pages): variant 3 (codex) died mid-phase both times; the logs
above are verbatim from `.orbit/logs/variant-3/phase-3-session.txt` and
`phase-4-run-2-session.txt` there.

## Change

- `internal/agents/codex/errors.go`: detect "hit your usage limit" before the
  generic rate-limit check; parse "try again at <time>" (bare clock time rolls
  to tomorrow if past; dated form supports ordinal days and full/abbreviated
  month names; explicit past date retries after 1 minute). Times are local —
  Codex prints them in the user's timezone. Returns `ErrorClassRateLimitWait`
  so the retry loop sleeps until reset and resets the attempt counter.
- `internal/agents/errors.go`: new shared `NewRateLimitWaitError` constructor +
  `FormatWaitDuration` (moved verbatim from claudecode's private
  `formatDuration`), matching the existing `New*Error` constructor pattern.
- `internal/agents/claudecode/errors.go`: now uses the shared constructor;
  private `formatDuration`/`pluralS` deleted.
- Tests: classifier tests with the real messages (future-dated so they don't
  rot), direct `parseUsageLimitRetryTime` tests against a pinned clock, moved
  `TestFormatDuration` → `agents.TestFormatWaitDuration`, added constructor
  test. CHANGELOG entry under [Unreleased] → Fixed.

## Verification

- `go test ./internal/agents/... ./internal/orbit -count=1` passes; full
  `make test` passes except `apsisweb TestIntegrationEndToEnd` (shutdown
  deadline), which fails identically on a pristine upstream/main — pre-existing,
  unrelated.
- `go vet` clean; `gofmt` clean on touched files (the rewrite of
  codex/errors.go incidentally fixed its pre-existing trailing-blank-line lint;
  `internal/agents/agent.go` and kiro/opencode files were already unformatted
  upstream and were left alone).
- `golangci-lint` not installed locally, so `make lint` was not run.
- Local binary rebuilt from the branch: `orbit v0.9.0-38-gadb2ad8` (this is
  what's on PATH now, one commit ahead of the upstream-main build).

## Remaining steps (deliberately not done)

1. `git push origin fix/codex-usage-limit-wait` (to the troobit fork).
2. Open PR against ArjenSchwarz/orbit. Suggested framing: quote the two
   verbatim error messages above as the motivating case, note the shared
   constructor keeps claude-code behavior identical, and mention the
   apsisweb test failure is pre-existing (reproducible on main) if CI trips
   on it.
3. Consider whether upstream wants `golangci-lint` run first.
