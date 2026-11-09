---
references:
    - requirements.md
    - design.md
    - decision_log.md
---
# Run Termination Guarantees

## Phase 1: Phase Loop Termination

- [ ] 1. Add a blocked terminal state to the variant status enum
  - New status distinct from completed and failed, meaning "stopped, remaining work needs a human" (Decision 2)
  - Update internal/variants/manager.go status values and the variants.json metadata it writes
  - Update the status display so the new state renders distinctly
  - Requirements: 1.3

- [ ] 2. Add a progress fingerprint helper
  - Derive from the GetNextPhase result already fetched by the loop: incomplete task count plus next-task identity
  - Compare count and identity together, so a phase that completes one task and adds another still reads as progress (Decision 1)
  - Unit tests for unchanged, advanced, and count-steady-but-identity-changed cases
  - Requirements: 1.1

- [ ] 3. Stop the variant phase loop on a stalled iteration
  - Capture the fingerprint before runVariantPhaseWithRetry and recompare after a successful return
  - On no change, stop with the task 1 blocked state; do not begin another iteration
  - Skip the check when the phase returned an error, since that path already exits
  - internal/orbit/variants.go:615-719
  - Requirements: 1.1, 1.2, 1.3, 1.6

- [ ] 4. Record a stop reason naming the phase and blocking tasks
  - Reason text must identify the phase and the tasks that remained incomplete
  - Persist it where a failure error is persisted today, so orbit status can surface it
  - Requirements: 1.4

- [ ] 5. Apply the same stall detection to the single-run phase loop
  - internal/orbit/single.go:771-837 has the identical shape; its ListPending guard does not compare across iterations
  - Requirements: 1.5

- [ ] 6. Add a configurable iteration ceiling to both phase loops
  - Default high enough that no legitimate multi-phase run reaches it; observed pathological runs hit 1223 and 1394
  - Record ceiling exhaustion distinctly from a fingerprint stall, since reaching it means the fingerprint missed something
  - Requirements: 2.1, 2.2, 2.3, 2.4

- [ ] 7. Integration test: an agent that never completes a task
  - Mock agent exits 0 without changing task state; assert the loop stops after a bounded number of iterations
  - Assert the recorded state is blocked, not failed, and the reason names the phase
  - Uses the existing mock-agent framework; no real CLI invocation
  - Requirements: 1.2, 1.3, 2.2

## Phase 2: Halt Sentinel

- [ ] 8. Define and document the halt sentinel path
  - Path lives in the variant's orbit-owned log directory, not the worktree (Decision 5)
  - Document it in CLAUDE.md and the agent-facing docs so agent authors can find it
  - Note explicitly that a worktree-relative .orbit/halt is NOT read, since an agent already wrote one there
  - Requirements: 3.1, 3.2

- [ ] 9. Check the sentinel after each phase and clear it at variant start
  - Present after a phase completes: stop the variant with the blocked state
  - Read the file contents and record them as the stop reason
  - Clear or ignore a sentinel left by a previous run so it cannot halt a later one
  - Requirements: 3.3, 3.4, 3.5

- [ ] 10. Verify the sentinel is optional
  - Test that stall detection stops a stalled run when no sentinel is written
  - Requirements: 3.6

## Phase 3: Error Classification

- [ ] 11. Parse the agent's structured error fields
  - Read terminal_reason, api_error_status and is_error from the Claude Code JSON result; none are referenced in the Go source today
  - Surface them on the result type the classifier receives
  - Requirements: 4.1

- [ ] 12. Add the human-action-required error class
  - Not retried and not waited on; carries an attributable, actionable message (Decision 4)
  - Note the upstream-divergence cost of widening the shared enum; ErrorClassFatal is the recorded fallback if that cost is judged too high
  - Update every agent switch on ErrorClass
  - Requirements: 4.3

- [ ] 13. Classify credit exhaustion from the structured fields
  - Structured signal is primary; message text refines it, because api_error_status 429 alone cannot separate credit exhaustion from a transient rate limit (Decision 3)
  - Retain literal-phrase matching as a fallback for agent versions without the fields
  - Requirements: 4.1, 4.2, 4.3

- [ ] 14. Stop discarding the agent's diagnostic text
  - processExecResult (internal/agents/claudecode/agent.go:303-309) returns the raw exec.ExitError, overriding the ClassifiedError built above it
  - classifyFromAgent (internal/orbit/single.go:71-73) synthesises "exit status 1" into errMsgs, which BuildUnknownMessage then prefers over stderr
  - Affects every classified failure that exits non-zero, not only credit exhaustion
  - Requirements: 4.4, 4.5

- [ ] 15. Classifier regression tests for all known message variants
  - The two historical wordings with and without reset time and timezone, plus the credits wording, plus the structured-field path
  - This table is the guard against a fourth recurrence
  - Requirements: 4.2, 4.6

- [ ] 16. Integration test: credit exhaustion mid-run
  - Mock agent emits the credit-exhaustion JSON; assert no retries, the new class, and a recorded message naming the cause
  - Requirements: 4.3, 4.4

## Phase 4: Run-Level Finalization

- [ ] 17. Define the run-level status aggregation rule
  - Every variant failed gives a failed run; at least one variant completed gives a successful run with per-variant detail; interrupted is recorded distinctly
  - A variant losing a comparison is a normal outcome, not a run failure
  - Requirements: 5.3

- [ ] 18. Finalize the run-level summary in variant mode
  - runWithVariants (internal/orbit/variants.go:176-308) never touches o.logManager; give it a terminal path calling Complete or Fail
  - Aggregate per-variant phases, cost and duration into the run-level totals rather than leaving zeros
  - Requirements: 5.1, 5.2

- [ ] 19. Update the run-level registry entry in variant mode
  - registerRun and updateRunStatus are single-run only; variant mode uses registerVariantRun and leaves the run-level entry absent
  - Fix alongside task 18 or the inconsistency merely moves
  - Requirements: 5.4

- [ ] 20. Record a terminal state on interrupt and cancellation
  - Signal or context cancellation must still close out the summary
  - Requirements: 5.5

- [ ] 21. Prevent run_number inflating past an unclosed summary
  - loadExistingSummary increments RunNumber and resets status to running (internal/logs/manager.go:236-239), so each invocation inflates the counter while never closing the previous run
  - Requirements: 5.6

- [ ] 22. Integration test: every terminal route closes the summary
  - Drive a variant run to success, all-variants-failed, blocked, and interrupt; assert a terminal status and completion timestamp in each
  - Requirements: 5.1, 5.3, 5.5

## Phase 5: Observability

- [ ] 23. Make repeated phase execution visible in the artifacts
  - Within one invocation RunNumber is frozen, so every iteration overwrote the same phase files; only the sessions array retained evidence
  - Requirements: 6.1

- [ ] 24. Warn when a phase is executed repeatedly within one invocation
  - Threshold well below the iteration ceiling, so a stall is visible while running rather than only afterwards
  - Requirements: 6.2

- [ ] 25. Document that the -run-N suffix counts invocations, not iterations
  - phaseFileName (internal/logs/manager.go:506-513) reads RunNumber, which increments once per orbit run invocation
  - This has already caused a misreading during diagnosis
  - Requirements: 6.3
