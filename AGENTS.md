<!-- shallnot:begin -->
## Tracing requirements to tests with shallnot

`shallnot` is a deterministic gate. It reads the specs, the test sources and
the JUnit XML test results, and fails unless every requirement in focus is
cited by at least one test that ran and passed. You write the tests and the
tags; the tool checks the mapping. It cannot be argued with and it does not
read intent: only requirement IDs, tags and test outcomes.

A repository with a `shallnot.yaml` is gated. Nobody has to ask you to use
shallnot there: the person you work for cares about their feature, and the
gate is part of doing that work properly. Whenever you add or change
behaviour:

1. Find the requirement the work implements in the specs listed in
   `shallnot.yaml`. If no requirement describes the behaviour you were asked
   for, state it first: write the requirement from the request, before the
   code (the `shallnot-plan` skill says how).
2. Bind the tests you write to it with a tag.
3. Run `shallnot gate` before you say the work is done, and resolve what it
   reports.
4. Check that each test you tagged would fail if the behaviour were missing
   (the `shallnot-review` skill says how). The gate only proves a tagged test
   passed.

### Requirements

A requirement is declared in a Markdown or YAML spec as `ID~REVISION`, for
example `CART-2~2` or `ABC-101.AC3~1`:

```markdown
- **ABC-101.AC3~1**: WHEN a customer clicks "Export" THE SYSTEM SHALL download the invoice as a PDF.
```

- The ID is stable. The revision is a positive integer; a declaration without
  one is at revision 1.
- The revision is bumped by whoever owns the spec when the requirement's
  meaning changes. Do not bump it yourself to silence a finding.
- A requirement may be marked non-testable, with a mandatory written
  justification. That decision belongs to the spec's owner.

### Binding a test

Put the tag `[verifies ID~REVISION]` where the test runner reports it. Several
requirements: `[verifies ID~1, OTHER~2]`. The revision is mandatory. Cite the
revision the spec declares now, after checking that the test verifies the
statement as it reads now.

