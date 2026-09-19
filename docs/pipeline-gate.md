# Using shallnot as a deterministic gate

This is the guide for wiring `shallnot check` into an automated agent
pipeline that ships code with no human review: a coding agent implements a
ticket, a test runner produces JUnit XML, and an orchestrator decides,
without a person in the loop, whether the result is allowed to proceed.

When `test_commands` is configured in `shallnot.yaml` (or passed as
`--test-command`), `shallnot gate` is the single entry point: it runs the
test commands, then checks the results they produced, in one invocation.
`shallnot check` remains the right command for a pipeline whose test runner
is a separate job or step from the traceability check — it only reads
results files that already exist. This guide otherwise applies equally to
both: substitute `gate` for `check` wherever this repository's own test
runner has not already produced the results.

See [docs/report.md](report.md) for the JSON report this guide reads,
[docs/configuration.md](configuration.md) for every flag and config key used
below, and [docs/agents.md](agents.md) for equipping the implementing agent
itself — the skill it reads and the end-of-turn hook that enforces the gate
before its turn ends.

## Contract

`shallnot check` is deterministic, offline and model-free: for the same
spec files, source tree and JUnit XML, it produces a byte-identical report
(see [docs/report.md](report.md#determinism)). It performs no network calls
and calls no model. A `pass` verdict proves that every requirement in focus
has a passing test bound to it at its current revision, and that no defect
(orphan tag, revision mismatch, malformed tag, unjustified non-testable
requirement, duplicate ID, malformed spec declaration) was found. It does
**not** prove that the test asserts the right thing, that the requirement's
statement is itself correct, or that no requirement was left out of the
spec. A tag is necessary evidence that a requirement is verified, not
sufficient evidence: the verdict is a structural precondition an orchestrator
can check mechanically, and the rigour of the tests remains a separate
concern.

## Per-ticket plan documents as specs

A plan document for one ticket is a spec file whose requirement IDs carry
the ticket's ID as a prefix, for example `ABC-101.AC1`, `ABC-101.AC2`,
`ABC-101.AC3` for ticket `ABC-101` (see
[fixtures/pipeline/plans/ABC-101.md](../fixtures/pipeline/plans/ABC-101.md)).
Configure `id_pattern` (or `--id-pattern`) to the shape used across the
repository's tickets, for example:

```yaml
id_pattern: '[A-Z]+-[0-9]+\.AC[0-9]+'
```

This is a whole-ID RE2 match: it must accept `ABC-101.AC3` and reject
anything containing `~`, `,`, `]` or whitespace. See
[docs/configuration.md](configuration.md#the-configuration-file).

## Known specs vs focus

`--specs` (or config `specs`) is the set of **known** specs: every plan ever
written, so that a tag citing an earlier ticket's requirement is a resolvable
citation, not an orphan. `--focus` (or config `focus`) is the plan just
implemented: the run demands coverage only for the requirements it declares.
A citation to a known-but-not-focused requirement is fine; a citation to an
unknown requirement is an `orphan_tag` finding regardless of focus (see
[docs/report.md](report.md#finding-categories)).

Run from `fixtures/pipeline`, with `plans` as the known specs and only
`ABC-101.md` in focus:

```
$ shallnot check --specs plans --focus plans/ABC-101.md \
    --tests service/tests --tests client/test \
    --results service/results --results 'client/results/*.xml'
shallnot: PASS
focus: plans/ABC-101.md
requirements: 5 known, 3 in focus (2 covered, 0 failed, 0 skipped, 0 not run, 0 uncovered, 1 non-testable)
tests: 3 in results, 3 bound, 0 untagged
findings: 0 error, 0 warning, 0 info (0 blocking)

REQUIREMENTS
  ABC-101.AC1~1  covered  plans/ABC-101.md:10
      passed  tests.test_invoices › test_export_is_named_after_the_invoice_number  service/tests/test_invoices.py:12
  ABC-101.AC2~1  covered  plans/ABC-101.md:12
      passed  test/export.test.ts › invoice page > starts the download in place when Export is clicked [verifies ABC-101.AC2~1]  client/test/export.test.ts:5
  ABC-101.AC3~1  non_testable  plans/ABC-101.md:14
```

Five requirements are known (`ABC-100.AC1`, `ABC-100.AC2` from an earlier
ticket, plus the three `ABC-101` ones); only the three `ABC-101` requirements
are in focus, and the earlier ticket's tag on `ABC-100.AC1` resolves cleanly
against the known spec without being demanded here.

If the earlier ticket's plan is left out of the known specs, its tag becomes
an orphan instead of being silently ignored:

```
$ shallnot check --no-config --id-pattern '[A-Z]+-[0-9]+\.AC[0-9]+' \
    --focus plans/ABC-101.md --tests service/tests --tests client/test \
    --results service/results --results 'client/results/*.xml'
shallnot: FAIL
focus: plans/ABC-101.md
requirements: 3 known, 3 in focus (2 covered, 0 failed, 0 skipped, 0 not run, 0 uncovered, 1 non-testable)
tests: 3 in results, 3 bound, 0 untagged
findings: 1 error, 0 warning, 0 info (1 blocking)
...
FINDINGS
  error  orphan_tag  service/tests/test_invoices.py:6  tag cites ABC-100.AC1~1, but no known spec declares ABC-100.AC1
```

Always pass the full set of plans as `--specs`, even when only the newest one
is `--focus`.

## Advisory mode as the adoption path

Advisory mode (`--advisory` / `advisory: true`) reports every finding exactly
as a normal run does, but forces `exit_code` to 0 regardless of `verdict`.
Use it while rolling shallnot out: wire the gate into the pipeline, collect
its reports, and confirm the findings it raises are the ones you want to act
on, without yet blocking the pipeline. An orchestrator in advisory mode must
still read `run.advisory` and `verdict` from the JSON report — not just the
exit code — to log or surface findings: `exit_code` alone is indistinguishable
from a genuine pass while advisory is on.

```
$ shallnot check --advisory
shallnot: FAIL (advisory mode: the exit code is 0 whatever the verdict)
...
$ echo $?
0
```

```
$ jq '{verdict, advisory: .run.advisory, exit_code}' report.json
{
  "verdict": "fail",
  "advisory": true,
  "exit_code": 0
}
```

Remove `--advisory` once the pipeline is ready to block on the verdict.

## Several repositories

`--tests` and `--results` (and their config equivalents) each accept more
than one entry, anywhere on disk — the ticket's plan can span a client
repository and a service repository checked out side by side. The pipeline
fixture demonstrates this with two roots and two results sets:

```yaml
tests:
  - service/tests
  - client/test
results:
  - service/results
  - client/results/*.xml
```

The report's `run.test_roots` and `run.results_files` list every root/file
actually read, and a requirement's `tests[].results_file` /
`tests[].source.file` show which repository each bound test came from (see
[docs/report.md](report.md#requirement-object)).

## Test runners must have already run

`shallnot check` reads JUnit XML; it does not run tests itself. The
orchestrator's pipeline must run every relevant test suite and produce JUnit
XML **before** invoking the gate. A `--results` pattern (file, directory or
glob) that matches no file is a tool failure, not a verdict:

```
$ shallnot check --specs plans --tests service/tests --tests client/test \
    --results does-not-exist.xml
shallnot: error: results "does-not-exist.xml" matches no file
$ echo $?
2
```

An orchestrator must not interpret this exit code as "tests failed" — it
means the gate could not even be evaluated, most often because a test runner
step did not run or wrote its output to a different path than the gate
expects.

## Exit codes

| Code | Orchestrator action |
|---|---|
| `0` | Proceed: no blocking finding (or the run was advisory — check `run.advisory`/`verdict` if advisory reporting still matters downstream). |
| `1` | Do not proceed. Read `findings` where `blocking` is true from the JSON report and route each one to the implementing agent per the table below. |
| `2` | Infrastructure problem, not a verdict. Do not feed this to the coding agent as if it were a failing test; fix the pipeline (missing config, bad flags, unreadable or unmatched input, invalid spec document) and re-run the gate. |

### Routing table: finding category to owner and action

Findings about tests go to the implementing agent. Findings about the spec go
to whoever owns the spec (the planning agent, or a person): the implementing
agent does not edit a requirement to satisfy the gate. Findings about inputs
go to the pipeline.

| Category | Owner | Action |
|---|---|---|
| `uncovered_requirement` | Implementing agent | Write a test whose assertions check the requirement's statement, and tag it. |
| `failed_requirement` | Implementing agent | Fix the implementation until a bound test passes. Fix the test only where it contradicts the requirement's statement. |
| `failing_bound_test` | Implementing agent | Same as `failed_requirement`, for the test named in the finding. |
| `skipped_requirement` | Implementing agent | Un-skip the bound test and make it pass. |
| `not_run_requirement`, `tag_not_in_results` | Implementing agent, then pipeline | Make the runner collect the test, and place the tag where the runner reports it (see [docs/binding.md](binding.md)). If the test ran in another job or repository, the pipeline passes that results file too. |
| `orphan_tag` | Implementing agent, then pipeline | Fix a mistyped ID. If the ID belongs to a spec the run was not given, the pipeline adds that spec to the known specs. If no spec declares it, the claim is removed and reported: the test verifies nothing stated. |
| `revision_mismatch` | Implementing agent | Re-read the requirement's current statement, update the test so that it verifies that statement, then cite the current revision. Never a blind text substitution. |
| `malformed_tag` | Implementing agent | Write the tag as `[verifies ID~REVISION]`. |
| `bound_non_testable` | Spec owner | Decide: the requirement is testable after all (make it `active`), or the tag is wrong. |
| `unjustified_non_testable` | Spec owner | Write the justification, or make the requirement `active`. |
| `duplicate_id` | Spec owner | Give one of the requirements another ID. |
| `malformed_requirement` | Spec owner | Fix the declaration (revision, statement, YAML keys). |
| `untagged_test` (blocking under `strict`) | Implementing agent | Tag the test with the requirement it verifies. A test that verifies no stated requirement is a sign of work outside the spec: report it rather than inventing a citation. |

**Prohibitions, whatever the category.** Each of these turns the report green
without making the software correct, and destroys the evidence the verdict
stands for:

- Never delete or alter a tag to make a finding disappear. Removing the claim
  does not verify the requirement; it hides that nothing does.
- Never mark a requirement `non_testable` to avoid writing a test. The status
  belongs to the spec's owner and removes the requirement from verification
  for good.
- Never weaken, remove or bypass an assertion, skip or delete a failing bound
  test, or edit a results file, to change the verdict.
- Never tag a test with a requirement its assertions do not check.

The agent skill in [plugin/skills/shallnot/SKILL.md](../plugin/skills/shallnot/SKILL.md)
states the same rules for the implementing agent. When a requirement cannot
be met or tested as written, the correct outcome is a blocked run with the
reason reported, not a green one.

## Stale-report protection

`shallnot check` deletes any file at `--json-out`/`--markdown-out` before it
does anything else, and only recreates it if the run completes far enough to
produce a report. A tool failure (exit 2) therefore leaves no report file
at all — an orchestrator that reads a report file must first confirm the
gate's exit code was 0 or 1; reading a report path left over from a previous
step (or from a run that never happened) is not possible, because the file
will simply not exist after a failed run.

## Worked orchestrator example

```sh
#!/bin/sh
set -eu

BIN="$1"
shift

report="$(mktemp)"
set +e
"$BIN" check --json-out "$report" "$@"
code=$?
set -e

case "$code" in
  0)
    echo "GATE: proceed"
    ;;
  1)
    echo "GATE: blocked, blocking findings:"
    jq -c '.findings[] | select(.blocking) | {category, location, message}' "$report"
    ;;
  2)
    echo "GATE: infrastructure failure (exit 2); not a verdict, do not route to the coding agent as a test failure" >&2
    ;;
esac
rm -f "$report"
exit "$code"
```

Passing, focused run (the ticket just implemented, `ABC-101`), from
`fixtures/pipeline`:

```
$ ./gate.sh "$(pwd)/../../bin/shallnot" --focus plans/ABC-101.md
...
GATE: proceed
exit:0
```

Failing, unfocused run (every known requirement in focus, exposing the
uncovered `ABC-100.AC2`):

```
$ ./gate.sh "$(pwd)/../../bin/shallnot"
...
FINDINGS
  error  uncovered_requirement  plans/ABC-100.md:7  requirement ABC-100.AC2~1 has no bound test
GATE: blocked, blocking findings:
{"category":"uncovered_requirement","location":{"file":"plans/ABC-100.md","line":7},"message":"requirement ABC-100.AC2~1 has no bound test"}
exit:1
```

## Revision bumps when a plan changes

When a requirement's statement changes in a way that changes what a test
must prove, bump its revision in the spec (`ABC-101.AC1~1` becomes
`ABC-101.AC1~2`). Every tag still citing the old revision stops resolving:
the requirement itself becomes uncovered (no test cites the new revision
yet), and each stale tag raises `revision_mismatch` at its own location,
naming both the cited and the current reference:

```
$ shallnot check --focus plans/ABC-101.md
shallnot: FAIL
focus: plans/ABC-101.md
requirements: 5 known, 3 in focus (1 covered, 0 failed, 0 skipped, 0 not run, 1 uncovered, 1 non-testable)
tests: 3 in results, 3 bound, 0 untagged
findings: 2 error, 0 warning, 0 info (2 blocking)

REQUIREMENTS
  ABC-101.AC1~2  uncovered  plans/ABC-101.md:10
  ABC-101.AC2~1  covered  plans/ABC-101.md:12
      passed  test/export.test.ts › invoice page > starts the download in place when Export is clicked [verifies ABC-101.AC2~1]  client/test/export.test.ts:5
  ABC-101.AC3~1  non_testable  plans/ABC-101.md:14

FINDINGS
  error  uncovered_requirement  plans/ABC-101.md:10                requirement ABC-101.AC1~2 has no bound test
  error  revision_mismatch      service/tests/test_invoices.py:12  tag cites ABC-101.AC1~1, but the spec declares ABC-101.AC1~2: re-verify the test against the current statement, then cite ABC-101.AC1~2
```

This is the mechanism by which a spec edit forces re-verification: the
implementing agent must re-check that the test proves the new statement
before updating the tag to `~2`, not merely bump the number in the tag.

## GitHub Actions

The repository's composite action ([action.yml](../action.yml)) installs a
pinned release of `shallnot` ([action/install.sh](../action/install.sh)) and
runs it ([action/check.sh](../action/check.sh)):

- **Inputs**: `version` (release tag, default `latest`), `args` (extra
  arguments for `shallnot check`, for example
  `--focus plans/ABC-101.md --advisory`), `working-directory` (default `.`,
  the directory holding `shallnot.yaml`), `json-report` (path, relative to
  `working-directory`, for the JSON report; default `shallnot-report.json`).
- **Outputs**: `exit-code` (0, 1 or 2) and `json-report` (the report's path;
  absent after a tool failure, per the stale-report protection above).
- `action/check.sh` runs `shallnot check --github-annotations --markdown-out
  <tmp> --json-out <json-report> $args`, appends the Markdown summary to
  `GITHUB_STEP_SUMMARY` when present, sets the `exit-code` and `json-report`
  step outputs, and exits with `shallnot`'s own exit code — so a workflow
  step naturally fails the job on exit 1 or 2 unless it explicitly checks
  `steps.<id>.outputs.exit-code` to branch (for instance, to still post a
  comment on advisory-mode failures without failing the job).
- `--github-annotations` makes GitHub render each finding as an inline
  annotation on its file and line, in addition to the job summary; see
  [docs/report.md](report.md#github-annotations---format-github-or---github-annotations).
