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
