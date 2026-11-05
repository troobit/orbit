# Local comparison prerequisites

The four coding phases in `tasks.md` need Go, Git, rune and Python 3 for the
commit-helper tests. They do not need model inference or Sanarte. Planned
runbook/template paths in the smolspec are outputs of those phases.

## Before releasing the evaluation gate

The first task of `tasks-evaluation.md` is an explicit STOP gate. It is completed
only after the owner requests live evaluation and the following are checked:

- The coding ledger is complete and `make test-local`, `make test`, `make lint`
  and `make build` pass. Use that built Orbit binary and record its commit.
- Record installed Codex/OpenCode versions and rune repository commit. Check
  CLI flags and configuration against those installed versions; configuration
  templates alone do not establish provider support or effective permissions.
- LM Studio already serves the intended `8-bit` model. Record `lms ps`, loaded
  context, model identity and slots. Both harnesses must fit the current load;
  do not reload, reconfigure or interrupt someone else's inference.
- The installed next-task skill exists and both harnesses can read the absolute
  path. Inspect the fixture's instruction files and verify the actual rules
  loaded. Tools and helper-model requests must remain local as specified.
- Record availability and authorization for the hosted `claude-code` judge.
  This is required for evaluation task 2.4 only; the direct local execution
  smoke can proceed without it. If unavailable or unauthorized, leave 2.4
  blocked rather than use another judge silently or complete the comparison
  smoke stage.
- Identify a disposable parent directory for fixture repositories, writable
  caches, result artifacts and an uncontended window for timeout probes.
  Keep logs outside the set of files staged by the commit helper.

Time passing does not release this gate. Coding task completion does not imply
permission to start inference, edit Sanarte or change the server load.

## Before the Sanarte task

- The fixture's direct local execution, individual aliases, parallel comparison
  and uncontended timeout-cleanup checks all passed and are recorded in `runs.md`.
  If cleanup fails with the background OpenCode service, try the installed
  `--standalone` mode and recheck. If still failing, stop and scope a cancellation
  repair; do not leave a long inference running behind a timed-out Orbit session.
- Confirm permission to edit `~/repos/sanarte/.worktrees/wt-**`. The worktrees are there to be used. Use them liberally. The main checkout, `simplify-for-square` branch remain intact.
- Prepare offline dependencies and writable worktree caches. Run the baseline
  Makefile checks without secrets. A failing baseline is a blocker, not a worker
  task to repair incidentally.
- Review the narrowed code-only ledger against the Sanarte smolspec. Original
  outcomes and stable IDs stay with the owner acceptance; new implementation
  tasks receive new IDs. Preserve completed history and verify all local blockers
  resolve. No automated production bookings, credentials, copied `.env`, browser
  acceptance or unsettled widget-policy decision belongs in the dispatch ledger.
- Record the new staging commit, exact base and settings before variant setup.
  Use an explicit ledger, guidance and helper base; disable pre/post prompts and
  auto-consolidation for the measured run. Preserve artifacts before cleanup.

## Evidence and completion

`runs.md` does not exist yet because no evaluation is claimed. Create it during
evaluation using the record fields in the smolspec. A failed Sanarte experiment
is a valid recorded outcome; a failed smoke prerequisite never unlocks Sanarte.
Task completion in this evaluation ledger records the procedure, not acceptance
of the original Sanarte feature or proof of autonomous SDD.
