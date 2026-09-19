---
name: shallnot-plan
description: Write or change requirements in a repository gated by shallnot (it has a `shallnot.yaml`). Use before implementing any feature, fix or behaviour change that no existing requirement describes, when asked to plan, specify or break down work, and when the meaning of an existing requirement changes. The user does not need to ask for requirements; a request for behaviour is a request to state it first.
---

# Writing requirements for shallnot

In a repository with a `shallnot.yaml`, behaviour is stated before it is
built. A requirement is the sentence a test will later be held against, and
the gate refuses work whose requirements have no passing test. When someone
asks for a feature, they are giving you the content of one or more
requirements: write them down first, then implement against them.

## Where and how

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

## What makes a requirement good

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

## Changing a requirement

- A change of **meaning** (different trigger, outcome or boundary) keeps the
  ID and increments the revision: `PWD-1~1` becomes `PWD-1~2`. Every tag that
  cites the old revision then fails the gate until its test has been checked
  against the new statement and cites the new revision. That is the point:
  do it, do not work around it.
- A change of **wording** that leaves the meaning intact (typo, clearer
  phrasing) keeps the revision.
- A requirement that no longer applies is deleted, together with the tags
  that cite it, in the same change, and you say so.

## Non-testable requirements

A requirement no automated test can verify ("the form shall feel welcoming")
is marked non-testable with a written justification saying how it is judged
instead:

```markdown
- **PWD-3~1**: THE password form SHALL feel welcoming.
  - Non-testable: judged in moderated usability sessions.
```

Use this only for requirements that are non-testable by nature. "Hard to
test" is not a justification.

## What you must never do

- **Never write or edit a requirement to make a gate finding go away.** A
  requirement comes from what someone asked for, before the code. Rewriting it
  to match what the code happens to do, weakening it until a failing test
  passes, or inventing one to give an orphan tag something to cite, turns the
  spec into a description of the code and leaves nothing to verify against.
- **Never bump a revision without a change of meaning**, and never change the
  meaning without bumping it.
- **Never mark a requirement non-testable to avoid writing a test.**

## Before you implement

Run `shallnot check` (or `shallnot gate`): a malformed declaration or a
duplicate ID is reported as `malformed_requirement` or `duplicate_id`, and the
new requirements appear as `uncovered_requirement`, which is what the tests
you write next will resolve. In your final message, list the requirements you
added or changed, so that the person who asked can see their request as it
was written down.
