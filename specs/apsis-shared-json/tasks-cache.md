---
references:
    - specs/apsis-shared-json/requirements.md
    - specs/apsis-shared-json/design.md
    - specs/apsis-shared-json/decision_log.md
---
# Apsis shared JSON: unchanged-refresh cache (PR 4)

## PR 4: Unchanged-refresh cache

- [x] 1. Share the Unix file fingerprint and test it before caching <!-- id:9kwhroi -->
  - The LM Studio cache's `statFingerprint` becomes the one fingerprint in `internal/sessions` (Decision 26); no new package or module change.
  - Stream: 3
  - Requirements: [5.9](requirements.md#5.9)
  - [x] 1.1. Write failing fingerprint and WAL-change tests <!-- id:9kwhrpa -->
    - Unix: an untouched file keeps its fingerprint; an equal-size replacement renamed over the file with its old mtime restored, and an equal-size in-place rewrite with mtime restored, both change it; a present and an absent companion differ
    - Unsupported or failing metadata is never a cache hit and never a size/mtime-only key (Decision 23)
    - Design risk 2: two writes through a WAL-mode `modernc.org/sqlite` connection each change the combined fingerprint of `data.sqlite3`, `-wal` and `-journal`
    - Stream: 3
    - Requirements: [5.9](requirements.md#5.9)
    - References: internal/sessions/cache_test.go
  - [x] 1.2. Move the fingerprint out of the LM Studio cache <!-- id:9kwhrpb -->
    - `fileFingerprint`, `fingerprintFile` and `statFingerprint` move to `fingerprint.go`, `fingerprint_unix.go` and `fingerprint_other.go`; add a companion fingerprint that records absence
    - Run the tests on a real Unix filesystem. If the WAL test fails, record the fallback for task 16: run the Kiro discover query on every refresh and count its page reads against 5.9
    - Blocked-by: 9kwhrpa (Write failing fingerprint and WAL-change tests), 9kwhron
    - Stream: 3
    - Requirements: [5.9](requirements.md#5.9)
    - References: internal/sessions/fingerprint.go, internal/sessions/fingerprint_unix.go, internal/sessions/fingerprint_other.go

- [x] 2. Cache listing metadata across apsis serve refreshes <!-- id:9kwhroj -->
  - internal/sessions cache and long-lived web Lister; complete only after the fingerprint tests pass and hard zero-unchanged-body-read assertions for every source.
  - Blocked-by: 9kwhroh, 9kwhroi (Share the Unix file fingerprint and test it before caching)
  - Stream: 2
  - Requirements: [5.1](requirements.md#5.1), [5.9](requirements.md#5.9)
  - [x] 2.1. Write failing zero-read refresh tests <!-- id:9kwhrpc -->
    - A counting opener injected into the Lister: a second `ListAll` over unchanged inputs reads zero transcript-body bytes for Claude, Codex, Copilot, Kiro IDE and codex-exec; a counting Kiro discover function is not called while the database fingerprints are unchanged (all-projects listing today calls `logs.DiscoverAll` directly, so it needs the same injection point as the project-scope discover)
    - The next pass reflects a content edit, an equal-size preserved-mtime replacement, an added and a removed file, a companion-only change (workspace.yaml, `-wal`) and a newly created nearer repository; unaffected exec identifiers stay the same
    - A changed file whose extraction fails is omitted with a warning, not served from stale metadata; a fingerprint error is a warning, never a hit; a key that changes during extraction is used for that pass and not cached
    - Reversing a returned slice, as the web handler does, does not change the next pass; concurrent `ListAll` calls are serialised (run under `-race`)
    - Benchmark a cold multi-source listing with generated Kiro IDE `.chat` fixtures (design risk 4)
    - Benchmark evidence includes generated fixture source counts, chat/message counts, sizes, platform/filesystem, cold and warm state, command and measurements. Investigate a local 1.5 s budget only as recorded evidence; choose the narrow Kiro IDE streaming decoder only when measured improvement justifies it. No host-sensitive timing acceptance assertion.
    - Blocked-by: 9kwhroh, 9kwhroi (Share the Unix file fingerprint and test it before caching)
    - Stream: 2
    - Requirements: [5.1](requirements.md#5.1), [5.9](requirements.md#5.9)
    - References: internal/sessions/lister_test.go, internal/sessions/cache_test.go
  - [x] 2.2. Implement the fingerprint-keyed cache in the Lister <!-- id:9kwhrpd -->
    - internal/sessions/cache.go: one mutex held for the whole pass; enumerate and fingerprint without opening bodies; entries keyed by path with companion fingerprints including absence; restat after extraction; sweep unseen entries; return fresh slices
    - The long-lived Lister in apsis serve carries the cache; CLI commands keep a per-command Lister
    - Apply the recorded fallbacks if the task 13 or task 16 benchmarks or the task 15 WAL test require them
    - CHANGELOG `[Unreleased]`: unchanged serve refreshes no longer re-read transcripts
    - Do not claim 5.9 until task 15.2 passes. If the WAL assumption fails, audit metadata-only discover queries/page reads and preserve the zero transcript-body-byte assertion; any relaxation requires a requirements decision. Run make test, make lint and make build; use -race for the concurrent cache test.
    - Blocked-by: 9kwhrpc (Write failing zero-read refresh tests)
    - Stream: 2
    - Requirements: [5.1](requirements.md#5.1), [5.9](requirements.md#5.9)
    - References: internal/sessions/cache.go, internal/sessions/lister.go, internal/apsisweb/server.go, CHANGELOG.md
