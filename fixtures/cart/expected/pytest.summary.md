## shallnot: FAIL

- **Focus:** every known requirement
- **Requirements:** 7 known, 7 in focus (2 covered, 1 failed, 1 skipped, 1 not run, 1 uncovered, 1 non-testable)
- **Tests:** 16 in results, 13 bound, 2 untagged
- **Findings:** 7 error, 1 warning, 2 info (7 blocking)

### Requirements in focus

| Requirement | Coverage | Bound tests | Declared at |
|---|---|---|---|
| `CART-1~1` | covered | 3 passed | `fixtures/cart/spec/cart.md:8` |
| `CART-4~1` | skipped | 1 skipped | `fixtures/cart/spec/cart.md:11` |
| `CART-2~2` | covered | 7 passed | `fixtures/cart/spec/cart.md:16` |
| `CART-3~1` | failed | 1 failed | `fixtures/cart/spec/cart.md:25` |
| `CART-5~1` | not_run | 1 not run | `fixtures/cart/spec/cart.md:27` |
| `CART-6~1` | uncovered | none | `fixtures/cart/spec/cart.md:32` |
| `CART-7~1` | non_testable | none | `fixtures/cart/spec/cart.md:34` |

### Findings

| Severity | Category | Location | Message |
|---|---|---|---|
| info | `untagged_test` | `fixtures/cart/pytest/results/junit.xml` | test "tests.test_misc › test_cart_starts_with_no_discount" verifies no stated requirement |
| info | `untagged_test` | `fixtures/cart/pytest/results/junit.xml` | test "tests.test_totals › test_total_with_no_discount_matches_subtotal" verifies no stated requirement |
| warning | `tag_not_in_results` | `fixtures/cart/pytest/tests/slow/test_quantity_limits.py:6` | tag citing CART-5~1 appears in no results file: the test did not run, or the tag sits where the runner does not report it |
| error | `revision_mismatch` | `fixtures/cart/pytest/tests/test_discounts.py:22` | tag cites CART-2~1, but the spec declares CART-2~2: re-verify the test against the current statement, then cite CART-2~2 |
| error | `orphan_tag` | `fixtures/cart/pytest/tests/test_misc.py:6` | tag cites CART-99~1, but no known spec declares CART-99 |
| error | `malformed_tag` | `fixtures/cart/pytest/tests/test_misc.py:13` | malformed tag pytest.mark.verifies("CART-1"): "CART-1" cites no revision (expected ID~REVISION) |
| error | `skipped_requirement` | `fixtures/cart/spec/cart.md:11` | requirement CART-4~1 is not verified: its bound tests were skipped |
| error | `failed_requirement` | `fixtures/cart/spec/cart.md:25` | requirement CART-3~1 is not verified: no bound test passed and at least one failed |
| error | `not_run_requirement` | `fixtures/cart/spec/cart.md:27` | requirement CART-5~1 is not verified: its tagged tests appear in no results file (they did not run, or the tag sits where the runner does not report it) |
| error | `uncovered_requirement` | `fixtures/cart/spec/cart.md:32` | requirement CART-6~1 has no bound test |
