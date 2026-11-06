# Cross-review: Claude tasks draft

Reviewer: `codex-gpt-6.1-sol`, non-author of the [Claude draft](draft.claude-code-opus-5.5.md). The [Codex draft](draft.codex-gpt-6.1-sol.md) is comparison evidence only. This review uses [approved requirements](../../requirements.md), [current design](../../design.md), decisions through Q62, and repository code. It is independent of the Claude author by harness/model; no human approval is implied.

## Major

1. **Task 15.2 / TD3: module files violate disjoint ownership.** Task 15 has no blockers and promotes x/sys in go.mod, while 3.2 adds the validator to go.mod/go.sum. Their streams can claim both concurrently. Keep Claude's separate platform stream and cache-only Windows gate, but serialize module mutations: block fingerprint implementation on the schema implementation, or reserve module changes to one owner and record the handoff. Adapter-only red work can remain independent. The Codex draft avoids overlap by serializing everything, but unnecessarily blocks unrelated work on Windows.

2. **Task 3.2 / TD2: staged block enums need a compatibility disposition.** A schema restricted to existing block kinds rejects later additive plan/error/event/usage kinds even at schema_version 1. Q13 requires consumers to ignore unknown fields, but does not explicitly promise unknown discriminator values; TD2 acknowledges rejection without a concrete consumer rule. Keep staged producers, not an assumption that a mutable upstream `$id` updates a cached validator. Either publish the complete known v1 block union in the first slice, or document that validation requires the producer's schema revision and add an old-schema/new-block test. Resolve this technical contract in the design/decision log; never call a breaking field-meaning change additive.

## Minor

3. **Tasks 15–16: expose the external prerequisite.** Keep Claude's isolated gate; Codex's predecessor chain incorrectly blocks all-projects and exec work. Add prerequisites.md naming the native Windows command, required replacement/in-place cases and evidence to record. No native result exists; cross-compilation is insufficient. Cache task completion remains blocked without it, while other slices proceed. Do not change CI or weaken 5.9 to manufacture a pass.

4. **Task 13.1: deterministic identifier-conflict coverage needs a seam.** Keep the Codex draft's injected digest-conflict test: production SHA-256 collisions cannot be fixture-generated. Inject only the digest helper in tests and assert that both paths are omitted with identifier_conflict, independent of root order ([2.8](../../requirements.md#2.8)).

5. **Tasks 13.1/16.1: make performance fallback decisions reproducible.** Keep synthetic benchmarks and source-specific fallback choices from both drafts. Record fixture dimensions, platform, cold/warm distinction and measurements. A universal 1.5-second failure threshold is unsupported by evidence and host-sensitive; treat it as a local investigation budget, not an acceptance test. Zero unchanged transcript-body reads remains a hard assertion ([5.9](../../requirements.md#5.9)).

6. **Draft links and adoption metadata need reconciliation.** Canonical-relative requirement links currently fail from drafts/tasks; preserve the draft and fix links in the adopted plan. Preserve all canonical stable IDs, statuses and owners, especially task 1. Snapshot tasks.md first. Replace stale freeze attribution in handoff and design questions using the checked 19:00Z note; no human freeze exists. Add the small missing baseline requirement reference and maintain detailed red/green pairs, source-selector constraints and per-slice README/CHANGELOG updates.

## Keep and remaining choices

Keep Claude's PR boundaries, narrow work units, provider-first metadata, Kiro matched-key and injectable all-project discovery fixes, complete-line detection and selected-rollout/Q62 regressions. Keep Codex's planned fingerprint placement unless the separate-package proposal is explicitly admitted in design, and its collision-injection coverage. Both plans cover all 31 criteria; the Claude draft's 46 stable IDs and rune JSON parse were checked. Source checks confirm project-scope GetSession currently returns only a reader, all-projects listKiro bypasses its injection, and CI runs Ubuntu while releases cross-compile Windows.

No new material user behaviour choice is necessary for merging the plan. U1/U2 stay within approved scope by retaining current Claude-model/Kiro-provider behaviour; U3 remains a reversible local dependency proposal. Native Windows execution is an external verification dependency, not an answered policy question. Headless work must not mark design/tasks approved or cache verified.
