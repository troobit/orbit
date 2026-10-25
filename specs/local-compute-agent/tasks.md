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
