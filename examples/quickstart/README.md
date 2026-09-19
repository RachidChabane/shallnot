# Quickstart

A spec, one module and its pytest tests.

`spec.md` declares three requirements:

```markdown
- **PWD-1~1**: WHEN a password is shorter than 12 characters THE SYSTEM SHALL
  reject it.
- **PWD-2~1**: WHEN a password contains the account's username THE SYSTEM
  SHALL reject it.
- **PWD-3~1**: THE password form SHALL feel welcoming.
  - Non-testable: judged in moderated usability sessions.
```

`tests/test_password.py` binds two tests to `PWD-1~1`:

```python
@pytest.mark.verifies("PWD-1~1")
def test_short_passwords_are_rejected():
    assert not is_acceptable("short", username="ada")
```

`conftest.py` makes pytest write the marker into its JUnit XML (five lines of
project code, no plugin to install):

```python
def pytest_collection_modifyitems(items):
    for item in items:
        for marker in item.iter_markers(name="verifies"):
            item.user_properties.append(("verifies", ", ".join(marker.args)))
```

Run the tests, then the gate:

```sh
cd examples/quickstart   # from the repository root
python -m pytest --junitxml=junit.xml
shallnot check --specs spec.md --tests tests --results junit.xml
```

The verdict is `FAIL`, with exit code `1`: `PWD-2~1` has no test. Write one, tag it `PWD-2~1`, fix `password.py` until it
passes, and the verdict turns to `PASS`. `username-rule.patch` holds that
change:

```sh
patch -p1 < username-rule.patch
python -m pytest --junitxml=junit.xml
shallnot check --specs spec.md --tests tests --results junit.xml   # PASS, exit code 0
```

[shallnot-demo](https://github.com/RachidChabane/shallnot-demo) is the same
project gated in GitHub Actions on Linux, macOS and Windows; its Actions
history shows the blocked run and the passing one. Change the meaning of `PWD-1` and
bump it to `PWD-1~2`: both existing tags become `revision_mismatch` findings
until the tests are re-verified and cite `PWD-1~2`.
