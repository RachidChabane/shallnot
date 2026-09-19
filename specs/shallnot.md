# shallnot requirements

The requirements of shallnot itself, in its own Markdown spec format.
`shallnot check` traces them to the Go tests of this repository in CI.

## Specifications

- **SN-1~1**: WHEN a Markdown line starts, after optional heading, list or
  emphasis markup, with a requirement ID followed by a colon THE SYSTEM SHALL
  read it as a requirement declaration with its statement and location.
- **SN-2~1**: WHEN a declaration states no revision THE SYSTEM SHALL give the
  requirement revision 1.
- **SN-3~1**: WHEN a requirement carries a non-testable marker THE SYSTEM SHALL
  record the non-testable status and the written justification.
- **SN-4~1**: WHEN a declaration sits inside a fenced code block THE SYSTEM
  SHALL ignore it.
- **SN-5~1**: WHEN a spec is written in YAML THE SYSTEM SHALL map it to the
  same requirement model as a Markdown spec.
- **SN-6~1**: WHEN a requirement declaration cannot be read THE SYSTEM SHALL
  report a `malformed_requirement` finding at its line.
- **SN-7~1**: WHEN the project configures an ID pattern THE SYSTEM SHALL
  recognise exactly the IDs matching it, in specs and in tags.

## Tags

- **SN-10~1**: WHEN a text contains `[verifies ID~REV, ...]` THE SYSTEM SHALL
  read every cited reference.
- **SN-11~1**: WHEN a tag cites no requirement, no revision, an invalid
  revision or an ID outside the pattern THE SYSTEM SHALL report a
  `malformed_tag` finding, once, at the source line when known.
- **SN-12~1**: WHEN a runner has replaced the spaces of a tag with underscores
  THE SYSTEM SHALL read the tag all the same.
- **SN-13~1**: WHEN a Python file carries a `pytest.mark.verifies` marker or a
  `record_property("verifies", ...)` call THE SYSTEM SHALL find the tag and
  the test, class or module it applies to.
- **SN-14~1**: WHEN a test root is scanned THE SYSTEM SHALL find tags in any
  text file, label each with the static part of the test title around it, and
  skip excluded paths, binary files and files that are other inputs.

## Results

- **SN-20~1**: WHEN a JUnit XML file is read THE SYSTEM SHALL keep passed,
  failed, errored and skipped test cases apart, including those of nested
  suites.
- **SN-21~1**: WHEN a test case carries a tag in its name, its classname or a
  `verifies` property THE SYSTEM SHALL bind it to the cited requirements.
- **SN-22~1**: WHEN given the results that pytest, Jest, Vitest, Maven Surefire
  and Gradle really emit THE SYSTEM SHALL read them and trace each project.
- **SN-23~1**: WHEN a results file is not JUnit XML THE SYSTEM SHALL refuse it
  as a tool failure.

## Analysis

- **SN-30~1**: WHEN at least one test bound to a requirement ran and passed
  THE SYSTEM SHALL report the requirement as covered.
- **SN-31~1**: WHEN no bound test passed THE SYSTEM SHALL distinguish failed,
  skipped, absent from results, and bound to nothing, each with its own
  coverage state and finding category.
- **SN-32~1**: WHEN a tag cites an ID that no known spec declares THE SYSTEM
  SHALL report an `orphan_tag` finding.
- **SN-33~1**: WHEN a tag cites another revision than the spec declares THE
  SYSTEM SHALL report a `revision_mismatch` finding and not count the test as
  coverage.
- **SN-34~1**: WHEN a requirement is non-testable without justification THE
  SYSTEM SHALL report an `unjustified_non_testable` finding.
- **SN-35~1**: WHEN an ID is declared twice THE SYSTEM SHALL report a
  `duplicate_id` finding and keep the first declaration.
- **SN-36~1**: WHEN a test carries no tag THE SYSTEM SHALL report it as
  information, and as an error in strict mode.
- **SN-37~1**: WHEN a bound test fails beside a passing one THE SYSTEM SHALL
  report a `failing_bound_test` finding.
- **SN-38~1**: WHEN a tagged test appears in no results file THE SYSTEM SHALL
  report a `tag_not_in_results` finding even if the requirement is covered.
- **SN-39~1**: WHEN a test cites a non-testable requirement THE SYSTEM SHALL
  report a `bound_non_testable` finding.

## Scope

- **SN-40~1**: WHEN a focus is given THE SYSTEM SHALL demand coverage of the
  requirements in focus only, while every known spec still resolves tags.
- **SN-41~1**: WHEN a focus ID glob is given THE SYSTEM SHALL put the matching
  requirements in focus.
- **SN-42~1**: WHEN several test roots and results files are given THE SYSTEM
  SHALL join them all, across repositories.

