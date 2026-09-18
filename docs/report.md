# JSON report reference

The JSON report is the machine-readable output of `shallnot check --format json`
(and of `--json-out <file>`). It is a public API: a consumer may parse it and
act on it without running `shallnot` itself. The canonical schema is
[schemas/report.schema.json](../schemas/report.schema.json), also printed by
`shallnot schema report`. This document mirrors that schema and explains the
meaning of every enum value.

For the configuration that produces a report, see
[docs/configuration.md](configuration.md). For using the report as a gate in
an automated pipeline, see [docs/pipeline-gate.md](pipeline-gate.md).

## Compatibility

- `schema_version` is `"MAJOR.MINOR"` (for example `"1.0"`). A minor bump adds
  properties only; a major bump may remove or change one.
- Consumers must ignore properties they do not know.
- `extensions` objects (at the top level and on each requirement) are reserved
  for data contributed by analyses outside the deterministic core (code
  coverage, mutation score, judge answers). They are empty under
  `schema_version` `"1.0"`; a consumer must tolerate unknown keys inside them
  and must not treat their absence of content as a finding.

## Determinism

For identical inputs, the report is byte-identical across runs: no
timestamps, no absolute paths (`location.file` is relative to the working
directory, with forward slashes), and stable ordering everywhere (see
"Ordering" below). The same guarantee holds for the terminal, Markdown and
GitHub-annotation outputs.

## Top-level object

