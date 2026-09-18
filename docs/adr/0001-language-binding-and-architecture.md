# ADR 0001: Language, binding convention and architecture

Status: accepted

## Context

shallnot must be a deterministic, model-free gate distributed as one static
binary, usable from CI, a terminal, and an orchestrator of coding agents. It
joins three inputs: specs with revisioned requirement IDs, test sources, and
test results. The hard constraint on the binding syntax is that the link
between a tag and a test's result is read from the results file, not assumed.

## Decisions

### Go

Go, with `CGO_ENABLED=0`. Cross-compiling six static targets is one
environment variable per target with the standard toolchain; the standard
library covers XML, JSON, regular expressions (RE2: linear time, no
catastrophic backtracking on hostile input) and file walking, leaving two
runtime dependencies (a YAML parser and a `**` glob matcher). Rust's sum types
are stronger, and were the main argument for it; they are compensated by
typed enumerations with text marshalling (`internal/domain`), JSON Schemas
checked against those enumerations in tests (`schemas/schemas_test.go`), and
a test that closes every schema object before validating real reports, which
proves the producer emits nothing undocumented. Static linking on Linux and
Windows on ARM needs extra targets, linkers or musl in Rust; none of that
exists in Go.

### The tag surfaces in JUnit XML; the source scan only locates

A test is bound to a requirement only when the tag `[verifies ID~REV]` appears
in the results file: in the test case `name`, its `classname`, or a
`verifies` property. The source scan never creates a binding. It supplies
file and line, and detects tags that reached no results file.

Per ecosystem:

- Jest, Vitest: test or `describe` title. Both default reporters put the full
  title path in the test case name.
- JUnit 5: `@DisplayName`. Gradle writes method display names natively;
  Surefire needs its stateless reporter's phrased-name options.
- pytest: a `verifies` marker copied into `user_properties` by a
  `pytest_collection_modifyitems` hook in the project's `conftest.py`, or
  `record_property`. Function names cannot carry an ID, and pytest reports
  nothing else about a test by default.
- Go: subtest names.

Rejected:

- **Binding by source adjacency** (a comment above a test function, joined to
  results by file and function name, as OpenFastTrace-style tags would
  suggest). The link would be inferred from naming rules per language and per
  runner, and a comment would count as coverage when the join guessed wrong.
- **A runtime plugin per framework** (pytest plugin, Jest reporter, JUnit
  extension). It would make the results self-describing, but installs a
  runtime dependency in every project and ties releases to four ecosystems.
  The pytest hook is five lines of project code instead.
- **Decorators or annotations without surfacing** (`@Verifies("X~1")` in
  Java): invisible in results unless a custom extension is installed.
- **A tag without a revision meaning "current revision"**: it would silently
  survive a change of meaning, which is what revisions exist to catch.
- **Docstrings and comments for pytest**: not reported by the runner.

### Matching source tags to results

A source tag is labelled with the static part of the string literal around it
(or the decorated Python definition). It matches a test case that cites the
same reference and whose suite, classname or name contains the label, after
folding whitespace and underscores. A wholly dynamic title degrades to
matching by reference. Unmatched tags are `tag_not_in_results`.

### Known specs and focus are separate inputs

Known specs resolve citations; focus selects the requirements whose coverage
the run demands. Tag and spec defects are reported regardless of focus,
because they are defects of the repository, not of the increment.

### A missing input is a tool failure

A spec, results or test-root argument that matches nothing exits 2. An absent
results file would otherwise read as "nothing ran", which is a verdict.
Report files requested by a run are deleted before the run so that a failure
cannot leave an earlier verdict in place.

### Architecture

`internal/domain` (types, tag grammar, policy) and `internal/analysis` (the
join, the findings, the verdict) are pure: no file access, no adapters, which
a test enforces. `internal/adapters` holds spec loaders, the source scan with
one `TagExtractor` strategy per tag syntax, the JUnit XML reader, the config
file, and one `Reporter` per output format. `internal/app` resolves inputs and
composes the adapters; `internal/cli` maps flags and exit codes.

## Extension points

These are outside the deterministic core's present scope; the structure
leaves room for them without a breaking change:

- **Code coverage join** (code executed by no bound test is possibly outside
  the spec) and **mutation score per requirement**: new inputs read by new
  adapters, reported under the `extensions` object of each requirement and of
  the report, which the schema reserves. `BoundTest` already identifies tests
  by results file, classname and name, the key coverage-per-test data uses.
- **A pluggable judge** (typed question in, typed answer with confidence out)
  for semantic checks such as whether a test really verifies its requirement:
  a separate command consuming the JSON report, never linked into `check`,
  so that the gate stays model-free and the architecture test on networking
  packages keeps holding. Its answers belong under `extensions`.
- **OpenFastTrace import**: a `spec.Loader` format for OFT Markdown items and
  a `TagExtractor` for `[type->id]` tags, mapping `type~name~revision` to an ID
  and a revision.
- **An empirical study** of whether test-level citation avoids the consistency
  penalty measured for per-line citation: needs run history from agent
  pipelines, which the JSON report provides.

## Consequences

- A project must make its runner report the tag: free for Jest, Vitest and
  Gradle; one reporter option for Surefire; a five-line hook for pytest.
- Tags placed where a runner does not report them (class-level display names
  under Gradle) do not bind, and are reported rather than ignored.
- The JSON report is a public API from its first version: additions are
  minor versions, consumers ignore unknown properties.
