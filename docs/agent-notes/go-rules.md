# Go Rules

## Time-Zone Interpretation

- Validate explicit time-zone settings when an input format relies on them. On Unix, Go can silently use UTC after a failed `TZ` load; a usable `time.Local` does not prove that the requested zone loaded.
- Distinguish unset `TZ` (host zone) from explicitly empty `TZ` (UTC). Portable input parsers must define supported override forms and expose failed interpretation rather than claim an unintended offset.
- Standard IANA zones need available zone data. The standard-library [time/tzdata package](https://pkg.go.dev/time/tzdata) can provide embedded fallback data; the main program should choose that integration rather than silently relying on a system database. This does not itself make the host-local zone API honour explicit overrides on every platform.

- `time.LoadLocation("")` returns UTC and `time.LoadLocation("Local")` returns the host zone, both without error. Validate those names before treating a load as a successful explicit override.
- A blank `time/tzdata` import is a fallback, not an override: `LoadLocation` still consults `ZONEINFO` and the system zone directories first, so a name a case-insensitive macOS zone directory accepts can still fail where only embedded data exists.
- `time.Date` picks an unspecified offset for nonexistent or ambiguous wall times. To detect them, try the zone's offsets at the wall time and 24 hours either side, and keep the result only when exactly one distinct candidate maps back to the same wall time.

Evidence: [Go's Unix local-zone initialization](https://go.dev/src/time/zoneinfo_unix.go). LM Studio's proposed outcome is wall-time-only output with a visible warning; see [requirements 5.1–5.5](../../specs/apsis-lmstudio-logs/requirements.md#5.1).

## File Change Detection

- Size and modification time alone do not establish file identity or detect every ordinary replacement preserving timestamps. Cache contracts that require both replacement detection and zero content reads need platform metadata adapters.
- This module already depends on `golang.org/x/sys`; its Windows package exposes `GetFileInformationByHandleEx`, `FileBasicInfo` and `FileIdInfo`. Their presence establishes an API path, not successful behaviour on every filesystem. Native verification must precede a claim that the cache meets its contract.

- For Unix file identity, use `golang.org/x/sys/unix.Stat_t` rather than `syscall.Stat_t`. The x/sys struct names the times `Mtim`/`Ctim` on Linux, Darwin, the BSDs and AIX, where `syscall` uses `Mtimespec` on Darwin. `Timespec.Nano()` is available on every Unix build. Field types still differ (`Dev` is `int32` on Darwin and `uint64` on Linux), so convert explicitly. Check the other release platforms with `GOOS=darwin go vet` and `GOOS=windows go vet`.

Evidence: Microsoft's [file identity](https://learn.microsoft.com/en-us/windows/win32/api/winbase/ns-winbase-file_id_info) and [metadata query](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-getfileinformationbyhandleex) documentation; [shared design Decision 23](../../specs/apsis-shared-json/decision_log.md).

## Property Tests with rapid

- `*rapid.T` does not implement `testing.TB` (with Go 1.27 it lacks `ArtifactDir`, among others). A helper shared between `*testing.T` and `*rapid.T` should take a small interface, such as `interface{ Helper(); Fatalf(string, ...any) }`.

## Follower Cancellation Tests

- `transcript.Follower.Run` returns nil on context cancellation. Tests must assert the error received from their completion channel; discarding an error-valued receive fails `errcheck` even when shutdown is expected to succeed.
- Managed worktrees may deny writes to the default Go/linter caches. Use writable temporary `GOCACHE` and `GOLANGCI_LINT_CACHE` directories with the repository's Makefile commands. A cache permission failure does not establish a source/build failure.

## Server Shutdown in Tests

- `http.Server.Shutdown` waits for connections in `StateNew` (dialled, no request read) for up to 5 seconds before treating them as idle. `http.DefaultClient` can dial a spare keep-alive connection it never uses, so a test that calls `Shutdown` with a 5-second deadline after `http.Get` calls fails intermittently with `context deadline exceeded`. Call `http.DefaultClient.CloseIdleConnections()` (or the test client's) before `Shutdown`.

Evidence: [`net/http` `closeIdleConns`](https://go.dev/src/net/http/server.go) treats `StateNew` connections as idle only after 5 seconds; `TestIntegrationEndToEnd` failed 4 of 4 package runs before the fix and passed 6 of 6 after (Q119).