| Property | Type | Meaning |
|---|---|---|
| `schema_version` | string | Format version, `"1.0"`. |
| `tool.name` | string | Always `"shallnot"`. |
| `tool.version` | string | Version of the binary that produced the report. |
| `run` | object | What the run was asked to do and what it read. See below. |
| `verdict` | `"pass"` \| `"fail"` | `"fail"` when at least one finding is blocking. Independent of advisory mode: see "verdict, exit_code and advisory" below. |
| `exit_code` | `0` \| `1` | Exit code of the process that wrote this report. A tool failure (exit 2) writes no report — any file at `--json-out`/`--markdown-out` is deleted before the run starts, so a consumer never reads a report left over from an earlier, unrelated run. |
| `summary` | object | Aggregate counts. See below. |
| `requirements` | array of [requirement](#requirement-object) | The coverage matrix: every known requirement, in focus or not, ordered by spec file then line. |
| `findings` | array of [finding](#finding-object) | Every finding whose severity is not `off`, ordered by file, then line, then category, then message. |
| `extensions` | object | Reserved, empty. |

### `run` object

| Property | Type | Meaning |
|---|---|---|
| `advisory` | boolean | True when the run was advisory: `exit_code` is 0 whatever `verdict` says. |
| `fail_on` | `"info"` \| `"warning"` \| `"error"` | Lowest severity that blocks. |
| `id_pattern` | string | Regular expression a requirement ID must match. |
| `focus.everything` | boolean | True when no focus was given: every known requirement is in focus. |
| `focus.files` | array of string | Spec files given as `--focus` / `focus`. |
| `focus.ids` | array of string | ID globs given as `--focus-id` / `focus_ids`. |
| `spec_files` | array of string | Every known spec file read (`--specs` and `--focus` together). |
| `test_roots` | array of string | Directories scanned for tags (`--tests` / `tests`). |
| `results_files` | array of string | JUnit XML files read (`--results` / `results`, expanded). |

### `summary` object

`summary.requirements`: counts over the coverage matrix, each `minimum: 0`.

| Property | Meaning |
|---|---|
| `known` | Requirements declared by the known specs. |
| `in_focus` | Requirements whose coverage this run demanded. |
| `covered` | In focus and covered. |
| `failed` | In focus and failed. |
| `skipped` | In focus and skipped. |
| `not_run` | In focus and not run. |
| `uncovered` | In focus and uncovered. |
| `non_testable` | In focus and non-testable. |

`summary.tests`:

| Property | Meaning |
|---|---|
| `total` | Test cases in the results files. |
| `bound` | Test cases citing at least one well-formed reference. |
| `untagged` | Test cases carrying no tag at all. |

`summary.findings`:

| Property | Meaning |
|---|---|
| `error` | Findings of severity `error`. |
| `warning` | Findings of severity `warning`. |
| `info` | Findings of severity `info`. |
| `blocking` | Findings at or above `fail_on`. |

## Requirement object

One entry per known requirement, whether or not it is in focus.

| Property | Type | Meaning |
|---|---|---|
| `id` | string | Requirement ID. |
| `revision` | integer ≥ 1 | Current revision, as declared by the spec (default 1 when the spec omits it). |
| `ref` | string | `ID~REVISION`: the text a tag must cite to bind at the current revision. |
| `title` | string, optional | Present when the spec gives a title. |
| `statement` | string | The requirement's statement. |
| `status` | `"active"` \| `"non_testable"` | See [Requirement status](#requirement-status). |
| `justification` | string, optional | Why the requirement is non-testable. Present only when `status` is `non_testable` and the spec supplied one. |
| `location` | [location](#location-object) | Where the requirement is declared. |
| `in_focus` | boolean | Whether this run demanded the requirement's coverage. |
| `coverage` | string enum | See [Coverage states](#coverage-states). |
| `tests` | array of [bound_test](#bound_test-object) | Tests bound to the requirement at its current revision, ordered by results file, classname, name, then location. |
| `extensions` | object | Reserved, empty. |

### Requirement status

- `active` — a test is expected to verify the requirement. Its coverage is
  judged by `coverage`.
- `non_testable` — no automated test is expected. A missing `justification`
  raises `unjustified_non_testable`; a test that still cites it raises
  `bound_non_testable`.

### Coverage states

Computed per requirement from its bound tests' outcomes, in this precedence:

| `coverage` | Meaning |
|---|---|
| `covered` | At least one bound test passed. |
| `failed` | None passed, at least one failed or errored. |
| `skipped` | None passed or failed, at least one was skipped. |
| `not_run` | Tagged tests exist in source but appear in no results file. |
| `uncovered` | No bound test at all. |
| `non_testable` | `status` is `non_testable` (independent of any test that still cites it — see `bound_non_testable`). |

A coverage state other than `covered` or `non_testable`, on a requirement
`in_focus`, is a coverage gap and raises the matching finding category (see
[Finding categories](#finding-categories)) — but only for requirements in
focus (see [docs/pipeline-gate.md](pipeline-gate.md#known-specs-vs-focus)).

### `bound_test` object

| Property | Type | Meaning |
|---|---|---|
| `name` | string | Test case name from the results file. For a `not_run` test, the label of its source tag (may be empty). |
| `classname` | string, optional | Test case classname from the results file. |
| `suite` | string, optional | Enclosing test suite name from the results file. |
| `outcome` | string enum | See [Test outcomes](#test-outcomes). |
| `results_file` | string, optional | Results file recording the test. Absent for a `not_run` test. |
| `source` | [location](#location-object), optional | Source tag the binding was matched to. Absent when no test root contains it. |

### Test outcomes

| `outcome` | Meaning |
|---|---|
| `passed` | The test case passed. |
| `failed` | The test case reported a failure. |
| `errored` | The test case reported an error. |
| `skipped` | The test case was skipped. |
| `not_run` | The tag exists in source, but no results file records this test (see `tag_not_in_results`). |

## Finding object

One entry per raised finding whose configured severity is not `off`.

| Property | Type | Meaning |
|---|---|---|
| `category` | string enum | Kind of finding. See [Finding categories](#finding-categories). |
| `severity` | `"info"` \| `"warning"` \| `"error"` | Configured severity of the category (never `off`: findings of severity `off` are not emitted). |
| `blocking` | boolean | True when `severity` is at or above `run.fail_on`. |
| `message` | string | Human-readable explanation, safe to display as-is. |
| `requirement_id` | string, optional | Requirement concerned, when there is one. |
| `location` | [location](#location-object) | Where to act: a spec line, a source tag line, or a results file. |
| `test` | object, optional | Test concerned, when the finding comes from a results file: `name`, `classname` (optional), `results_file`. |

### Location object

| Property | Type | Meaning |
|---|---|---|
| `file` | string | Path relative to the working directory, forward slashes. |
| `line` | integer ≥ 1, optional | 1-based line. Absent when the location is a whole file. |

### Finding categories

For every category: what it means, its default severity, where its
`location` points, whether it is raised only for requirements in focus, and
what a consumer should do about it. Severities are configurable per category
(see [docs/configuration.md](configuration.md#severities)); the values below
are the defaults.

| Category | Default severity | `location` | Focus-gated | Meaning |
|---|---|---|---|---|
| `uncovered_requirement` | error | requirement declaration | yes | A requirement in focus has no bound test. |
| `failed_requirement` | error | requirement declaration | yes | A requirement in focus has bound tests, none of which passed; at least one failed or errored. |
| `skipped_requirement` | error | requirement declaration | yes | A requirement in focus has bound tests, none passed or failed, at least one was skipped. |
| `not_run_requirement` | error | requirement declaration | yes | A requirement in focus has tagged tests in source, but none of them appear in any results file. |
| `failing_bound_test` | error | source tag if found, else the results file | yes | One specific test bound to a requirement failed or errored, while another bound test on the same requirement passed (the requirement itself is `covered`, but this test still needs attention). |
| `tag_not_in_results` | warning | source tag | no | A tag in source cites a known requirement at its current revision, but no results file records that test: it did not run, or the tag sits where the runner does not report it. |
| `orphan_tag` | error | source tag or results file | no | A tag cites a requirement ID that no known spec declares. |
| `revision_mismatch` | error | source tag or results file | no | A tag cites a revision of a requirement that no longer matches the spec's current revision. |
| `malformed_tag` | error | source tag or results file | no | A `[verifies ...]` tag does not parse: bad ID shape, missing revision, or similar. |
| `unjustified_non_testable` | error | requirement declaration | no | A requirement is `non_testable` but has no justification. |
| `bound_non_testable` | warning | requirement declaration | no | A requirement is `non_testable`, yet one or more tests cite it. |
| `duplicate_id` | error | the later declaration | no | The same requirement ID is declared more than once across the known specs. |
| `malformed_requirement` | error | the offending line | no | A spec declaration is invalid: bad revision, empty statement (Markdown), or an invalid key/value (YAML). |
| `untagged_test` | info | the results file | no | A test case in the results carries no `[verifies ...]` tag and no `verifies` property. |

What to do about each category, and who owns the action, is specified once, in the
[routing table of docs/pipeline-gate.md](pipeline-gate.md#routing-table-finding-category-to-owner-and-action).

## Ordering guarantees

- `requirements`: by spec file path, then by declaration line.
- `requirements[].tests`: by results file, then classname, then name, then
  the test's location.
- `findings`: by location file, then location line, then category
  (declaration order in the schema), then message.
- `run.spec_files`, `run.test_roots`, `run.results_files`: as resolved by the
  tool (test roots are deduplicated and sorted).

Combined with the determinism guarantee, two runs over the same inputs
produce byte-identical reports, safe to diff or hash.

## `verdict`, `exit_code` and advisory

`verdict` is `"fail"` whenever at least one finding is blocking
(`severity >= run.fail_on`), regardless of advisory mode. `exit_code` folds
in advisory mode: it is `0` unless `verdict` is `"fail"` **and** `run.advisory`
is `false`, in which case it is `1`. A tool failure never writes a report; its
exit code (`2`) is not representable inside the JSON report at all — a
consumer must treat "no report file, or a non-zero, non-one process exit"
distinctly from `verdict`. See
[docs/pipeline-gate.md](pipeline-gate.md#exit-codes) for how an orchestrator
should branch on each exit code.

## Complete example

Run against the fixture pipeline, restricted to the plan just implemented
(`ABC-101`), from `fixtures/pipeline`:

```
$ shallnot check --focus plans/ABC-101.md --json-out report.json
```

Terminal output:

```
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

`report.json`:

```json
{
  "schema_version": "1.0",
  "tool": {
    "name": "shallnot",
    "version": "dev"
  },
  "run": {
    "advisory": false,
    "fail_on": "error",
    "id_pattern": "[A-Z]+-[0-9]+\\.AC[0-9]+",
    "focus": {
      "everything": false,
      "files": [
        "plans/ABC-101.md"
      ],
      "ids": []
    },
    "spec_files": [
      "plans/ABC-100.md",
      "plans/ABC-101.md"
    ],
    "test_roots": [
      "client/test",
      "service/tests"
    ],
    "results_files": [
      "client/results/junit.xml",
      "service/results/junit.xml"
    ]
  },
  "verdict": "pass",
  "exit_code": 0,
  "summary": {
    "requirements": {
      "known": 5,
      "in_focus": 3,
      "covered": 2,
      "failed": 0,
      "skipped": 0,
      "not_run": 0,
      "uncovered": 0,
      "non_testable": 1
    },
    "tests": {
      "total": 3,
      "bound": 3,
      "untagged": 0
    },
    "findings": {
      "error": 0,
      "warning": 0,
      "info": 0,
      "blocking": 0
    }
  },
  "requirements": [
    {
      "id": "ABC-100.AC1",
      "revision": 1,
      "ref": "ABC-100.AC1~1",
      "statement": "WHEN a customer opens the invoices page THE SYSTEM SHALL list their invoices, newest first.",
      "status": "active",
      "location": { "file": "plans/ABC-100.md", "line": 5 },
      "in_focus": false,
      "coverage": "covered",
      "tests": [
        {
          "name": "test_invoices_are_listed_newest_first",
          "classname": "tests.test_invoices",
          "suite": "pytest",
          "outcome": "passed",
          "results_file": "service/results/junit.xml",
          "source": { "file": "service/tests/test_invoices.py", "line": 6 }
        }
      ],
      "extensions": {}
    },
    {
      "id": "ABC-100.AC2",
      "revision": 1,
      "ref": "ABC-100.AC2~1",
      "statement": "WHEN a customer has no invoice THE SYSTEM SHALL show an empty state.",
      "status": "active",
      "location": { "file": "plans/ABC-100.md", "line": 7 },
      "in_focus": false,
      "coverage": "uncovered",
      "tests": [],
      "extensions": {}
    },
    {
      "id": "ABC-101.AC1",
      "revision": 1,
      "ref": "ABC-101.AC1~1",
      "statement": "WHEN the client requests an invoice export THE SYSTEM SHALL return a PDF named after the invoice number.",
      "status": "active",
      "location": { "file": "plans/ABC-101.md", "line": 10 },
      "in_focus": true,
      "coverage": "covered",
      "tests": [
        {
          "name": "test_export_is_named_after_the_invoice_number",
          "classname": "tests.test_invoices",
          "suite": "pytest",
          "outcome": "passed",
          "results_file": "service/results/junit.xml",
          "source": { "file": "service/tests/test_invoices.py", "line": 12 }
        }
      ],
      "extensions": {}
    },
    {
      "id": "ABC-101.AC2",
      "revision": 1,
      "ref": "ABC-101.AC2~1",
      "statement": "WHEN a customer clicks \"Export\" THE SYSTEM SHALL start the download without leaving the page.",
      "status": "active",
      "location": { "file": "plans/ABC-101.md", "line": 12 },
      "in_focus": true,
      "coverage": "covered",
      "tests": [
        {
          "name": "invoice page > starts the download in place when Export is clicked [verifies ABC-101.AC2~1]",
          "classname": "test/export.test.ts",
          "suite": "test/export.test.ts",
          "outcome": "passed",
          "results_file": "client/results/junit.xml",
          "source": { "file": "client/test/export.test.ts", "line": 5 }
        }
      ],
      "extensions": {}
    },
    {
      "id": "ABC-101.AC3",
      "revision": 1,
      "ref": "ABC-101.AC3~1",
      "statement": "THE exported PDF SHALL match the brand guidelines.",
      "status": "non_testable",
      "justification": "reviewed by the brand team on each template change; no automated check covers visual identity.",
      "location": { "file": "plans/ABC-101.md", "line": 14 },
      "in_focus": true,
      "coverage": "non_testable",
      "tests": [],
      "extensions": {}
    }
  ],
  "findings": [],
  "extensions": {}
}
```

## `jq` one-liners

Against a report produced without `--focus` (every requirement in focus),
run from `fixtures/pipeline`:

```
$ shallnot check --json-out report.json
shallnot: FAIL
...
$ echo $?
1
```

List blocking findings:

```
$ jq '[.findings[] | select(.blocking)]' report.json
[
  {
    "category": "uncovered_requirement",
    "severity": "error",
    "blocking": true,
    "message": "requirement ABC-100.AC2~1 has no bound test",
    "requirement_id": "ABC-100.AC2",
    "location": {
      "file": "plans/ABC-100.md",
      "line": 7
    }
  }
]
```

List requirements in focus that are not covered (excludes `non_testable`,
which is a deliberate exemption, not a gap):

```
$ jq '[.requirements[] | select(.in_focus and (.coverage != "covered" and .coverage != "non_testable"))]' report.json
[
  {
    "id": "ABC-100.AC2",
    "revision": 1,
    "ref": "ABC-100.AC2~1",
    "statement": "WHEN a customer has no invoice THE SYSTEM SHALL show an empty state.",
    "status": "active",
    "location": {
      "file": "plans/ABC-100.md",
      "line": 7
    },
    "in_focus": true,
    "coverage": "uncovered",
    "tests": [],
    "extensions": {}
  }
]
```

Verdict and exit code together:

```
$ jq '{verdict, exit_code}' report.json
{
  "verdict": "fail",
  "exit_code": 1
}
```

Count findings per category:

```
$ jq '[.findings[].category] | group_by(.) | map({category: .[0], count: length})' report.json
```

## Other output formats

### Terminal (`--format terminal`, the default)

A human-oriented rendering on stdout: a header (`shallnot: PASS|FAIL`, the
focus description, and the three summary lines), a `REQUIREMENTS` table
restricted to requirements in focus (each with its bound tests indented
below), and, when any finding exists, a `FINDINGS` table. Not meant to be
parsed; use `--format json` or `--json-out` for automation.

### Markdown (`--format markdown` or `--markdown-out <file>`)

Suited to posting as a GitHub Actions job summary (the bundled composite
action writes it to `GITHUB_STEP_SUMMARY`, see
[docs/pipeline-gate.md](pipeline-gate.md#github-actions)). It renders the same
header as a bullet list, a "Requirements in focus" table (`Requirement`,
`Coverage`, `Bound tests` as a counted summary like `1 passed`, `Declared
at`), and, when any finding exists, a "Findings" table (`Severity`,
`Category`, `Location`, `Message`). Table cells escape `|` and newlines.

Example (`--focus plans/ABC-101.md --format markdown`):

```
## shallnot: PASS

- **Focus:** plans/ABC-101.md
- **Requirements:** 5 known, 3 in focus (2 covered, 0 failed, 0 skipped, 0 not run, 0 uncovered, 1 non-testable)
- **Tests:** 3 in results, 3 bound, 0 untagged
- **Findings:** 0 error, 0 warning, 0 info (0 blocking)

### Requirements in focus

| Requirement | Coverage | Bound tests | Declared at |
|---|---|---|---|
| `ABC-101.AC1~1` | covered | 1 passed | `plans/ABC-101.md:10` |
| `ABC-101.AC2~1` | covered | 1 passed | `plans/ABC-101.md:12` |
| `ABC-101.AC3~1` | non_testable | none | `plans/ABC-101.md:14` |
```

### GitHub annotations (`--format github` or `--github-annotations`)

Prints one GitHub Actions workflow command per finding
(`::error file=...,line=...,title=shallnot <category>::<message>`), which
GitHub renders as an inline annotation on the offending line. Severity maps
`error` to `error`, `warning` to `warning`, `info` to `notice`.
`--github-annotations` adds these commands after the primary `--format`
output (unless the primary format is already `github`); it does not replace
it. Only findings produce annotations; the requirements matrix is not
represented.

Example (`shallnot check --format github` on the unfocused run above):

```
::error file=plans/ABC-100.md,line=7,title=shallnot uncovered_requirement::requirement ABC-100.AC2~1 has no bound test
```
