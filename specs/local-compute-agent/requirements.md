# Requirements: Local Compute Agent

## Introduction

Orbit currently drives five hosted coding agents, all of which send repository content to a vendor API. This feature makes a locally-served model a first-class orbit target, so implementation phases and variant runs can execute entirely on the developer's own hardware. The model is served by baseRT and driven by the existing OpenCode agent loop; no new agent implementation is created.

The requirements below concern the plumbing that connects orbit to a local runtime. The quality of the local model's output is explicitly not in scope — see Non-Goals.

## Non-Goals

- **Model output quality.** Whether a locally-served model writes good code, or drives an agent loop reliably across many turns, is a property of the model and improves independently of this feature. No acceptance criterion gates on it; the Verification section is where real-world behaviour is observed.
- Rendering local-run transcripts in apsis or the `orbit serve` transcript views — OpenCode sessions are not readable by `internal/sessions/` or `internal/transcript/` today, and this feature inherits that pre-existing gap rather than closing it.
- Managing the lifecycle of the local model server — starting, stopping, supervising, or health-checking it before a run.
- A native agent package implementing `agents.Agent` directly against baseRT.
- Changing which agent `orbit compare` and `orbit consolidate` use. Requirement 7 documents the existing split rather than altering it.
- How local runs are rendered in the `orbit serve` dashboard, `orbit status`, or the run registry.
- Session resume semantics for OpenCode, including the existing `--continue` fallback behaviour.
- Serving the model from a second machine over the network.
- Changing timeout, retry, or error-classification behaviour for hosted agents, including hosted OpenCode runs.