## Gate

- **SN-50~1**: WHEN a category's severity is configured THE SYSTEM SHALL apply
  it, block at or above the `fail_on` severity, and stay silent on `off`.
- **SN-51~1**: WHEN the run is advisory THE SYSTEM SHALL report the real
  verdict and exit 0.
- **SN-52~1**: THE SYSTEM SHALL exit 0 when clean, 1 when blocked, and 2 with
  no report on standard output when it cannot produce a verdict, including
  when an input matches no file.
- **SN-53~1**: WHEN a config file is given or present in the working directory
  THE SYSTEM SHALL apply it under the flags, resolve its paths against its
  directory, and refuse unknown keys.
- **SN-54~1**: WHEN a run fails THE SYSTEM SHALL leave no report file of an
  earlier run at the requested output paths.

- **SN-70~1**: WHEN asked to gate THE SYSTEM SHALL run the configured test
  commands in the config file's directory, whatever their exit status, then
  check the results they produced.
- **SN-71~1**: WHEN a results file is absent after the test commands, or was
  left untouched by them, THE SYSTEM SHALL exit 2 without a verdict.

## Agents

- **SN-72~2**: WHEN an agent harness calls its end-of-turn hook in a project
  with a `shallnot.yaml` THE SYSTEM SHALL gate the project and, if it is
  blocked or gives no verdict, send the agent back to work with the reason in
  the harness's own protocol; in a project without a `shallnot.yaml`, in
  advisory mode, and on a gate that passes without having held the agent in
  the session, it SHALL stay silent.
- **SN-73~1**: WHEN a hook has sent the agent back three times in a row THE
  SYSTEM SHALL let the turn end and tell the user the gate is still blocked.
- **SN-79~1**: WHEN the gate passes after the hook had sent the agent back in
  the same session THE SYSTEM SHALL tell the user so in one line.
- **SN-74~1**: WHEN asked to equip a repository THE SYSTEM SHALL write a
  starter config for the test runners it detects without rewriting an existing
  one, add the pytest hook to `conftest.py`, and list what remains to be done.
- **SN-75~3**: WHEN asked to equip a repository THE SYSTEM SHALL install every
  packaged skill, in reading order, as one marked section of `AGENTS.md`,
  import it from `CLAUDE.md`, and copy each skill to `.agents/skills` and to
  `.claude/skills`, leaving other content alone.
- **SN-76~1**: WHEN asked to equip a repository THE SYSTEM SHALL add the
  end-of-turn hook to each selected harness's configuration, keeping the
  settings already there.
- **SN-80~1**: WHEN asked to equip a repository without a list of harnesses
  THE SYSTEM SHALL select Claude Code and each harness whose configuration the
  project holds, and no other.
- **SN-81~1**: WHEN the directory a harness names in its end-of-turn hook holds
  no `shallnot.yaml` THE SYSTEM SHALL gate the nearest parent directory that
  holds one.
- **SN-82~1**: WHEN installing the Gemini CLI hook THE SYSTEM SHALL add
  `AGENTS.md` to Gemini CLI's context file names, keeping the names already
  configured.
- **SN-77~1**: WHEN equipping a repository that is already equipped THE SYSTEM
  SHALL change nothing, and with `--check` it SHALL write nothing and exit 1 if
  a file would change.
- **SN-78~2**: THE plugin package SHALL be a conformant Agent Plugins
  package and a Claude Code plugin carrying the same skills and version, whose
  hooks only call the binary; each skill SHALL name itself and say when to use
  it without being asked.

## Reports

- **SN-60~1**: THE JSON report SHALL conform to the published JSON Schema,
  whose enumerations equal those of the code.
- **SN-61~1**: WHEN run again on the same inputs THE SYSTEM SHALL produce
  byte-identical output in every format.
- **SN-62~1**: THE SYSTEM SHALL render a Markdown summary, on standard output
  or to a file.
- **SN-63~1**: THE SYSTEM SHALL render findings as GitHub Actions annotations
  on the offending file and line.
- **SN-64~1**: THE SYSTEM SHALL render a terminal report with the verdict,
  the matrix in focus and the findings.
- **SN-65~1**: WHEN asked for a schema THE SYSTEM SHALL print the embedded
  JSON Schema of the report, the config file or the YAML spec.

## Architecture

- **SN-66~1**: THE binary SHALL link no networking package.
- **SN-67~1**: THE domain and the analysis SHALL import neither an adapter nor
  file access.
- **SN-68~1**: THE documentation SHALL let an agent with no other context
  integrate the tool into a pipeline.
  - Non-testable: whether a document is sufficient for a reader is judged by
    that reader; each integration report is reviewed instead.
