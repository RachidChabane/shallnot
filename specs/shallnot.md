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
- **SN-84~1**: THE README, `llms.txt` and `docs/agents.md` SHALL name, as the
  harnesses that get an end-of-turn hook, exactly those `shallnot init` can
  install one for.
- **SN-85~1**: THE usage text and `docs/agents.md` SHALL name exactly the
  hooks `shallnot hook` answers.

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
- **SN-83~1**: WHEN a commit is given with `--commit`, or else by the CI
  environment in `GITHUB_SHA` or `CI_COMMIT_SHA`, THE SYSTEM SHALL record it in
  the JSON, Markdown and terminal reports; with none given, the reports SHALL
  carry no commit.
- **SN-65~1**: WHEN asked for a schema THE SYSTEM SHALL print the embedded
  JSON Schema of the report, the config file or the YAML spec.

## Proof gate

- **SN-86~1**: WHEN a requirement's text is reformatted without changing its
  words, by wrapping lines, changing indentation or whitespace, using CRLF
  endings or decomposing characters, THE SYSTEM SHALL compute the same
  statement digest, the SHA-256 of the canonical form `c1` of its title,
  statement, status and justification.
- **SN-87~1**: WHEN the title, the statement, the status or the justification
  of a requirement changes in any other way THE SYSTEM SHALL compute another
  statement digest.
- **SN-88~1**: WHEN a requirement is declared as a Markdown list item, as a
  Markdown heading, or in YAML with a folded or a literal scalar, THE SYSTEM
  SHALL give the same requirement the same statement digest.
- **SN-89~1**: THE JSON report SHALL be schema version 1.2, name the
  canonicalization version in `run.statement_canonicalization`, and carry in
  each requirement its `statement_digest`.
- **SN-90~1**: WHEN a joined statement holds markers `Example:` THE SYSTEM
  SHALL take the text before the first marker as the prose and the text
  after each marker, up to the next, as one example, whichever format
  declared the statement, and SHALL leave the statement digest computed over
  the whole statement.
- **SN-91~1**: WHEN an example is a call of a named operation with typed
  arguments, followed by `=>` and either a typed value or `raises` and an
  exception name, THE SYSTEM SHALL read its operation, its arguments and its
  expected outcome, where the types are datetime, decimal, integer, boolean,
  null and string.
- **SN-92~1**: WHEN a segment after an example marker of a statement is
  outside the example grammar, or holds text after its outcome, THE SYSTEM
  SHALL report a `malformed_example` finding at the requirement's
  declaration, a warning unless configured otherwise, and keep the other
  examples of the statement.
- **SN-93~1**: WHEN a Markdown line reads as an example, after an optional
  list bullet, and no statement absorbs it, because it follows a blank line,
  a non-testable justification or an out-of-scope declaration, or sits
  outside every requirement, THE SYSTEM SHALL report a `detached_example`
  finding at the line, a warning unless configured otherwise.
- **SN-94~1**: THE JSON report SHALL list in each requirement its examples
  with their index, operation, typed arguments by name and expected outcome,
  an empty list when the statement holds none.
- **SN-95~1**: WHEN a Markdown line outside a fenced code block reads
  `Out of scope (DIMENSION): REASON`, with an optional list marker or
  emphasis, THE SYSTEM SHALL record the dimension, the reason and the line as
  an out-of-scope declaration, end the open statement or justification before
  it, and never absorb it into one; WHEN the dimension or the reason is
  missing it SHALL report a `malformed_requirement` finding at the line.
- **SN-96~1**: WHEN a YAML spec holds a top-level `out_of_scope` list of
  dimension and reason pairs THE SYSTEM SHALL record each as an out-of-scope
  declaration at its line, report each unreadable one as a
  `malformed_requirement` finding without failing the run, and publish the
  key in the YAML spec schema.
- **SN-97~1**: WHEN an out-of-scope declaration names a dimension outside the
  closed list of error handling, boundaries, permissions, concurrency,
  compatibility, input validation, time and locale, and state and
  persistence THE SYSTEM SHALL report an `unknown_dimension` finding at its
  line, a warning unless configured otherwise.
- **SN-98~1**: THE JSON report SHALL list the out-of-scope declarations of
  every known spec, with spec file, line, dimension and reason, ordered by
  file and line, an empty list when there are none.
- **SN-99~1**: WHEN given `--baseline` with an earlier JSON report of schema
  version 1.2 or later THE SYSTEM SHALL report a `statement_changed` finding
  for each requirement whose statement digest differs from the baseline's at
  the same `ID~REVISION`, a warning unless configured otherwise, and SHALL
  exit 2 without a report when the baseline cannot be read, holds no
  statement digests, or computed them with another canonicalization.