**Shared-code changes this feature does require.** Requirements 4, 5 and 6 touch code paths common to all agents. Their effect on hosted agents is bounded by acceptance criteria [4.4](#4.4), [5.1](#5.1) and [6.4](#6.4) rather than avoided, and this is a deliberate departure from the Non-Goal above.

## Requirements

### 1. Local Compute as a Run Target

**User Story:** As a developer, I want to select a locally-served model when starting a run, so that my implementation phases never leave my machine.

**Acceptance Criteria:**

1. <a name="1.1"></a>The system SHALL provide an agent type identifier for locally-served models, distinct from the existing `opencode` type and nameable in an alias's `type` field  
2. <a name="1.2"></a>The system SHALL accept an alias of the local agent type via `--agent` and execute phases against it  
3. <a name="1.3"></a>The system SHALL accept an alias of the local agent type in `--variant-agents` and assign it to variants on the same cycling rules as any other alias  
4. <a name="1.4"></a>WHEN a phase runs under an alias of the local agent type, the system SHALL direct model requests to the locally configured endpoint  
5. <a name="1.5"></a>IF a value passed to `--variant-agents` names an unknown or uninstalled agent THEN the system SHALL report it before creating any worktree  

### 2. Bounded Execution for Local Runs

**User Story:** As a developer, I want a local phase given hours rather than minutes before it is abandoned, so that slow inference finishes while a genuinely stuck run still terminates on its own.

**Acceptance Criteria:**

1. <a name="2.1"></a>The system SHALL apply a default execution timeout of 2 hours to a phase run under an alias of the local agent type  
2. <a name="2.2"></a>WHERE a timeout is configured on the alias, the system SHALL apply that value in place of the default  
3. <a name="2.3"></a>IF a local phase exceeds its effective timeout THEN the system SHALL terminate it and report both the elapsed duration and the limit that was exceeded  
4. <a name="2.4"></a>The system SHALL NOT retry a phase that was terminated for exceeding its timeout  
5. <a name="2.5"></a>The system SHALL apply the default timeout only to the local agent type, leaving every other agent's effective timeout determined solely by its alias configuration as it is today  
6. <a name="2.6"></a>WHEN comparison runs under an alias of the local agent type, the system SHALL apply that alias's effective timeout in place of the comparison default  

### 3. Failure Diagnostics for the Local Runtime

**User Story:** As a developer, I want an immediate error when my local setup is wrong, so that I am not waiting two hours for a run that could never succeed.

**Acceptance Criteria:**

1. <a name="3.1"></a>IF the local endpoint refuses the connection THEN the system SHALL stop the run and report that the local runtime is unreachable  
2. <a name="3.2"></a>IF the local runtime reports that the requested model is unavailable THEN the system SHALL stop the run and report the requested model name  
3. <a name="3.3"></a>The system SHALL classify unreachable-endpoint and unavailable-model conditions as fatal for the local agent type in both single-run and variant mode, and SHALL NOT retry them  
4. <a name="3.4"></a>The system SHALL leave error classification for hosted OpenCode runs unchanged  
5. <a name="3.5"></a>WHEN a local run fails, the system SHALL make the runtime's own error output available in the reported failure alongside any classified message  

### 4. Guarding Against False Completion

**User Story:** As a developer, I want orbit to stop when phases report success without advancing any task, so that a misbehaving setup cannot loop for hours at a time.

**Acceptance Criteria:**

1. <a name="4.1"></a>The system SHALL treat a phase as having made no progress when it completes without error and the set of pending task identifiers is identical before and after the phase  
2. <a name="4.2"></a>IF two consecutive phases make no progress THEN the system SHALL stop the run and report the phase number and the consecutive count  
3. <a name="4.3"></a>The system SHALL apply no-progress detection in both single-run and variant mode  
4. <a name="4.4"></a>The system SHALL apply no-progress detection only to runs using an alias of the local agent type and the default task-advancing command, leaving runs with a custom command unaffected  

### 5. Cost Reporting Without a Misleading Zero

**User Story:** As a developer comparing variants, I want a run with no cost data to show no figure, so that free and failed runs are not confused.

**Acceptance Criteria:**

1. <a name="5.1"></a>WHERE an agent reports no cost data, the system SHALL omit the monetary figure rather than displaying zero, in both the comparison report and variant progress output  
2. <a name="5.2"></a>WHERE the local runtime reports token usage, the system SHALL record and display input and output token counts  
3. <a name="5.3"></a>WHERE the local runtime reports no token usage, the system SHALL omit token counts rather than displaying zero  

### 6. Visibility Into a Running Local Phase

**User Story:** As a developer watching a phase that may run for hours, I want evidence that it is still working, so that I can tell progress from a hang without waiting for the timeout.

**Acceptance Criteria:**

1. <a name="6.1"></a>WHILE a phase is executing, the system SHALL write whatever output the agent has produced to a file readable before the phase completes  
2. <a name="6.2"></a>IF a run is interrupted before a phase completes THEN the system SHALL retain the output captured up to that point  
3. <a name="6.3"></a>WHILE a phase is executing, the system SHALL report elapsed time at intervals in both single-run and variant mode, independently of whether the agent has produced output  
4. <a name="6.4"></a>The system SHALL apply output capture to all agents without changing what any agent reports on completion  
5. <a name="6.5"></a>WHEN a run starts, the system SHALL report the resolved alias, agent type and model for each agent it will use, without requiring verbose mode  

> Acceptance criterion [6.1](#6.1) depends on the agent CLI writing output progressively rather than emitting a single document at exit. Spike question 5 (decision log, Decision 8) settles this, and carries a downgrade branch: if output does not stream, [6.1](#6.1) reduces to capture-at-exit and [6.3](#6.3) becomes the sole liveness signal.

### 7. Configuration and Setup Documentation

**User Story:** As a developer setting this up for the first time, I want the configuration and its limits documented, so that I can reach a working local run and know what it does not cover.

**Acceptance Criteria:**

1. <a name="7.1"></a>The repository SHALL document a working `.orbit.yaml` alias definition for the local agent type  
2. <a name="7.2"></a>The repository SHALL document the provider configuration that points the agent loop at the local endpoint  
3. <a name="7.3"></a>The repository SHALL document the steps to install the local runtime and pull a model before first use  
4. <a name="7.4"></a>`CLAUDE.md` SHALL list local compute as a supported run target  
5. <a name="7.5"></a>The documentation SHALL state that local-run transcripts are not viewable in apsis  
6. <a name="7.6"></a>The documentation SHALL state that comparison during `orbit run` uses the run's own agent, while the standalone `orbit compare` and `orbit consolidate` commands use the configured default agent, which may be a hosted provider  

## Verification

Completion gates for the feature, confirmed by a person rather than by an automated test.

1. <a name="V.1"></a>At least one full orbit phase has executed end-to-end against a locally-served model, producing tracked file changes and at least one commit  
2. <a name="V.2"></a>The verification run used a public model pulled to the local machine and sized to run within 64 GB of unified memory  
3. <a name="V.3"></a>The elapsed duration of the verification run is not grounds for failing it, provided it completed within its effective timeout  
4. <a name="V.4"></a>The model used, the observed duration, and the spike answers are recorded in the feature's documentation  
