---
name: shallnot-review
description: Review whether tests really verify their requirements in a repository gated by shallnot (it has a `shallnot.yaml`). Use when asked to review a change, a branch or a pull request there, when reviewing your own work before reporting it done, and when asked whether a requirement is actually covered. The gate proves a tagged test passed; this review judges whether that test checks what the requirement says.
---

# Reviewing tests against their requirements

`shallnot` guarantees the mapping: every requirement in focus is cited by a
test that ran and passed. It cannot tell whether that test checks what the
requirement says. A tag on a test that asserts nothing makes the gate green
and the requirement unverified. This review is the half the machine cannot
do, and it is only worth doing requirement by requirement.

## Procedure

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

## Questions for each requirement

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

## Signs of a gamed gate in a diff

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

## Verdicts and report

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