- **SN-100~1**: THE documentation SHALL name exactly the finding categories,
  the out-of-scope dimensions and the `check` flags the code defines.
- **SN-101~1**: WHEN given a proof file THE SYSTEM SHALL read the in-toto
  Statement v1 envelope and the alignment, discriminating-power and
  completeness predicates with the standard JSON library alone, ignore a
  member the format does not define, ignore a statement of another predicate
  type, and report a `_type` other than Statement v1 or a predicate type that
  differs from the statement's own as a violation.
- **SN-102~1**: WHEN a claim is judged without a confidence, executed with a
  confidence, carries a confidence outside [0,1], or has another class than
  the one its field allows THE SYSTEM SHALL report a violation.
- **SN-103~1**: WHEN a statement is about a criterion THE SYSTEM SHALL require
  exactly one subject named `ID~REV` after its requirement, with a SHA-256
  digest; WHEN it is about a spec THE SYSTEM SHALL require a relative POSIX
  path; and WHEN the recorded cache key differs from the SHA-256 of the
  canonical form of the subject and the inputs THE SYSTEM SHALL report a
  violation.
- **SN-104~1**: WHEN an input role, a verdict, a validity, an outcome, a
  producer, a channel, an isolation level or a decision basis is outside its
  closed set, a digest is not a SHA-256 hex string, or an array is not
  strictly ascending in the order the format fixes THE SYSTEM SHALL report a
  violation.
- **SN-105~1**: WHEN a judged VALID mutant validity lists no executed
  corroboration, an executed claim carries a corroboration or an executor, an
  executor lacks the fields its isolation level requires, a sentinel mutant is
  not EQUIVALENT, an alignment judge was not blind, a completeness proof is not
  advisory, or a mutant is both unattributed and off-target THE SYSTEM SHALL
  report a violation.
- **SN-106~1**: WHEN it reads the golden proofs and predicate schemas that
  spec-proof owns THE SYSTEM SHALL give each the decode result and the exact
  violation codes its manifest states, and SHALL refuse a copied file whose
  SHA-256 differs from the manifest of the copy.
- **SN-108~2**: WHEN shallnot.yaml holds a `proofs` block of the keys
  verified_dir, require, code, trust, thresholds with validity and alignment,
  escalation, signed_pass_basis, min_isolation and max_host_exclusions THE
  SYSTEM SHALL apply each over its default, refuse an unknown key, a value
  outside its closed set or a confidence outside [0,1], refuse to require
  completeness proofs, which are advisory, and publish the block in the config
  schema; and WHEN a threshold is a number THE SYSTEM SHALL take it as the
  default of every confidence source, and WHEN it is a mapping THE SYSTEM SHALL
  read `default` and one number per confidence source, as a kind
  (`execution`, `host-votes`, `system-one`) or as one exact source, and refuse
  a key that names no confidence source.
- **SN-109~1**: WHEN `proofs` is configured THE SYSTEM SHALL read the manifest
  of the verified directory and the statement files it lists with the
  operating system and the standard JSON library alone, record the SHA-256 of
  the manifest and of each statement, and treat as absent a missing manifest,
  an unreadable or malformed manifest, and a statement whose bytes differ from
  the digest the manifest lists, that does not decode or that breaks an
  invariant, while ignoring a statement of another predicate type and a file
  the manifest does not list.
- **SN-110~3**: WHEN `proofs` is configured THE SYSTEM SHALL read the SHA-256
  of the exact bytes of each file that a `code_tree` or `test_source`
  descriptor of a well-formed proof names, resolved against the directory of the
  configuration file whatever the working directory of the process is, record
  no digest for a file that is absent, is not a regular file or whose path is
  not relative POSIX, read no other file of the code or test tree, and record
  the SHA-256 of each known spec file.
- **SN-111~2**: WHEN a proof's subject digest or revision differs from the
  requirement's current statement, or a spec file a completeness proof covers
  changed, THE SYSTEM SHALL classify the proof stale, and under `trust: signed`
  SHALL classify stale a proof produced locally and every proof of a set whose
  manifest was not written by the signature verifier.
- **SN-112~2**: WHEN a proof lists an input of the role `proof` and the
  verified set holds no valid proof of that kind, about the criterion the
  input's `annotations.criterion` names, whose file has the input's digest THE
  SYSTEM SHALL classify it stale, transitively, and SHALL accept several such
  inputs of one kind that name different criteria.
