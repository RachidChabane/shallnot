# Binding

A test **binds** to a requirement by citing its [reference](spec-format.md#requirement-model)
(`ID~REVISION`) in a **tag**, placed somewhere a test runner reports it in
JUnit XML: the `<testcase>` element's `name`, its `classname`, or a
`<property name="verifies" value="...">` inside it. `shallnot check` reads
`--results` files for these citations and matches them against the
requirements known from `--specs`/`--focus` (see [spec-format.md](spec-format.md)).

Independently, `shallnot check` scans the directories given by `--tests` for
the same tags written in source. The source scan never decides coverage by
itself — a citation only counts once it is read back from a results file —
but it gives every finding a file:line location, and it is how a tagged test
that never reached any results file is detected (`tag_not_in_results`,
turning the requirement's coverage state `not_run`). A tag that sits only
inside a comment is never read by the source scan and never binds.

Presence of a tag on a test is necessary but not sufficient evidence that
the test verifies the requirement: `shallnot check` confirms that a test
citing the reference ran and what it reported, not that its assertions are
adequate. That judgment stays with whoever writes and reviews the test.

## Tag grammar

A tag is written `[verifies ID~REV]` or, for several references,
`[verifies ID~REV, ID~REV, ...]`. Every reference in the list must state a
revision; `[verifies ID]` is a `malformed_tag` ("cites no revision"), not a
citation of the current revision. An ID that does not match
[`id_pattern`](spec-format.md#id-pattern), or a revision that is not a
positive integer, is likewise `malformed_tag`. A tag naming no reference at
all (`[verifies]`, `[verifies ,]`) is `malformed_tag` ("cites no
requirement"). Whitespace and underscores around each reference and around
the separating comma are trimmed, so `[verifies_ID~1]` (as `go test`
rewrites a subtest name containing spaces) and `[verifies ID~1, ID2~1]` both
parse.

A reference the tag cites that names an ID no known spec declares is
`orphan_tag`. A reference that names a known ID but the wrong revision is
`revision_mismatch`. Both are reported once at the tag's location (source
line when the source scan found it, otherwise the results file), never
again for the same reference seen a second time at the same defect.

## How the source scan labels and matches tags

Every source tag is recorded with a **label**: the text the scan judged to
identify the test (or suite) it decorates. How the label is derived depends
on which extractor found the tag:

- The bracket extractor (`[verifies ...]`, used for every language) looks at
  the enclosing quoted string literal on the tag's line, if any. Within that
  literal, it identifies the *static* parts — the text outside of any
  runner placeholder (`%d`/`%s`/... printf verbs, `${...}`, `$name`,
  `{0}`/`{index}` templates) — and joins them (trimming the placeholders
  themselves) into the label. A tag not inside a quoted string on its line
  gets an empty label.
- The pytest extractor labels a `pytest.mark.verifies(...)` with the name of
  the function or class it decorates (found by scanning forward), and a
  module-level `pytestmark = pytest.mark.verifies(...)` with the module's
  file name. It labels `record_property("verifies", ...)` with the name of
  the nearest preceding function or class definition.

A results citation is matched back to the source tags that produced it by:
the tag must cite the same reference, and the tag's label — after
normalizing whitespace — must occur as a substring of the test's *identity*,
the space-joined `suite + classname + name` from the results file. Because a
dynamic title (built from a placeholder, or from a title vitest/Jest
constructs from a test's parameters) is not matched by its runtime value,
only by the label's static remainder, this is a match **by reference**, not
by the exact rendered text: several distinct results test cases (one per
parametrised invocation) can each satisfy the same source tag.

More than one source tag can match the same result (typically: a tag on the
test itself, and a tag on an enclosing `describe`/class whose title is also
part of the identity). When several match, the one whose label is longest —
the most specific one — is the location reported for that binding.

## Pitfalls

- **`tag_not_in_results`**: a tag exists in source but no results test case
  ever cites the same reference with a matching label. Two distinct causes
  produce it, and `shallnot check` cannot tell them apart: the test did not
  run (excluded, filtered out, a different suite was executed), or it ran
  but the runner/reporter did not put the tag's enclosing title into the
  XML at all — see the per-runner notes below for exactly which titles each
  reporter surfaces.
- A tag inside a comment is invisible to the source scan and therefore never
  gives a location or a `tag_not_in_results` finding; if the test still
  cites the reference in the runner's output, it binds anyway, just without
  a source location for the citation itself (though a `[verifies ...]`
  string inside a real test title, method name, or `@DisplayName` still
  gives one).
- A skipped pytest test never executes its body, so `record_property` never
  runs and nothing is recorded for that invocation — use
  `pytest.mark.verifies` (read by the `conftest.py` hook before execution)
  for a requirement that must still be traceable when its test is skipped.

## What the source scan skips

The scan under each `--tests` root skips, by default:

- the directories in `internal/adapters/scan/scanner.go`'s `DefaultExcludes`:
  `.git`, `node_modules`, `target`, `build`, `dist`, `.venv`, `venv`,
  `__pycache__`, `.gradle`, `.pytest_cache`, `.mypy_cache`, `coverage` (each
  matched recursively, anywhere under the root) — disable with
  `--no-default-excludes`; add more with `--exclude`;
- a file that appears to be binary (a NUL byte in its first 8 KiB);
- a file larger than 2 MiB;
- any file that is itself one of the run's `--specs`/`--focus` spec files or
  `--results` results files, so a reference example embedded in a spec or a
  tag quoted inside a results file is never read back as a source tag.

## pytest

Tag with `@pytest.mark.verifies("ID~REV", ...)` on a function or class, or
`pytestmark = pytest.mark.verifies(...)` at module level to tag every test
in the module. This requires registering the marker and forwarding it as a
result property with a `conftest.py` hook — copy it verbatim
(`fixtures/cart/pytest/conftest.py`):

```python
def pytest_collection_modifyitems(items):
    for item in items:
        for marker in item.iter_markers(name="verifies"):
            item.user_properties.append(("verifies", ", ".join(marker.args)))
```

and register the marker, to keep `pytest --strict-markers` (if used) quiet:

```ini
[pytest]
markers =
    verifies: link a test to one or more requirement IDs in the form ID~REV
```

Without a `conftest.py` at all, `record_property("verifies", "ID~REV")` as
the first line of a test works too — no marker registration needed — but a
test skipped by a marker never executes its body, so nothing is recorded for
it (see [Pitfalls](#pitfalls)). `record_property` triggers a pytest warning under `-o junit_family=xunit2`;
`-o junit_family=xunit1` is warning-free. Both the marker family and the
`record_property` family are read from results; a project may mix them.

Example (`fixtures/cart/pytest/tests/test_totals.py`):

```python
@pytest.mark.verifies("CART-1~1")
def test_total_sums_price_times_quantity():
    cart = Cart()
    cart.add_item("mug", "9.50", 3)
    cart.add_item("plate", "4.00", 2)
    assert cart.total() == pytest.approx(36.50)
```

Run (`fixtures/cart/pytest/regen.sh`):

```
$ python -m pytest -o junit_family=xunit1 --junitxml=results/junit.xml -q
```

emits, in `results/junit.xml`:

```xml
<testcase classname="tests.test_totals" name="test_total_sums_price_times_quantity" file="tests/test_totals.py" line="5" time="0.000"><properties><property name="verifies" value="CART-1~1" /></properties></testcase>
```

and

```
$ shallnot check --specs fixtures/cart/spec --focus fixtures/cart/spec \
    --tests fixtures/cart/pytest --results fixtures/cart/pytest/results/junit.xml
```

binds it:

```
  CART-1~1  covered  fixtures/cart/spec/cart.md:8
      passed  tests.test_totals › test_total_sums_price_times_quantity  fixtures/cart/pytest/tests/test_totals.py:6
```

Parametrised tests (`@pytest.mark.parametrize` above `@pytest.mark.verifies`,
`fixtures/cart/pytest/tests/test_discounts.py`) each produce their own
`<testcase>`, one per parameter set, each carrying the `verifies` property
and each bound separately, for example
`test_discount_code_reduces_total[10.00-2-10-18.0]`. A class decorated with
`@pytest.mark.verifies` tags every test method inside it. A skipped test
(`pytest.mark.skip`) still emits its `<testcase>` with a `<skipped>` child
and, when tagged with the marker (not `record_property`), keeps its
`verifies` property — it binds, with outcome `skipped`.

## Jest

Tag in the `it`/`test` title, or in a `describe` title to tag every test
nested inside it. Configure `jest-junit` as a reporter; default options are
enough (`fixtures/cart/jest/jest.config.js`):

```javascript
module.exports = {
  reporters: [
    "default",
    ["jest-junit", { outputDirectory: "results", outputName: "junit.xml" }],
  ],
};
```

Example (`fixtures/cart/jest/test/cart.test.js`):

```javascript
it("sums line items [verifies CART-1~1]", () => {
  const items = [{ price: 10, quantity: 2 }, { price: 5, quantity: 3 }];
  expect(calculateTotal(items)).toBe(35);
});
```

Run:

```
$ npx jest
```

emits, in `results/junit.xml`:

```xml
<testcase classname=" sums line items [verifies CART-1~1]" name=" sums line items [verifies CART-1~1]" time="0.001">
</testcase>
```

(`jest-junit` repeats the full title in both `name` and `classname`, with a
leading space when there is no enclosing `describe`). Then:

```
$ shallnot check --specs fixtures/cart/spec --focus fixtures/cart/spec \
    --tests fixtures/cart/jest/test --results fixtures/cart/jest/results/junit.xml
```

binds it:

```
  CART-1~1  covered  fixtures/cart/spec/cart.md:8
      passed  sums line items [verifies CART-1~1]  fixtures/cart/jest/test/cart.test.js:11
```

`it.each([...])(title, fn)` titles work: each generated title becomes its
own `<testcase>` and each binds independently, for example
`"discount codes applies 10% off a 100 total to get 90 [verifies CART-2~2]"`.
A tag on a `describe` title (`fixtures/cart/jest/test/cart.test.js`:
`describe("discount codes [verifies CART-2~2]", ...)`) is included by
`jest-junit` in every nested testcase's `name`/`classname`, so it binds
every test underneath, however deeply nested. `it.skip` still emits a
`<testcase>` with a `<skipped/>` child, carrying the same title, and binds
with outcome `skipped`.

## Vitest

Tag in the `it`/`test` title, or in a `describe` title to tag every nested
test, exactly as with Jest — Vitest's own JUnit reporter includes the
enclosing `describe` titles in each testcase's name the same way. No
plugin, no reporter options beyond selecting it
(`fixtures/cart/vitest/vitest.config.ts`):

```typescript
export default defineConfig({
  test: {
    reporters: ["junit"],
    outputFile: "results/junit.xml",
  },
});
```

Example (`fixtures/cart/vitest/test/cart.test.ts`):

```typescript
it("sums line items [verifies CART-1~1]", () => {
  const total = calculateTotal([{ price: 10, quantity: 2 }, { price: 5, quantity: 3 }]);
  expect(total).toBe(35);
});
```

Run:

```
$ npx vitest run
```

emits, in `results/junit.xml`:

```xml
<testcase classname="test/cart.test.ts" name="sums line items [verifies CART-1~1]" time="0.000575417">
</testcase>
```

(Vitest puts the file path, not the `describe` nesting, in `classname`; the
full nested title is in `name`.) Then:

```
$ shallnot check --specs fixtures/cart/spec --focus fixtures/cart/spec \
    --tests fixtures/cart/vitest/test --results fixtures/cart/vitest/results/junit.xml
```

binds it:

```
  CART-1~1  covered  fixtures/cart/spec/cart.md:8
      passed  test/cart.test.ts › sums line items [verifies CART-1~1]  fixtures/cart/vitest/test/cart.test.ts:11
```

`test.each([...])(title, fn)` titles (including the `$field` interpolation
form) work the same way as Jest's `it.each`. A tag on a `describe` title
tags every test nested inside it, at any depth. `it.skip` emits a
`<testcase>` with a `<skipped/>` child and binds with outcome `skipped`.

## JUnit 5 (Java and Kotlin)

Tag in `@DisplayName`. Which display names reach the XML — and therefore
which tags surface — depends on the build tool and its reporter
configuration; verify against the fixture's actual `results/*.xml`, not
against what the source alone suggests.

### Gradle

Gradle's built-in test task writes JUnit XML with each test method's own
`@DisplayName` as the `<testcase name="...">`, natively, with no extra
configuration beyond `useJUnitPlatform()`
(`fixtures/cart/junit5-gradle/build.gradle`):

```groovy
test {
    useJUnitPlatform()
}
```

Example (`fixtures/cart/junit5-gradle/src/test/java/cart/CartTotalTest.java`):

```java
@Test
@DisplayName("sums line item prices times quantities [verifies CART-1~1]")
void sumsLinePricesTimesQuantities() { ... }
```

Run:

```
$ ./gradlew test
```

emits, in `build/test-results/test/TEST-cart.CartTotalTest.xml` (copied to
`results/` by the fixture's `regen.sh`):

```xml
<testcase name="sums line item prices times quantities [verifies CART-1~1]" classname="cart.CartTotalTest" time="0.005"/>
```

Then:

```
$ shallnot check --specs fixtures/cart/spec --focus fixtures/cart/spec \
    --tests fixtures/cart/junit5-gradle/src --results fixtures/cart/junit5-gradle/results
```

binds it:

```
  CART-1~1  covered  fixtures/cart/spec/cart.md:8
      passed  cart.CartTotalTest › sums line item prices times quantities [verifies CART-1~1]  fixtures/cart/junit5-gradle/src/test/java/cart/CartTotalTest.java:18
```

Under Gradle, a class-level `@DisplayName` on a `@Nested` class becomes the
*testsuite* `name` of that nested class's own XML file, not any
`<testcase>`'s `name` or `classname` — a tag placed there never surfaces on
a testcase and is reported `tag_not_in_results`
(`fixtures/cart/junit5-gradle/src/test/java/cart/CartTotalTest.java`,
`@Nested @DisplayName("gift bundle pricing [verifies CART-1~1]") class GiftBundlePricing`).
Likewise, a `@ParameterizedTest`'s own `@DisplayName` (the annotation on the
method, naming what the parameterized case as a whole verifies) is not
written to the XML under Gradle — only the per-invocation `name` from
`@ParameterizedTest(name = "...")` is. To tag every invocation of a
parameterized test under Gradle, put the tag in that `name` pattern and on
each plain `@Test` method individually; do not rely on a container-level
`@DisplayName`.

### Maven Surefire

Surefire writes plain method names by default; configure the
`statelessTestsetReporter` to phrase names from JUnit 5 display names
(`fixtures/cart/junit5-maven/pom.xml`, and identically
`fixtures/cart/junit5-kotlin/pom.xml`):

```xml
<plugin>
  <groupId>org.apache.maven.plugins</groupId>
  <artifactId>maven-surefire-plugin</artifactId>
  <configuration>
    <statelessTestsetReporter implementation="org.apache.maven.plugin.surefire.extensions.junit5.JUnit5Xml30StatelessReporter">
      <disable>false</disable>
      <version>3.0</version>
      <usePhrasedFileName>false</usePhrasedFileName>
      <usePhrasedTestSuiteClassName>true</usePhrasedTestSuiteClassName>
      <usePhrasedTestCaseClassName>true</usePhrasedTestCaseClassName>
      <usePhrasedTestCaseMethodName>true</usePhrasedTestCaseMethodName>
    </statelessTestsetReporter>
  </configuration>
</plugin>
```

Example (`fixtures/cart/junit5-maven/src/test/java/com/example/cart/CartTotalTest.java`):

```java
@Test
@DisplayName("sums line item prices times quantities [verifies CART-1~1]")
void totalIsSumOfLineItemPricesTimesQuantities() { ... }
```

Run:

```
$ mvn test
```

emits, in `target/surefire-reports/TEST-com.example.cart.CartTotalTest.xml`:

```xml
<testcase name="sums line item prices times quantities [verifies CART-1~1]" classname="com.example.cart.CartTotalTest" time="0.0"/>
```

Then:

```
$ shallnot check --specs fixtures/cart/spec --focus fixtures/cart/spec \
    --tests fixtures/cart/junit5-maven/src --results fixtures/cart/junit5-maven/results
```

binds it:

```
  CART-1~1  covered  fixtures/cart/spec/cart.md:8
      passed  com.example.cart.CartTotalTest › sums line item prices times quantities [verifies CART-1~1]  fixtures/cart/junit5-maven/src/test/java/com/example/cart/CartTotalTest.java:18
```

Under Surefire with this configuration, both a `@ParameterizedTest`'s own
`@DisplayName` (as the parameterized case's container title, tagging every
invocation) and a class-level `@DisplayName` on a `@Nested` class (surfacing
in `classname`, so tagging every test method inside it) do surface — the
opposite of Gradle's behaviour for the same annotations. The example project
places
`@ParameterizedTest(name = "{0}% off {1} gives {2} [verifies CART-2~2]")` and
a redundant `@DisplayName("percentage discount code reduces the total")` on
`percentageDiscountReducesTotal`; the `name` pattern's tag is what appears
per invocation, since Surefire substitutes `{0}`, `{1}`, `{2}` and keeps the
rest.

### Kotlin

JUnit 5 tests written in Kotlin behave identically to Java under either
build tool — `@DisplayName`, `@Nested`, and `@ParameterizedTest` follow the
same Gradle/Surefire rules above. `fixtures/cart/junit5-kotlin` builds with
Maven and the same `statelessTestsetReporter` configuration as
`junit5-maven`; a class-level `@DisplayName` on an `inner class` annotated
`@Nested` (`BulkOrderHandling` in
`fixtures/cart/junit5-kotlin/src/test/kotlin/cart/CartTest.kt`) surfaces in
`classname` (`"CartTest bulk order handling [verifies CART-1~1]"`) exactly
as the Java `@Nested` example does.

## Go

`shallnot` uses this convention on itself. Tag in a `t.Run` subtest name.
`go test` rewrites spaces in a subtest name to underscores before it reaches
the results; `shallnot` reads `[verifies_ID~1]` (underscores in place of the
spaces around and inside the tag) exactly as it would read
`[verifies ID~1]`, since the tag grammar trims spaces and underscores around
each reference. Produce JUnit XML with `gotestsum` (`script/test`):

```
$ go tool gotestsum --format pkgname --junitfile build/test-results/go.xml -- -count=1 ./...
```

## Any other runner

Emit the tag in the testcase's `name` (or `classname`), or as a
`<property name="verifies" value="ID~REV, ID~REV"/>` inside the `<testcase>`
— whichever the runner's JUnit XML reporter supports. Both forms are read
identically regardless of which runner produced them.

## See also

- [spec-format.md](spec-format.md) — requirements, revisions and the
  `id_pattern` a reference's ID must match.
- [report.md](report.md) — finding categories (including `tag_not_in_results`,
  `orphan_tag`, `revision_mismatch`, `malformed_tag`), coverage states and
  report formats.
- [configuration.md](configuration.md) — `--tests`, `--results`, `--exclude`
  and `--no-default-excludes`.
