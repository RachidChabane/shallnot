---
name: shallnot
description: Requirement-to-test traceability with the `shallnot` CLI. Use in any repository that has a `shallnot.yaml`, whenever you write or change code or tests there, before you report work as done, and whenever a `shallnot` report, hook or CI check shows findings. The user does not need to mention shallnot. Covers how to bind a test to a requirement with a `[verifies ID~REV]` tag, how to run the gate, and how to react to each finding.
---

# Tracing requirements to tests with shallnot

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

## Requirements

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

## Binding a test

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

## Running the gate

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

## Reacting to findings

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

## Prohibitions

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