- **SN-113~1**: WHEN a requirement in focus is covered THE SYSTEM SHALL pass
  it only if traceability and every module in `proofs.require` pass; a
  required module with no valid proof SHALL be a `stale_proof` finding that
  names the module, and SHALL block for discriminating power, whose evidence is
  executed, and escalate for alignment, whose evidence is judged; an absent
  module that is not required SHALL raise no finding.
- **SN-114~1**: WHEN a discriminating-power proof records that the tests fail
  on the reference code, or that an example of the criterion fails on it, THE
  SYSTEM SHALL block with `correct_code_fails` or `example_failed`.
- **SN-115~2**: WHEN a mutant survived every test and is VALID THE SYSTEM
  SHALL block with `surviving_valid_mutant` if an example of the criterion
  fails on it, or if its confidence reaches the `thresholds.validity` of its
  confidence source and it names
  a confirmed witness, a build and the other criteria's tests passing, and
  otherwise escalate with `escalated_mutant`; a survivor judged UNDETERMINED
  SHALL escalate.
- **SN-116~1**: WHEN a mutant survived every test and is judged EQUIVALENT THE
  SYSTEM SHALL exclude it only on a differential search that found no
  difference, and WHEN it is judged NOT_VIOLATING, off target or unattributed
  THE SYSTEM SHALL exclude it on an example of another criterion that fails on
  it, or on the judgment alone only where `signed_pass_basis` is judged, the
  confidence reaches the threshold, the isolation reaches the floor and fewer
  than `max_host_exclusions` survivors are already excluded; any other
  survivor SHALL escalate.
- **SN-117~1**: WHEN a VALID mutant was killed THE SYSTEM SHALL grant pass
  credit if an example of the criterion fails on it, or if the confidence
  reaches the threshold, the isolation reaches the floor and the basis allows:
  executed-only grants none to a judgment, corroborated requires a specific
  kill, a build, the other criteria's tests passing and a confirmed witness,
  judged requires nothing more; WHEN no VALID mutant was produced THE SYSTEM
  SHALL escalate with `no_valid_mutant`, and WHEN VALID mutants exist and none
  earned credit with `no_killed_valid_mutant`.
- **SN-118~1**: WHEN a test killed a sentinel mutant THE SYSTEM SHALL escalate
  with `mutation_sensitive_test`, give its kills no credit and cap its
  alignment at UNDETERMINED.
- **SN-119~2**: WHEN a passing test of a covered requirement has an alignment
  proof THE SYSTEM SHALL clear it only if the verdict is ALIGNED at or above
  the `thresholds.alignment` of its confidence source, the isolation reaches the floor, and, unless trust is
  local or the basis is judged, the verdict lists specific matrix evidence;
  any other verdict SHALL escalate with `misaligned_test` and never block.
- **SN-120~1**: WHEN `trust` is local THE SYSTEM SHALL apply the judged pass
  basis, with the same thresholds and isolation floor.
- **SN-121~1**: WHEN a valid completeness proof lists questions THE SYSTEM
  SHALL report each as a `completeness_question` finding at the spec, which
  does not block by default and cannot be required.
- **SN-122~1**: WHEN a finding escalates a judged signal THE SYSTEM SHALL give
  it the severity of `proofs.escalation`, error by default and warning when
  configured, unless `severities` sets one for its category.
- **SN-123~1**: WHEN the manifest lists a pending proof for a required module
  and criterion that lacks a valid proof THE SYSTEM SHALL report the pending
  reason, `awaiting_host`, `awaiting_proof`, `awaiting_host_judgment` or
  `host_unresponsive`, instead of `stale_proof`, and its message SHALL name the
  module and its fulfil skill and say never to answer the requests oneself, or,
  for `host_unresponsive`, to report to the user and not to loop; and WHEN a
  required module has no valid proof for a criterion for which the manifest
  lists a pending proof of a module whose proof it consumes THE SYSTEM SHALL
  report `awaiting_proof` naming that module, instead of `stale_proof`.
- **SN-124~1**: WHEN the manifest cannot be used, a listed proof is malformed,
  or several valid proofs of one kind answer one criterion THE SYSTEM SHALL
  report `malformed_proof` and count what it cannot use as missing.
- **SN-125~1**: WHEN the configuration has a `proofs` block THE SYSTEM SHALL
  record in `extensions.proofs` the trust level, the policy, the manifest and
  its digest, every consumed statement with its digest and state, the pending
  entries and, per criterion, the module standings, the absent modules, the
  channels, the isolation levels, the host exclusions and the screen
  exclusions, and SHALL write the same report bytes for the same inputs; WHEN
  it has none the extension SHALL be absent.
