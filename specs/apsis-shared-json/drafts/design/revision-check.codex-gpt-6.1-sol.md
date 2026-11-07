# Revision Check: Shared JSON Design

Fresh-context non-author verification of the completed **18:36Z revision** against both completed 18:26Z reviews, approved requirements, current decisions/questions and relevant parser/lister/resolver code. This checks contracts and planned verification, not implemented behavior. This is same-model fresh-context evidence, **not independent-model evidence or human approval**. U1–U3 and accepted human policies were preserved.

The requested fixes are explicit, with one remaining scope contradiction affecting disposition 2:

1. **Detection resolved.** [Design §2](../../design.md#2-codex-exec-stream-parser) and [Decision 21](../../decision_log.md#decision-21-complete-the-exec-probe-before-trusting-heuristic-kiro-recognition) continue the complete-line probe past heuristic-only Kiro recognition, recombine split lines and replay consumed bytes. All three outcomes are mapped: only `exec` promotes the format; the others preserve the legacy result. Planned tests cover long malformed prefixes, long split first objects and existing Kiro recognition.

2. **Selected-winner suppression resolved; listing scope needs reconciliation.** [Design §5](../../design.md#5-exec-discovery-identity-project-and-suppression) and [Decision 22](../../decision_log.md#decision-22-suppress-only-the-verified-rollout-candidate-the-resolver-selects) validate the actual first filename-match winner before indexing it. A later valid duplicate cannot rescue an invalid winner; explicit stream opening remains available. Planned same-winner tests cover both scopes. See the residual below.

3. **Windows contract resolved, capability unverified.** [Design §4](../../design.md#4-listing-scope-and-cache-sessions) and [Decision 23](../../decision_log.md#decision-23-gate-cache-admission-on-platform-identity-and-change-fingerprints) require file/volume identity and change metadata, native replacement tests before cache admission, and an explicit failure outcome without a weak size/mtime fallback. This review does not establish runtime support.

4. **First-known summaries resolved.** [Design §§1/3](../../design.md#1-shared-representation-additions-transcript) and [Decision 24](../../decision_log.md#decision-24-preserve-first-known-parsed-summaries-without-requiring-assistant-content) retain `FirstModel`/`FirstProvider` independently of assistant entries, preserve chronological first-known values and later active entry values, and cover header-only/user-only file, stdin and identifier conversion.

5. **Scan compatibility resolved.** [Design §4](../../design.md#4-listing-scope-and-cache-sessions)/Q57 preserve existing Claude timestamp scan/error semantics and Codex metadata scan limits; only new optional Claude cwd extraction has the total cap. These instructions agree with [ParseFirstTimestamp](../../../../internal/transcript/parser.go) and the [existing Codex scanners](../../../../internal/sessions/lister.go), including legacy time fallback behavior.

6. **Invalid item IDs resolved.** [Design §2](../../design.md#2-codex-exec-stream-parser)/Q55 route missing, empty or invalid IDs to individually keyed generic events with `item_decode_failed`, creating no tool links. Verification includes two missing-ID tool items across renderers.

7. **Ordering/options resolved.** [Design §4](../../design.md#4-listing-scope-and-cache-sessions)/Q60–Q61 restrict extended ordering to feature-enabled modes, preserve no-flag ties, and configure both constructors through shared `Option func(*Options)`.

No critical findings. **One major residual — global candidate selection conflicts with protected project listings** ([5.1](../../requirements.md#5.1)). Q49 still says project-scope listings keep current behavior, but Decision 22 and design §5 apply global winner selection to listing before project filtering without a mode boundary. Current [listCodex](../../../../internal/sessions/lister.go) filters each file independently: valid duplicates A and B with the same filename UUID but different cwd values let project B list B, even when [findCodexSession](../../../../internal/sessions/resolver.go) selects A globally. Applying the revised global selection unconditionally removes B from a no-flag project listing, contradicting Q49/5.1. This does not challenge human-approved global stream suppression.

**Fix:** scope winner-based row selection/deduplication to all-projects listings, retain per-file enumeration/cwd filtering for project listings under Q49, and always validate the global resolver winner for suppression. Reconcile Decision 22, design §§4–5 and the duplicate tests with that boundary.

Before implementation admission: which exact listing-mode condition applies winner filtering, and does a no-flag cross-project duplicate fixture retain its legacy row? Native Windows verification also remains a stated implementation gate.
