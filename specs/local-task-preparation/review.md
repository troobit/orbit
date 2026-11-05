# Preparation review

## Explanation validation

- Beginner: a planner turns a large approved job into small job cards. Each
  card says what to change and how the host will check it. Failed work stays
  on disk and the next worker receives the relevant failure instead of starting blind.
- Intermediate: structured plan validation creates a separate rune ledger,
  packets and source-outcome map. Approval pins definitions; fresh workers edit
  declared files; the host executes checks and alone advances statuses. Runtime
  journals reconcile interrupted attempts before selection.
- Expert: portable content hashes distinguish source scope from mutable execution
  state. Status transitions require matching host evidence; per-worktree locks
  and per-variant runtime instances prevent cooperating runners from sharing
  mutable task state. Structural coverage cannot establish semantic fidelity.

## Independent review resolutions

Two Codex reviewers applied the design-critic and peer-validation roles. No
external AI service or live model was used for those reviews.

| Finding | Resolution |
| --- | --- |
| A model can mark work complete before host validation | Prepared prompts omit rune mutations; only host checks permit completion |
| Status hashes alone permit stopped-run `[x]` tampering | Pin initial statuses and reconcile all progress with accepted/recoverable journals |
| Nonzero exit can be an environment failure | Separate RED checkpoints require named failures, assertion evidence, baseline and a pinned structured verifier |
| Coverage keys can hide removed requirements | Host assigns immutable source obligations; review sees verbatim outcomes and deferrals |
| Deferring acceptance can accidentally release prerequisites | Deferred records have blocking edges; incomplete source dependencies remain enforced |
| Pre-existing dirty files hide further writes | Compare contents against each attempt's starting state, including untracked files |
| Alias flags can defeat planning's read-only policy | Initial planner is Codex with an argument allowlist and enforced sandbox; reject other configurations |
| Parallel variants could share local iteration state | One runtime per worktree/client/log manager; no snapshots on parent Orbit |
| Missing logs could lose host acceptance evidence | Required journal/output failures prevent inference/completion |
| Runtime artifacts dirty comparable branches | Candidate `.gitignore` and `.orbit/` attempts; commit definitions explicitly |
| Existing flags allow phase/model selection | Reject those options for prepared ledgers before side effects |
| Preserved correct work can produce a no-op | Host acceptance may finish retained work; no diff alone is not failure |

## Validation boundary

The coding ledger has six sequential phases and adjacent RED/GREEN children.
Only one stream is justified because each phase depends on the preceding
contract. Live evaluation is outside that ledger. Requirements and task links
are machine checked; the scoped mechanical audit has no findings. Baseline
`make test-local` and `make lint` pass. No preparation implementation is claimed.