- **SN-126~2**: THE documentation SHALL name exactly the keys of the `proofs`
  block, the manifest keys, the pending reasons, the trust levels, the pass
  bases, the isolation levels, the kinds of confidence source and the
  properties of `extensions.proofs` the code defines.
- **SN-127~1**: WHEN a hook blocks a gate whose only blocking findings are
  pending proofs THE SYSTEM SHALL count the block in a counter of its own, in
  the attempts directory, that lets the hook give up after `MaxHostWaits` = 3
  such blocks, apart from the `MaxAttempts` = 3 of every other block; neither
  kind of block SHALL reset the other counter, a pass or a give-up SHALL reset
  both, and any interleaving of the two kinds SHALL end within `MaxAttempts` +
  `MaxHostWaits` blocks.
- **SN-128~1**: WHEN a hook blocks on pending proofs THE SYSTEM SHALL tell the
  agent which module waits, which fulfil skill to run and never to answer the
  requests itself, and, for `host_unresponsive`, to report to the user and not
  to loop.
- **SN-129~1**: WHEN a response is added that blocks or escalates, a surviving
  mutant, a failing execution or example, a test that kills a sentinel, or an
  alignment proof, THE SYSTEM SHALL never turn an outcome of block into one of
  escalate or pass, nor an outcome of escalate into pass, for any
  configuration.
- **SN-130~1**: WHEN no judged claim of a proof lists executed corroboration
  and nothing executed failed THE SYSTEM SHALL never block the criterion, and
  under a signed pass basis other than judged SHALL never pass it.
- **SN-131~1**: WHEN the rows of the claim-power table are evaluated under each
  trust level and signed pass basis THE SYSTEM SHALL give the outcome and the
  finding categories of the committed gate-outcome goldens.
- **SN-132~1**: WHEN a `code_tree` or `test_source` descriptor of a proof names a
  file that is absent or whose bytes no longer have the recorded SHA-256 THE
  SYSTEM SHALL classify the proof stale and name the file, and WHEN a file that
  no descriptor names changes THE SYSTEM SHALL leave the proof valid.
- **SN-133~2**: WHEN a `code_tree` descriptor names a file outside the
  configured `proofs.code` entries, or a `test_source` descriptor a file outside
  the configured `tests` roots, or either names an excluded path, THE SYSTEM
  SHALL classify the proof stale.
- **SN-134~2**: WHEN a test file holds a test that the source tags bind to the
  criterion a discriminating-power proof is about, or, for an alignment proof,
  the test the proof is about, and the proof lists that file in no
  `test_source` descriptor THE SYSTEM SHALL classify the proof stale and name
  the file.
- **SN-135~2**: WHEN the `spec` descriptor of a proof about a criterion is not
  named `ID~REV` of that criterion or does not carry the current statement
  digest of the criterion THE SYSTEM SHALL classify the proof stale, or, when
  it disagrees with the subject of the proof itself, report it as malformed.
- **SN-136~1**: WHEN an input descriptor of a statement breaks the form of its
  role, THE SYSTEM SHALL report an `input_descriptor` violation for a file
  input whose name is not a relative POSIX path, a `spec` input of a criterion's
  proof that is not named `ID~REV` with the digest of the subject, and a `proof`
  input that is not named by a proof kind or whose `annotations.criterion` is
  absent or not `ID~REV`; THE SYSTEM SHALL accept several `proof` inputs of one
  kind that name different criteria, order inputs by role, name and criterion,
  report two inputs with the same role, name and criterion as unsorted, and
  leave the criterion annotation out of the cache key.
- **SN-137~1**: WHEN a decision of a statement records a `confidenceSource`
  THE SYSTEM SHALL read it and its `confidenceMapping`, and SHALL report an
  unknown value for a source that is not `execution`, `host-votes(k)` with k at
  least 1 or `system-one:` followed by an origin.
- **SN-138~1**: WHEN a judged claim is evaluated THE SYSTEM SHALL apply the
  threshold of the confidence source of the decision recorded on its mutant,
  else on its predicate: the entry of the exact source, else the entry of its
  kind, else the default, SHALL name that threshold in the finding, and SHALL
  report the defaults and the entries by source in `extensions.proofs.policy`.

## Architecture

- **SN-66~1**: THE binary SHALL link no networking package.
- **SN-67~1**: THE domain and the analysis SHALL import neither an adapter nor
  file access.
- **SN-68~1**: THE documentation SHALL let an agent with no other context
  integrate the tool into a pipeline.
  - Non-testable: whether a document is sufficient for a reader is judged by
    that reader; each integration report is reviewed instead.