| Ecosystem | Where the tag goes |
|---|---|
| Jest, Vitest | In the test title: `it("rejects expired tokens [verifies AUTH-7~2]", ...)`. In a `describe` title it binds every test inside. |
| JUnit 5 (Java, Kotlin) | In `@DisplayName("rejects expired tokens [verifies AUTH-7~2]")` on the test method. For a parameterized test, in `@ParameterizedTest(name = "{0} is rejected [verifies AUTH-7~2]")`. |
| pytest | `@pytest.mark.verifies("AUTH-7~2")` on the function or class (the project's `conftest.py` copies it into the results). Without that hook: `record_property("verifies", "AUTH-7~2")` as the first line of the test. |
| Go | In a subtest name: `t.Run("rejects expired tokens [verifies AUTH-7~2]", ...)`. |
| Anything else | In the test case name, or a `verifies` property of the `<testcase>` in the JUnit XML. |

A tag in a comment binds nothing: the tag counts only if it appears in the
test results file. One test per behaviour, tagged with the requirement whose
statement its assertions check. A test that verifies no stated requirement
carries no tag; do not invent one.

### Running the gate

1. Run `shallnot gate --json-out shallnot-report.json` from the directory
   holding `shallnot.yaml`. It runs the project's test commands
   (`test_commands` in `shallnot.yaml`), then checks the results they wrote.
   When the config names no test command, run the project's tests so that they
   write JUnit XML, then run `shallnot check --json-out shallnot-report.json`.
2. Read the exit code first:
   - `0`: clean. Nothing to do.
   - `1`: blocked. Read the report.
   - `2`: no verdict (a results file is missing or was not rewritten by the
     test run, the config is wrong). This says nothing about your code. Read
     the message on standard error and fix the cause: usually the tests did
     not run to the point of writing their report. If the project has no spec
     at all, stop and tell the user; requirements come from a request for
     behaviour, never from the need to satisfy the gate.
3. In `shallnot-report.json`, read `findings` where `blocking` is `true`. Each
   has a `category`, a `message`, a `location` (`file`, `line`) and usually a
   `requirement_id`. `requirements[]` holds the full matrix: each requirement's
   `coverage` and its `tests` with their `outcome`.

Some projects install an end-of-turn hook that runs the gate for you and hands
you the blocking findings. Treat that message exactly like a blocked gate.

### Reacting to findings

| Category | Meaning | What to do |
|---|---|---|
| `uncovered_requirement` | No test cites the requirement. | Write a test whose assertions check the requirement's statement, and tag it. |
| `failed_requirement` | Bound tests ran; none passed. | Fix the implementation until the test passes. If the test itself is wrong about the requirement, fix the test to match the statement. |
| `failing_bound_test` | A bound test fails although another passes. | Same as above: fix the code. |
| `skipped_requirement` | Its bound tests were all skipped. | Un-skip the test and make it pass. |
| `not_run_requirement`, `tag_not_in_results` | The tag is in the source but in no results file. | Make sure the test is collected and run, that its results file is passed to `shallnot`, and that the tag sits where the runner reports it (see the table above). |
| `orphan_tag` | The tag cites an ID no spec declares. | Fix a typo in the ID. If the requirement does not exist, remove the claim: the test verifies nothing stated. A requirement is written from a request for behaviour, before the code; never invent one to give a tag something to cite. |
| `revision_mismatch` | The tag cites another revision than the spec. | Re-read the requirement's current statement, update the test so that it verifies that statement, then cite the current revision. |
| `malformed_tag` | The tag cannot be read. | Write it as `[verifies ID~REVISION]`. |
| `unjustified_non_testable`, `duplicate_id`, `malformed_requirement` | The spec is defective. | Report it to the spec's owner, or fix the spec if you own it. |
| `bound_non_testable` | A test cites a requirement marked non-testable. | Report it: either the requirement is testable after all, or the tag is wrong. |
| `untagged_test` | A test cites no requirement. | Information. Tag it if it verifies a stated requirement; otherwise leave it. |

### Prohibitions

These make the report green without making the software correct. The gate
exists to prevent exactly that, and each one destroys the evidence that a
reviewer or a later run relies on.

- **Never delete or alter a tag to make a finding disappear.** A tag is a claim
  that a test verifies a requirement. Removing the claim does not make the
  requirement verified; it hides that it is not.
- **Never mark a requirement non-testable to avoid writing a test.** That
  status is a decision about the requirement, made by the spec's owner with a
  written justification, and it removes the requirement from verification
  permanently.
- **Never weaken, remove or bypass an assertion to make a bound test pass.** A
  passing test that no longer checks the statement makes the report lie: the
  requirement is reported covered while nothing verifies it.
- **Never skip, exclude or delete a failing bound test**, and never edit a
  results file or the spec's revision to change the verdict.
- **Never tag a test with a requirement it does not check.** A tag on an
  unrelated passing test is a false claim of verification.

If a requirement cannot be met or tested as written, stop and say so, with the
finding and the reason. That is a correct outcome; a falsely green report is
not.

## Writing requirements for shallnot

In a repository with a `shallnot.yaml`, behaviour is stated before it is
built. A requirement is the sentence a test will later be held against, and
the gate refuses work whose requirements have no passing test. When someone
asks for a feature, they are giving you the content of one or more
requirements: write them down first, then implement against them.

### Where and how

1. Read `shallnot.yaml`. `specs` lists the spec files and directories;
   `id_pattern`, when set, is the shape every ID must match. Read the existing
   specs to learn the project's conventions: file per feature or per ticket,
   ID prefix, numbering, Markdown or YAML.
2. If a requirement already describes the behaviour asked for, do not write a
   new one. Implement against the existing one.
3. Otherwise add requirements to the spec the work belongs to, or a new spec
   file beside the others. Take the next free ID in the project's numbering.
   Never reuse or renumber an ID: tests and history cite it.

Markdown form, one requirement per list item, revision after `~`:

```markdown
- **PWD-2~1**: WHEN a password contains the account's username THE SYSTEM
  SHALL reject it.
```

YAML form:

```yaml
requirements:
  - id: PWD-2
    revision: 1
    statement: WHEN a password contains the account's username THE SYSTEM SHALL reject it.
```

### What makes a requirement good

- **One behaviour each.** If the sentence needs "and" between two outcomes,
  it is two requirements. A test must be able to fail for exactly one reason.
- **Observable.** State a trigger and an outcome someone could watch happen:
  "WHEN <trigger> THE SYSTEM SHALL <outcome>". Not how the code is organised,
  not which function does it.
- **Decidable.** Name the boundary: "shorter than 12 characters", not "too
  short". If the request is vague on a boundary, pick the reading the user
  most plausibly means, state it in the requirement, and say in your final
  message that you chose it.
- **The user's intent, not your implementation.** Write what was asked for.
  Do not add requirements for behaviour nobody requested, and do not leave
  out a part of the request because it is hard to test.
- **Cover the refusals too.** When the request implies an error case ("reject",
  "only if", "at most"), the rejection is a requirement of its own or an
  explicit part of the statement.

### Changing a requirement

- A change of **meaning** (different trigger, outcome or boundary) keeps the
  ID and increments the revision: `PWD-1~1` becomes `PWD-1~2`. Every tag that
  cites the old revision then fails the gate until its test has been checked
  against the new statement and cites the new revision. That is the point:
  do it, do not work around it.
- A change of **wording** that leaves the meaning intact (typo, clearer
  phrasing) keeps the revision.
- A requirement that no longer applies is deleted, together with the tags
  that cite it, in the same change, and you say so.

### Non-testable requirements

A requirement no automated test can verify ("the form shall feel welcoming")
is marked non-testable with a written justification saying how it is judged
instead:

```markdown
- **PWD-3~1**: THE password form SHALL feel welcoming.
  - Non-testable: judged in moderated usability sessions.
```

Use this only for requirements that are non-testable by nature. "Hard to
test" is not a justification.

### What you must never do

- **Never write or edit a requirement to make a gate finding go away.** A
  requirement comes from what someone asked for, before the code. Rewriting it
  to match what the code happens to do, weakening it until a failing test
  passes, or inventing one to give an orphan tag something to cite, turns the
  spec into a description of the code and leaves nothing to verify against.
- **Never bump a revision without a change of meaning**, and never change the
  meaning without bumping it.
- **Never mark a requirement non-testable to avoid writing a test.**

### Before you implement

Run `shallnot check` (or `shallnot gate`): a malformed declaration or a
duplicate ID is reported as `malformed_requirement` or `duplicate_id`, and the
new requirements appear as `uncovered_requirement`, which is what the tests
you write next will resolve. In your final message, list the requirements you
added or changed, so that the person who asked can see their request as it
was written down.

## Reviewing tests against their requirements

`shallnot` guarantees the mapping: every requirement in focus is cited by a
test that ran and passed. It cannot tell whether that test checks what the
requirement says. A tag on a test that asserts nothing makes the gate green
and the requirement unverified. This review is the half the machine cannot
do, and it is only worth doing requirement by requirement.

### Procedure

1. Get the matrix. Run `shallnot gate --json-out shallnot-report.json` (or
   `shallnot check --json-out shallnot-report.json` when the tests already
   ran). A blocked gate is reported first; the review below is for what the
   gate accepts.
2. Decide the scope: the requirements added or changed by the change under
   review, and the requirements cited by tests the change touches. With git,
   `git diff <base>...HEAD` on the specs and the test directories gives both.
3. For each requirement in scope, read its `statement` in the report, then
   open each bound test at `requirements[].tests[].source` (`file`, `line`)
   and read the test body and what it calls.
4. Judge the requirement with the questions below and give it one verdict.

### Questions for each requirement

- **Trigger.** Does the test put the system in the situation the statement
  names? A requirement about passwords *containing the username* is not
  exercised by a test that only tries short passwords.
- **Outcome.** Does an assertion fail if the outcome does not happen? Look
  for tests that only check that nothing was raised, assert on a mock they
  configured themselves, assert a constant, or catch the failure.
- **Boundary.** When the statement names a limit ("shorter than 12",
  "above 99"), is the limit itself tested on both sides, or only a value far
  from it?
- **Refusal.** When the statement says reject, deny or fail, does a test show
  the rejection, and another that the legitimate case still passes?
- **Isolation.** Does the test reach the behaviour, or is the code that
  implements the requirement replaced by a stub on the way?
- **One claim per tag.** A test citing several requirements must verify each
  of them. A tag added to a broad test "because it touches that code" is a
  false claim.

### Signs of a gamed gate in a diff

Treat each of these as a finding to explain, not as proof of bad faith:

- a removed or altered `verifies` tag, or a test deleted with its tag;
- a requirement newly marked non-testable, or a justification that says
  "hard to test";
- a requirement reworded or a revision bumped in the same change that makes
  its test pass, without a change of intent behind it;
- an assertion loosened, a test newly skipped, an expected value edited to
  match the actual one;
- a severity lowered or a category switched off in `shallnot.yaml`, `advisory`
  turned on, a spec removed from `specs` or a path added to `exclude`;
- new tests that cite no requirement, or new behaviour that no requirement
  describes: work outside the spec.

### Verdicts and report

Give each requirement in scope one verdict:

- **verified**: the bound tests exercise the trigger and would fail if the
  outcome did not happen.
- **weak**: the behaviour is exercised but something in the statement is not
  checked (a boundary, the refusal, one of two outcomes). Say what is missing.
- **not verified**: the tag is a claim the test does not support.

Report as a list: requirement reference, verdict, the test location, one or
two sentences of evidence quoting the assertion or its absence. Put the
`not verified` and `weak` ones first. Add the diff findings from the section
above. Do not pad the report with the verified ones beyond a count.

When you are reviewing your own work, fix what you find: strengthen the test
so that it would fail without the behaviour, then run the gate again. When
you are reviewing someone else's, report and do not rewrite their tests
unless asked. Never resolve a finding by removing the tag, weakening the
requirement or marking it non-testable.
<!-- shallnot:end -->
