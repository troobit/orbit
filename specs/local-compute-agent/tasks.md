---
references:
    - specs/local-compute-agent/requirements.md
    - specs/local-compute-agent/design.md
    - specs/local-compute-agent/decision_log.md
---
# Local compute agent tasks

## Implementation

- [x] 1. Add local preset, task dispatch and iteration guards

- [x] 2. Capture live diagnostics and handle worker failures and cancellation

## Verification

- [x] 3. Verify behavior with real rune and fake Codex integration tests
  - Real-rune integration covers task/phase/model dispatch
  - nested tasks
  - dependencies
  - STOP and halt signals
  - graph and status changes
  - failures
  - timeout
  - iteration limit and resume
  - and relative paths with shell punctuation. Existing HTTP test bodies are drained and follower errors asserted so repository checks pass with current tooling.

- [x] 4. Run local model smoke test and document migration and results

- [x] 5. Write a quick guide for arbitrary task files using the LM Studio 8-bit model
  - Quick guide uses an explicit 8-bit model for setup, preview, implementation and recovery. Shell examples and local links validated.
  - References: docs/runbooks/local-implementation-quickstart.md

- [x] 6. Include in-progress tasks when checking local run completion
  - Regression reproduced with a final in-progress task. Local completion now checks all statuses. make build test lint passed; installed dry-run selects the unfinished task without mutations.

- [x] 7. Disable unsupported local tools and document Codex protocol warnings
  - Dispatch regression failed before the fix and passed afterward. Real Codex 0.164.0 loopback captures verified removal of unsupported tool types, retained protocol metadata, and two actual shell/Makefile/rune iterations. make test-local, make build test lint, Markdown links and rune parsing passed. Installed launcher updated and checked.
  - References: specs/bugfixes/local-provider-warnings/report.md, specs/local-compute-agent/verification.md

- [x] 8. Bound local context and correct dependency and completion handling
  - Verified explicit context and compaction budgets, opt-in sandbox networking and corrected failed-attempt completion counts. Real-Codex loopback checks activated compaction and completed two tasks with shell/Makefile/rune, including HTTP tool calls with networking enabled. Build, tests, lint, document links and rune parsing passed; installed launcher updated. Documented task sizing and harness alternatives; visual activity rendering excluded.
  - References: specs/bugfixes/local-context-progress/report.md, specs/local-compute-agent/verification.md

- [x] 9. Exclude authenticated app namespaces and record the longer live run
  - Authenticated capture reproduced seven connected-app namespaces; apps=false removed them while retaining shell tools and network access. Rebuilt authenticated fixture completed two tasks and HTTP checks; build, tests and lint passed. Observed live iteration 1 complete after real compaction (20243 to 10947 server tokens), three tests and Ruff checks, in 1098.955s with no middle truncation. Orbit accepted the ledger and began iteration 2; overall run continues undisturbed.
  - References: specs/bugfixes/local-provider-warnings/report.md, specs/local-compute-agent/verification.md

- [x] 10. Clear previous terminal state when restarting a flat-log run
  - Existing-summary regression reproduced stale error and completion timestamp on a running restart. New starts clear terminal fields while preserving history; build, tests and lint pass. Active user summary left untouched.
  - References: specs/bugfixes/local-context-progress/report.md

- [x] 11. Add optional leaf dispatch and measure repeated local coding runs
  - Added runner-only --leaf with read-only ancestor context, dependency and STOP guards, tests-first guidance and integration coverage. make test-local and make build test lint passed; installed command checked. Benign live baseline completed in 378.773s; reviewed red/green leaf sessions completed in 396.267s and 359.476s with unchanged tests and automatic parent completion. Leaf handoff works but was slower on this small supervised sample; native delegation remains unsupported.

## Local worker instruction budgets

- [x] 12. Allow an explicit local worker base-instruction file
  - Scope: cmd/orbit/local.go and local_test.go; update local runbook and verification with actual M5 observations. Defaults, selected task prompts and sandbox flags remain unchanged.
  - Requirements: [15.1](specs/local-compute-agent/requirements.md#15.1)
  - [x] 12.1. RED: Verify instruction-file dispatch and invalid-file preflight <!-- id:hcum1hw -->
    - Use the existing real-rune/fake-Codex CLI fixture. Assert absolute path configuration on every worker, including relative paths with spaces/quotes, and unchanged selected prompt/routing/sandbox. Missing, directory, empty and invalid UTF-8 inputs must not invoke Codex or create logs.
    - Acceptance: make test-local fails on the new instruction-file assertions before implementation.
    - Requirements: [15.1](specs/local-compute-agent/requirements.md#15.1)
  - [x] 12.2. GREEN: Add the validated instruction-file option <!-- id:hcum1hx -->
    - Add --instructions-file, resolve/validate it before logs and pass the supported model_instructions_file override without replacing selected task prompts. Document compact worker use and runtime reasoning-default findings.
    - Acceptance: unchanged new tests, make test-local, make test, make lint and make build pass. Verify real M5 runs separately; do not claim success from fake agents.
    - Blocked-by: hcum1hw (RED: Verify instruction-file dispatch and invalid-file preflight)
    - Requirements: [15.1](specs/local-compute-agent/requirements.md#15.1)

## Local execution reliability

- [x] 13. Bound rune queries and cancel stalled ledger subprocesses
  - All rune queries now use a 30-second deadline, shutdown context and shared process-group cleanup. Failure envelopes and malformed or missing task payloads fail closed. Cancellation/child-process and payload regressions passed; make build test lint and make test-local passed.
  - References: specs/local-compute-agent/design.md

- [x] 14. Complete arbitrary ledger paths without a fixed default session ceiling
  - Default max-iterations zero derives a finite budget from initial incomplete tasks. RED reproduced the twenty-session cutoff; GREEN completes twenty-one tasks. Explicit caps, negative input, custom nested filenames, punctuation, relative paths and symlinks passed. Codex startup now fails within five seconds before logs. Updated command installed.

- [x] 15. Verify both local models and record independent acceptance
  - Both real two-task fixtures completed with exit zero and eight unchanged host tests passing: 8-bit 452.775s; Bonsai 503.738s. Both models returned completed independent code reviews. Final installed 15s timeout probes exited in 16.246s/15.571s without surviving workers or inference. Bonsai recovery passed in 216.645s; an extra 8-bit recovery hit its 600.156s deadline after completing correct code/statuses, then independent host acceptance and a no-inference replay finalised the ledger. That failed worker is retained and explicitly not counted as a successful recovery. All artifacts and limitations documented; models idle, no experiment workers remain.
  - References: specs/local-compute-agent/verification-reliability.md
