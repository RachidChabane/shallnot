# Spec format

A spec is a file that declares requirements. `shallnot check` reads specs
given by `--specs` and `--focus` (or the `specs` / `focus` keys of
[shallnot.yaml](configuration.md)), in Markdown or YAML, and joins them with
test results and source tags to produce the report documented in
[report.md](report.md). This document specifies the requirement model and
both file formats.

## Requirement model

A requirement has:

| Field           | Meaning                                                                 | Default              |
|-----------------|--------------------------------------------------------------------------|-----------------------|
| `id`            | Stable identifier, matching [`id_pattern`](#id-pattern).                | required              |
| `revision`      | Positive integer. Incremented whenever the requirement's *meaning* changes — a wording fix that does not change what must be true is not a bump. | `1` |
| `title`         | Short title.                                                            | empty                 |
| `statement`     | What the system shall do.                                               | required (non-empty)  |
| `status`        | `active` or `non_testable`.                                             | `active`              |
| `justification` | Why no test can verify the requirement.                                 | empty                 |

A test cites a requirement with a **reference**, `ID~REVISION` (for example
`CART-2~2`, `ABC-101.AC3~1`). A reference always names a revision; there is
no way to cite "the current revision" implicitly.

Bumping the revision in the spec invalidates every reference that still
cites the old revision: `shallnot check` reports those tags as
`revision_mismatch`, which forces re-verification of the tests that cite
them (see [binding.md](binding.md#tag-grammar)) before the requirement can
count as covered again under its new revision. This is the mechanism by
which spec drift is caught: a requirement's statement cannot change silently
underneath tests that were written against an earlier meaning.

A requirement marked `non_testable` is excluded from coverage findings, but
it still needs a `justification`: a `non_testable` requirement without one
is reported as `unjustified_non_testable`. A `non_testable` requirement that
a test nonetheless cites is reported as `bound_non_testable` — its
justification claims no test can verify it, so a citing test is itself a
defect to resolve, either by removing the tag or by reconsidering the status.

Requirement statements should follow the EARS pattern
(`WHEN <trigger> THE SYSTEM SHALL <behaviour>`, or the plain form
`THE SYSTEM SHALL <behaviour>` for unconditional requirements). This
phrasing is a recommendation for writing statements that are unambiguous and
individually testable; `shallnot check` does not enforce it.

### ID pattern

`id_pattern` is a RE2 regular expression that a requirement ID must match in
full (it is anchored on both ends). The default is:

```
[A-Z][A-Z0-9]*-[0-9]+(?:\.[A-Za-z0-9]+)*
```

which matches IDs such as `REQ-42` and `ABC-101.AC3`. A custom pattern must
not match any of `~`, `,`, `]`, or whitespace — these are reserved by the
reference and tag grammar (see [binding.md](binding.md#tag-grammar)); a
pattern that matches one of them is rejected at startup.

A heading such as `# ABC-101: Export invoices` is a declaration whenever
`ABC-101` matches `id_pattern` — under the default pattern it does, so a
document titled with what looks like a ticket key is read as a heading-form
declaration for that ID, with "Export invoices" taken as its title. Projects
whose documents are titled this way, without meaning to declare a
requirement, need an `id_pattern` narrow enough to match only real
requirement IDs, for example `[A-Z]+-[0-9]+\.AC[0-9]+` (matching IDs like
`ABC-101.AC3` but not `ABC-101`), as configured in
[`fixtures/pipeline/shallnot.yaml`](../fixtures/pipeline/shallnot.yaml).

### Duplicate IDs

Requirements are indexed by ID across every spec file given to a run. The
first declaration, in file path then line order, is kept; every later
declaration of the same ID — in the same file or another one — is reported
as `duplicate_id` and does not replace the first.

## Markdown spec

A Markdown spec is prose. Requirements are embedded as **declarations**: a
line that, after stripping

1. an optional heading marker (`#` through `######`),
2. an optional list marker (`-`, `*`, `+`, or `1.`/`1)`), with an optional
   `[ ]`/`[x]`/`[X]` checkbox,
3. optional emphasis (`**`, `__`, or a backtick),

starts with `ID` or `ID~REVISION`, then a colon. Everything after the colon
(with the same optional trailing emphasis marker stripped) is the start of
the declaration's content. An ID mentioned in the middle of a sentence is
never a declaration — the pattern must be anchored at the start of the line
after the optional prefixes above.

A declaration takes one of two forms:

- **List or paragraph form**: the declaration is not on a heading line. The
  requirement's `statement` is the rest of the line after the colon, plus
  every following non-blank line up to the first blank line (lines are
  joined with a single space). There is no title.
- **Heading form**: the declaration is on a heading line (`#` through
  `######`). The heading text after the colon becomes the `title`. The
  `statement` is the first paragraph that follows — the run of non-blank
  lines up to the first blank line, starting after the heading.

A block — the declaration and everything absorbed into its statement and
justification — ends at the next declaration or at any heading line
(including one that is not itself a declaration). Whichever comes first
closes the current block.

Inside an open block, a line matching (case-insensitively, optionally as a
sub-bullet, optionally wrapped in `**`/`__`)

```
Non-testable: <justification>
```

sets `status` to `non_testable` and starts the `justification`, which
absorbs every following non-blank line up to the first blank line, the same
way a statement does. The colon after `Non-testable` is optional; `-`, `_`
and ` ` are all accepted between "non" and "testable".

Fenced code blocks (opened and closed by a line starting with ` ``` ` or
`~~~`, ignoring leading whitespace) are skipped entirely: nothing inside one
is read as a declaration, statement, or justification line.

A declaration with no revision defaults to revision `1`. A declaration whose
statement (and title, for the heading form) is empty, or whose revision text
is not a positive integer written in plain decimal, is not added as a
requirement: it is reported as a `malformed_requirement` problem instead, at
the declaration's line.

### Markdown example

`fixtures/cart/spec/cart.md`:

```markdown
# Cart pricing

The cart service computes what a customer pays at checkout. Prices are
decimal amounts; quantities are whole numbers.

## Totals

- **CART-1~1**: WHEN a cart holds line items THE SYSTEM SHALL compute the
  total as the sum of each line item's price times its quantity.

- **CART-4~1**: WHEN a total has more than two decimals THE SYSTEM SHALL round
  it to two decimals, half-up.

## Discounts

### CART-2~2: Percentage discount codes

WHEN a customer applies a percentage discount code THE SYSTEM SHALL reduce the
total by that percentage.

Revision 2 replaced flat-amount codes with percentage codes.

## Checkout

- **CART-3~1**: WHEN a customer checks out an empty cart THE SYSTEM SHALL
  reject the checkout with an error.
- **CART-5~1**: WHEN a line item quantity exceeds 99 THE SYSTEM SHALL reject
  the line item.

## Presentation

- **CART-6~1**: WHEN a total is displayed THE SYSTEM SHALL show it in the
  customer's currency.
- **CART-7~1**: THE checkout page SHALL feel uncluttered on a phone screen.
  - Non-testable: judged in the quarterly design review with customer panels;
    no automated check can stand in for it.
```

`CART-2~2` is a heading-form declaration: title "Percentage discount codes",
statement the paragraph that follows ("Revision 2 replaced..." is a second
paragraph, separated by a blank line, and is not part of the statement).
`CART-1~1`, `CART-4~1`, `CART-3~1`, `CART-5~1` and `CART-6~1` are list-form
declarations. `CART-7~1` is non-testable, with the justification taken from
the `Non-testable:` sub-bullet.

The tests at `fixtures/cart/pytest` bind these requirements with
`@pytest.mark.verifies` tags (see [binding.md](binding.md#pytest)); the real
results are at `fixtures/cart/pytest/results/junit.xml`.

Commands run from the repository root need `--no-config`: the repository
root holds shallnot's own `shallnot.yaml`, which would otherwise apply (see
[configuration.md](configuration.md)). Build first with `script/build`, then
run:

```
$ bin/shallnot check --no-config --specs fixtures/cart/spec/cart.md --tests fixtures/cart/pytest --results fixtures/cart/pytest/results/junit.xml
```

Output:

```
shallnot: FAIL
focus: every known requirement
requirements: 7 known, 7 in focus (2 covered, 1 failed, 1 skipped, 1 not run, 1 uncovered, 1 non-testable)
tests: 16 in results, 13 bound, 2 untagged
findings: 7 error, 1 warning, 2 info (7 blocking)

REQUIREMENTS
  CART-1~1  covered  fixtures/cart/spec/cart.md:8
      passed  tests.test_misc › test_discounted_total_reflects_line_items  fixtures/cart/pytest/tests/test_misc.py:26
      passed  tests.test_totals › test_total_reflects_each_line_item       fixtures/cart/pytest/tests/test_totals.py:15
      passed  tests.test_totals › test_total_sums_price_times_quantity     fixtures/cart/pytest/tests/test_totals.py:6
  CART-4~1  skipped  fixtures/cart/spec/cart.md:11
      skipped  tests.test_rounding › test_total_rounds_half_up_to_two_decimals  fixtures/cart/pytest/tests/test_rounding.py:7
  CART-2~2  covered  fixtures/cart/spec/cart.md:16
      passed  tests.test_discounts › test_discount_code_reduces_total[10.00-2-10-18.0]        fixtures/cart/pytest/tests/test_discounts.py:6
      passed  tests.test_discounts › test_discount_code_reduces_total[15.00-4-5-57.0]         fixtures/cart/pytest/tests/test_discounts.py:6
      passed  tests.test_discounts › test_discount_code_reduces_total[50.00-1-20-40.0]        fixtures/cart/pytest/tests/test_discounts.py:6
      passed  tests.test_discounts.TestDiscounts › test_full_discount_zeroes_total            fixtures/cart/pytest/tests/test_discounts.py:30
      passed  tests.test_discounts.TestDiscounts › test_partial_discount_on_multiple_items    fixtures/cart/pytest/tests/test_discounts.py:30
      passed  tests.test_discounts.TestDiscounts › test_zero_discount_leaves_total_unchanged  fixtures/cart/pytest/tests/test_discounts.py:30
      passed  tests.test_misc › test_discounted_total_reflects_line_items                     fixtures/cart/pytest/tests/test_misc.py:26
  CART-3~1  failed  fixtures/cart/spec/cart.md:25
      failed  tests.test_checkout › test_checkout_rejects_empty_cart  fixtures/cart/pytest/tests/test_checkout.py:6
  CART-5~1  not_run  fixtures/cart/spec/cart.md:27
      not_run  test_quantity_above_limit_is_rejected  fixtures/cart/pytest/tests/slow/test_quantity_limits.py:6
  CART-6~1  uncovered  fixtures/cart/spec/cart.md:32
  CART-7~1  non_testable  fixtures/cart/spec/cart.md:34

FINDINGS
  info     untagged_test          fixtures/cart/pytest/results/junit.xml                     test "tests.test_misc › test_cart_starts_with_no_discount" verifies no stated requirement
  info     untagged_test          fixtures/cart/pytest/results/junit.xml                     test "tests.test_totals › test_total_with_no_discount_matches_subtotal" verifies no stated requirement
  warning  tag_not_in_results     fixtures/cart/pytest/tests/slow/test_quantity_limits.py:6  tag citing CART-5~1 appears in no results file: the test did not run, or the tag sits where the runner does not report it
  error    revision_mismatch      fixtures/cart/pytest/tests/test_discounts.py:22            tag cites CART-2~1, but the spec declares CART-2~2: re-verify the test against the current statement, then cite CART-2~2
  error    orphan_tag             fixtures/cart/pytest/tests/test_misc.py:6                  tag cites CART-99~1, but no known spec declares CART-99
  error    malformed_tag          fixtures/cart/pytest/tests/test_misc.py:13                 malformed tag pytest.mark.verifies("CART-1"): "CART-1" cites no revision (expected ID~REVISION)
  error    skipped_requirement    fixtures/cart/spec/cart.md:11                              requirement CART-4~1 is not verified: its bound tests were skipped
  error    failed_requirement     fixtures/cart/spec/cart.md:25                              requirement CART-3~1 is not verified: no bound test passed and at least one failed
  error    not_run_requirement    fixtures/cart/spec/cart.md:27                              requirement CART-5~1 is not verified: its tagged tests appear in no results file (they did not run, or the tag sits where the runner does not report it)
  error    uncovered_requirement  fixtures/cart/spec/cart.md:32                              requirement CART-6~1 has no bound test
```

Exit code 1 (blocked): `CART-3~1` failed, `CART-4~1`'s test was skipped,
`CART-5~1`'s tagged test never reached the results file, `CART-6~1` has no
bound test at all, and three tag defects (a stale revision, a tag citing an
unknown requirement, and a malformed tag) each raise an error finding.

## YAML spec

A YAML spec is a mapping with one key, `requirements`, a list of mappings.
Each item accepts these keys:

| Key             | Type    | Default   |
|-----------------|---------|-----------|
| `id`            | string, matching `id_pattern` | required |
| `revision`      | positive integer               | `1`      |
| `title`         | string                          | empty    |
| `statement`     | non-empty string                | required |
| `status`        | `active` or `non_testable`      | `active` |
| `justification` | string                           | empty    |

The full schema is [`schemas/spec.schema.json`](../schemas/spec.schema.json)
(also printed by `shallnot schema spec`).

An item with an unknown key, a non-scalar value for a known key, an ID that
does not match `id_pattern`, a `revision` that is not a positive integer, an
unrecognised `status`, or a missing `id`/`statement`, is not added as a
requirement: it is reported as a `malformed_requirement` problem at that
item's line, and the run continues with the rest of the file.

A document that fails to parse as YAML, or that parses but is not a mapping
with a `requirements` list (for example a bare list, or a mapping missing
that key), is a tool failure: `shallnot check` exits 2 and produces no
report.

### YAML example

`fixtures/cart/spec/cart.yaml`:

```yaml
requirements:
  - id: CART-1
    revision: 1
    statement: >
      WHEN a cart holds line items THE SYSTEM SHALL compute the total as the
      sum of each line item's price times its quantity.
  - id: CART-4
    revision: 1
    statement: >
      WHEN a total has more than two decimals THE SYSTEM SHALL round it to two
      decimals, half-up.
  - id: CART-2
    revision: 2
    title: Percentage discount codes
    statement: >
      WHEN a customer applies a percentage discount code THE SYSTEM SHALL
      reduce the total by that percentage.
  - id: CART-3
    revision: 1
    statement: >
      WHEN a customer checks out an empty cart THE SYSTEM SHALL reject the
      checkout with an error.
  - id: CART-5
    revision: 1
    statement: >
      WHEN a line item quantity exceeds 99 THE SYSTEM SHALL reject the line
      item.
  - id: CART-6
    revision: 1
    statement: >
      WHEN a total is displayed THE SYSTEM SHALL show it in the customer's
      currency.
  - id: CART-7
    revision: 1
    statement: THE checkout page SHALL feel uncluttered on a phone screen.
    status: non_testable
    justification: >
      judged in the quarterly design review with customer panels; no automated
      check can stand in for it.
```

This declares the same seven requirements as `fixtures/cart/spec/cart.md`
above. Commands run from the repository root need `--no-config`, for the
same reason as the Markdown example: the repository root holds shallnot's
own `shallnot.yaml`. Build first with `script/build`, then run against the
same tests and results:

```
$ bin/shallnot check --no-config --specs fixtures/cart/spec/cart.yaml --tests fixtures/cart/pytest --results fixtures/cart/pytest/results/junit.xml
```

Output:

```
shallnot: FAIL
focus: every known requirement
requirements: 7 known, 7 in focus (2 covered, 1 failed, 1 skipped, 1 not run, 1 uncovered, 1 non-testable)
tests: 16 in results, 13 bound, 2 untagged
findings: 7 error, 1 warning, 2 info (7 blocking)

REQUIREMENTS
  CART-1~1  covered  fixtures/cart/spec/cart.yaml:2
      passed  tests.test_misc › test_discounted_total_reflects_line_items  fixtures/cart/pytest/tests/test_misc.py:26
      passed  tests.test_totals › test_total_reflects_each_line_item       fixtures/cart/pytest/tests/test_totals.py:15
      passed  tests.test_totals › test_total_sums_price_times_quantity     fixtures/cart/pytest/tests/test_totals.py:6
  CART-4~1  skipped  fixtures/cart/spec/cart.yaml:7
      skipped  tests.test_rounding › test_total_rounds_half_up_to_two_decimals  fixtures/cart/pytest/tests/test_rounding.py:7
  CART-2~2  covered  fixtures/cart/spec/cart.yaml:12
      passed  tests.test_discounts › test_discount_code_reduces_total[10.00-2-10-18.0]        fixtures/cart/pytest/tests/test_discounts.py:6
      passed  tests.test_discounts › test_discount_code_reduces_total[15.00-4-5-57.0]         fixtures/cart/pytest/tests/test_discounts.py:6
      passed  tests.test_discounts › test_discount_code_reduces_total[50.00-1-20-40.0]        fixtures/cart/pytest/tests/test_discounts.py:6
      passed  tests.test_discounts.TestDiscounts › test_full_discount_zeroes_total            fixtures/cart/pytest/tests/test_discounts.py:30
      passed  tests.test_discounts.TestDiscounts › test_partial_discount_on_multiple_items    fixtures/cart/pytest/tests/test_discounts.py:30
      passed  tests.test_discounts.TestDiscounts › test_zero_discount_leaves_total_unchanged  fixtures/cart/pytest/tests/test_discounts.py:30
      passed  tests.test_misc › test_discounted_total_reflects_line_items                     fixtures/cart/pytest/tests/test_misc.py:26
  CART-3~1  failed  fixtures/cart/spec/cart.yaml:18
      failed  tests.test_checkout › test_checkout_rejects_empty_cart  fixtures/cart/pytest/tests/test_checkout.py:6
  CART-5~1  not_run  fixtures/cart/spec/cart.yaml:23
      not_run  test_quantity_above_limit_is_rejected  fixtures/cart/pytest/tests/slow/test_quantity_limits.py:6
  CART-6~1  uncovered  fixtures/cart/spec/cart.yaml:28
  CART-7~1  non_testable  fixtures/cart/spec/cart.yaml:33

FINDINGS
  info     untagged_test          fixtures/cart/pytest/results/junit.xml                     test "tests.test_misc › test_cart_starts_with_no_discount" verifies no stated requirement
  info     untagged_test          fixtures/cart/pytest/results/junit.xml                     test "tests.test_totals › test_total_with_no_discount_matches_subtotal" verifies no stated requirement
  warning  tag_not_in_results     fixtures/cart/pytest/tests/slow/test_quantity_limits.py:6  tag citing CART-5~1 appears in no results file: the test did not run, or the tag sits where the runner does not report it
  error    revision_mismatch      fixtures/cart/pytest/tests/test_discounts.py:22            tag cites CART-2~1, but the spec declares CART-2~2: re-verify the test against the current statement, then cite CART-2~2
  error    orphan_tag             fixtures/cart/pytest/tests/test_misc.py:6                  tag cites CART-99~1, but no known spec declares CART-99
  error    malformed_tag          fixtures/cart/pytest/tests/test_misc.py:13                 malformed tag pytest.mark.verifies("CART-1"): "CART-1" cites no revision (expected ID~REVISION)
  error    skipped_requirement    fixtures/cart/spec/cart.yaml:7                             requirement CART-4~1 is not verified: its bound tests were skipped
  error    failed_requirement     fixtures/cart/spec/cart.yaml:18                            requirement CART-3~1 is not verified: no bound test passed and at least one failed
  error    not_run_requirement    fixtures/cart/spec/cart.yaml:23                            requirement CART-5~1 is not verified: its tagged tests appear in no results file (they did not run, or the tag sits where the runner does not report it)
  error    uncovered_requirement  fixtures/cart/spec/cart.yaml:28                            requirement CART-6~1 has no bound test
```

The Markdown and YAML specs declare the same requirements, so the two runs
produce the same coverage matrix and the same findings, differing only in
the spec file path and line numbers.

## Spec paths

`--specs` (known specs) and `--focus` (specs a run demands coverage for) are
repeatable and each accept:

- a single file, read as one spec;
- a directory, walked recursively for every file whose extension is `.md`,
  `.markdown`, `.yaml` or `.yml`;
- a glob, including `**` for recursive matching (for example
  `specs/**/*.yaml`).

A pattern that matches no file is a tool failure (exit 2), never a silent
empty result. Every file matched by `--focus` is also implicitly known,
whether or not it is also matched by `--specs`; conversely, a spec matched
only by `--specs` is known but its requirements are only in focus if no
`--focus`/`--focus-id` is given at all, or if it is separately matched by
one of them. See [configuration.md](configuration.md) for how `--specs`,
`--focus` and `--focus-id` combine, and [report.md](report.md) for how
focus affects which findings a run raises.

## See also

- [binding.md](binding.md) — how tests cite requirements, per test runner.
- [report.md](report.md) — finding categories, severities and report formats.
- [configuration.md](configuration.md) — flags and `shallnot.yaml`.
