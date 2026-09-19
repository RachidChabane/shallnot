# Configuration reference

`shallnot check` is driven by flags, by an optional `shallnot.yaml`
configuration file, and by built-in defaults. This document is the complete
reference of the configuration file, of every `check` flag, of how they
combine, and of the other commands (`schema`, `version`, `help`).

See also [docs/report.md](report.md) for the output the check produces and
[docs/pipeline-gate.md](pipeline-gate.md) for using it as an automated gate.

## The configuration file

`shallnot.yaml` (schema: [schemas/config.schema.json](../schemas/config.schema.json),
also printed by `shallnot schema config`) is a YAML mapping. Unknown keys are
refused (the decoder rejects them, exit code 2). Relative paths inside it are
resolved against the directory holding the file, not against the working
directory.

| Key | Type | Default | Semantics |
|---|---|---|---|
| `version` | integer, required | — | Must be `1`. Any other value is a tool failure. |
| `id_pattern` | string | `[A-Z][A-Z0-9]*-[0-9]+(?:\.[A-Za-z0-9]+)*` | RE2 regular expression a whole requirement ID must match. Must not match text containing `~`, `,`, `]` or whitespace. |
| `specs` | array of string | `[]` | Known specs: files, directories (scanned recursively) or globs (`**` supported). Resolved relative to the config file's directory. |
| `focus` | array of string | `[]` | Spec files whose requirements this run must find covered. Each entry is also a known spec. Resolved relative to the config file's directory. |
| `focus_ids` | array of string | `[]` | Globs on requirement IDs this run must find covered. Not path-resolved (matched against IDs, not files). |
| `tests` | array of string | `[]` | Test source root directories scanned for tags. Resolved relative to the config file's directory. |
| `results` | array of string | `[]` | JUnit XML results: files, directories or globs. Resolved relative to the config file's directory. |
| `exclude` | array of string | `[]` | Globs of source paths the scan ignores, matched against the path as reported. Not path-resolved. |
| `default_excludes` | boolean | `true` | Whether dependency and build directories (`.git`, `node_modules`, `target`, `build`, `dist`, `.venv`, `venv`, `__pycache__`, `.gradle`, `.pytest_cache`, `.mypy_cache`, `coverage`, each with `**/` around them) are excluded from the scan in addition to `exclude`. |
| `advisory` | boolean | `false` | Report everything but always exit 0. |
| `strict` | boolean | `false` | Treat `untagged_test` as `error` instead of its default `info`. Equivalent to `severities: {untagged_test: error}` but does not prevent a further explicit override (see precedence below). |
| `fail_on` | string | `error` | Lowest blocking severity: `info`, `warning` or `error`. `off` is rejected: use `advisory` to never block. |
| `severities` | map of category to severity | `{}` | Per-category severity override. Keys are the finding categories of [docs/report.md](report.md#finding-categories); values are `off`, `info`, `warning` or `error`. |
| `test_commands` | array of string | `[]` | Command lines `shallnot gate` runs, in order, before checking. See [Gate, init and hook](#gate-init-and-hook) below. |

At least one of `specs` or `focus` must resolve to files across config and
flags combined, and `results` must resolve to at least one file: their
absence is a tool failure (`no spec given...` / `no results file given...`),
never a silent "nothing to check".

## Flags of `shallnot check`

Every flag has a config-file equivalent, except `--config` and `--no-config`
(which select the configuration file itself) and `--json-out`/`--markdown-out`/
`--format`/`--github-annotations` (which control output, not analysis).

| Flag | Config equivalent | Semantics |
|---|---|---|
| `--config <file>` | — | Configuration file to load, instead of the default `shallnot.yaml` lookup. |
| `--no-config` | — | Ignore `shallnot.yaml` in the working directory (and refuse to combine with `--config`). |
| `--id-pattern <expression>` | `id_pattern` | Overrides the pattern entirely. |
| `--specs <glob>` (repeatable) | `specs` | Replaces the whole list; see precedence. |
| `--focus <glob>` (repeatable) | `focus` | Replaces the whole list. |
| `--focus-id <glob>` (repeatable) | `focus_ids` | Replaces the whole list. |
| `--tests <directory>` (repeatable) | `tests` | Replaces the whole list. |
| `--results <glob>` (repeatable) | `results` | Replaces the whole list. |
| `--exclude <glob>` (repeatable) | `exclude` | Replaces the whole list. |
| `--no-default-excludes` | `default_excludes: false` | Scans dependency and build directories too. |
| `--severity <category=severity>` (repeatable) | `severities` | Sets one category; repeatable flags apply on top of the config's `severities` map (see precedence). |
| `--advisory` | `advisory: true` | Sets advisory mode; a flag cannot turn it off if the config sets it. |
| `--strict` | `strict: true` | Sets `untagged_test` to `error`; a flag cannot turn it off if the config sets it. |
| `--fail-on <severity>` | `fail_on` | Overrides the lowest blocking severity. |
| `--format <name>` | — | Output format printed on stdout: `terminal` (default), `json`, `markdown`, `github`. |
| `--json-out <file>` | — | Also writes the JSON report to this file. |
| `--markdown-out <file>` | — | Also writes the Markdown summary to this file. |
| `--github-annotations` | — | Also prints GitHub Actions workflow-command annotations on stdout, in addition to `--format`. |
| `--test-command <line>` (repeatable) | `test_commands` | Replaces the whole list; see precedence. Used by `gate`, ignored by `check`. |

Flags with `directory`/`glob`/`file`/`expression` argument descriptions above
are as `flag` prints them in `--help`; run `shallnot check --help` for the
canonical, up-to-date list.

## Precedence

Settings are resolved in three layers, later overriding earlier:

1. **Defaults** — `id_pattern` = the built-in pattern, `default_excludes` =
   `true`, `fail_on` = `error`, everything else empty/false.
2. **Configuration file** — loaded per `--config`/`--no-config`/auto-discovery
   below; every key it sets replaces the corresponding default.
3. **Flags** — each flag that was actually passed overrides the
   corresponding config value. **List flags (`--specs`, `--focus`,
   `--focus-id`, `--tests`, `--results`, `--exclude`) replace the config's
   list wholesale when passed at least once; they do not merge with it.**
   `--severity category=severity` is the one exception among lists: each
   assignment sets one category on top of whatever `severities` (and
   `strict`) the config already set, so `--severity` flags and a config
   `severities` map combine key by key. `--advisory` and `--strict` are
   one-way switches: passing the flag turns the option on even if the config
   already had it on; there is no flag to force it off when the config turns
   it on.

## Path resolution

- Paths inside `shallnot.yaml` (`specs`, `focus`, `tests`, `results`) are
  resolved relative to the directory holding that file — not the working
  directory the command runs from, and not `--config`'s directory if that
  differs from the file's own location (they are the same location, since
  `--config` names the file directly).
  `focus_ids` and `exclude` are glob/ID patterns, not paths, and are never
  rebased.
- Paths given as flags (`--specs`, `--focus`, `--tests`, `--results`) are
  resolved relative to the current working directory, following normal shell
  and OS conventions; `shallnot` does not rebase them.
- `--json-out` and `--markdown-out` (and their config equivalents do not
  exist — they are flag-only) are always relative to the working directory.
- A `--specs`/`--focus`/`--results` pattern (file, directory or glob) that
  matches no file is a tool failure (exit 2): a typo or an empty result set
  is never silently treated as "nothing to check".

## `--config`, `--no-config` and auto-discovery

- `--config <file>` loads exactly that file as the configuration.
- `--no-config` ignores `shallnot.yaml` in the working directory even if
  present, and skips loading any configuration file; settings come from
  defaults and flags only.
- `--config` and `--no-config` are mutually exclusive; passing both is a
  tool failure.
- With neither flag: if `shallnot.yaml` exists in the working directory, it
  is loaded; otherwise defaults are used.

## Severities and `fail_on`

Every finding category (see [docs/report.md](report.md#finding-categories))
has a default severity: `off`, `info`, `warning` or `error`. `severities`
(config) and repeated `--severity category=severity` (flag) override a
category's severity individually; setting a category to `off` silences it —
findings of that category are not emitted at all, and do not appear in the
report's `findings` array or in `summary.findings`.

`fail_on` is the lowest severity that blocks: a finding with
`severity >= fail_on` sets `blocking: true` on itself and, if at least one
such finding exists, sets the run's `verdict` to `fail`. `fail_on: off` is
rejected (`fail_on: "off" is not a blocking severity; use advisory mode to
never block`) — advisory mode is the way to never fail the exit code.

`strict` (config) / `--strict` (flag) is exactly `severities: {untagged_test:
error}` applied on top of whatever severities are already set; it does not
touch any other category.

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Clean: no finding reached the blocking severity, or the run is advisory. |
| `1` | Blocked: at least one finding reached the blocking severity, and the run is not advisory. |
| `2` | Tool failure: no verdict was produced (bad flags, bad or missing config, unreadable or unmatched input, invalid spec document). Nothing is printed on stdout; the message goes to stderr; any file at `--json-out`/`--markdown-out` is deleted at the start of the run, so a stale report from an earlier run can never be mistaken for this run's verdict. |

## Gate, init and hook

- `shallnot gate` takes the same flags as `shallnot check`, plus
  `--test-command`. It runs every `test_commands` command line (config or
  `--test-command`) in order, in the configuration file's directory, through
  the platform shell (`sh -c` on Unix, `cmd /C` on Windows), sending each
  command's own stdout and stderr to `shallnot`'s standard error. A
  command's own exit status is logged but never affects the verdict —
  a test runner that exits non-zero because tests failed is a result for the
  check to report, not a reason to stop early. With no `test_commands`
  configured or passed, `gate` is a tool failure (exit 2) before running
  anything.
- **Freshness rule**: after the commands run, `gate` requires every file
  matched by `results` to exist and to have a modification time different
  from the one it had (or its absence) before the commands ran. A results
  file that the test commands left untouched, or that still does not exist,
  is a tool failure (exit 2): `gate` never checks a stale or missing report
  as if it were this run's verdict.
- `shallnot init` equips a repository with a starter `shallnot.yaml`, agent
  instructions and end-of-turn hooks. See
  [docs/agents.md](agents.md#shallnot-init) for the full reference: every
  file it writes, its flags, and its idempotence guarantee.
- `shallnot hook <harness>` answers an agent harness's end-of-turn hook by
  gating the project and replying in that harness's protocol. See
  [docs/agents.md](agents.md#end-of-turn-hooks) for the full protocol.

## Other commands

- `shallnot schema <name>` — prints one JSON Schema document to stdout:
  `report` ([schemas/report.schema.json](../schemas/report.schema.json)),
  `config` ([schemas/config.schema.json](../schemas/config.schema.json)) or
  `spec` ([schemas/spec.schema.json](../schemas/spec.schema.json)). Any other
  name, or a wrong argument count, is a tool failure.
- `shallnot version` (also `--version`) — prints `shallnot <version>` to
  stdout and exits 0.
- `shallnot help` (also `--help`, `-h`, or running `shallnot` with no
  arguments) — prints the top-level usage, including the command list and
  the exit code table, to stdout (stderr and exit 2 when invoked as no
  arguments at all). Flag-level help for a check is `shallnot check --help`.

## Complete runnable example

[fixtures/pipeline/shallnot.yaml](../fixtures/pipeline/shallnot.yaml):

```yaml
version: 1
id_pattern: '[A-Z]+-[0-9]+\.AC[0-9]+'
specs:
  - plans
tests:
  - service/tests
  - client/test
results:
  - service/results
  - client/results/*.xml
severities:
  tag_not_in_results: error
```

Run from `fixtures/pipeline`, with no flags (the file is auto-discovered):

```
$ shallnot check
shallnot: FAIL
focus: every known requirement
requirements: 5 known, 5 in focus (3 covered, 0 failed, 0 skipped, 0 not run, 1 uncovered, 1 non-testable)
tests: 3 in results, 3 bound, 0 untagged
findings: 1 error, 0 warning, 0 info (1 blocking)

REQUIREMENTS
  ABC-100.AC1~1  covered  plans/ABC-100.md:5
      passed  tests.test_invoices › test_invoices_are_listed_newest_first  service/tests/test_invoices.py:6
  ABC-100.AC2~1  uncovered  plans/ABC-100.md:7
  ABC-101.AC1~1  covered  plans/ABC-101.md:10
      passed  tests.test_invoices › test_export_is_named_after_the_invoice_number  service/tests/test_invoices.py:12
  ABC-101.AC2~1  covered  plans/ABC-101.md:12
      passed  test/export.test.ts › invoice page > starts the download in place when Export is clicked [verifies ABC-101.AC2~1]  client/test/export.test.ts:5
  ABC-101.AC3~1  non_testable  plans/ABC-101.md:14

FINDINGS
  error  uncovered_requirement  plans/ABC-100.md:7  requirement ABC-100.AC2~1 has no bound test
$ echo $?
1
```

`--no-config` in the same directory, with no `--specs`/`--focus` flag, fails
as a tool error rather than as a verdict:

```
$ shallnot check --no-config
shallnot: error: no spec given: pass --specs or --focus, or set `specs` in the config file
$ echo $?
2
```

An unknown key in the configuration file is also a tool failure:

```
$ cat > bad.yaml <<'EOF'
version: 1
bogus: true
EOF
$ shallnot check --config bad.yaml
shallnot: error: bad.yaml: yaml: unmarshal errors:
  line 2: field bogus not found in type config.File
$ echo $?
2
```
