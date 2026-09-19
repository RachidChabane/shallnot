# shallnot

A spec-to-test traceability gate. `shallnot` reads specifications whose
requirements carry stable IDs, finds the tests that declare which requirement
they verify, joins that with the **actual test results**, and gives a verdict:
every requirement has a real, passing test behind it, and no test claims to
verify a requirement that does not exist.

It is a single static binary. It makes no network access and calls no model:
the same inputs give the same output, byte for byte.

![An agent is asked for a feature and nothing else. It writes the requirement into the spec, implements it, tags the tests it writes, and runs the shallnot gate on its own](https://raw.githubusercontent.com/RachidChabane/shallnot-demo/main/media/session.gif)

The request in that session is "Passwords should also have to contain at least
one digit. Can you add that?". Nobody mentions shallnot: the repository is
equipped by `shallnot init`, so the agent writes the requirement down first
(`PWD-4~1`), implements it, tags its tests, and cannot end its turn on a
blocked gate.

What the gate prints when a requirement has no test:

```text
$ shallnot check --specs spec.md --tests tests --results junit.xml
shallnot: FAIL
focus: every known requirement
requirements: 3 known, 3 in focus (1 covered, 0 failed, 0 skipped, 0 not run, 1 uncovered, 1 non-testable)
tests: 2 in results, 2 bound, 0 untagged
findings: 1 error, 0 warning, 0 info (1 blocking)

REQUIREMENTS
  PWD-1~1  covered  spec.md:3
      passed  tests.test_password › test_long_passwords_are_accepted   tests/test_password.py:11
      passed  tests.test_password › test_short_passwords_are_rejected  tests/test_password.py:6
  PWD-2~1  uncovered  spec.md:5
  PWD-3~1  non_testable  spec.md:7

FINDINGS
  error  uncovered_requirement  spec.md:5  requirement PWD-2~1 has no bound test
```

## Why

When coding agents write both the code and the tests, nobody re-derives from a
diff whether the implementation does what was asked. The tests become the only
ground truth about whether a requirement is met, and that truth is worth
something only if a machine guarantees the mapping between requirements and
passing tests. With that guarantee, three reviewers stop doing each other's
job: whoever owns intent reviews the spec, whoever owns verification reviews
the rigour of the tests, and `shallnot` enforces the mapping between the two.

Requiring an agent to cite a requirement ID on every line of generated code
makes invented requirements mechanically detectable, but measurably reduces
the consistency of the agent's output
([arXiv 2606.30689](https://arxiv.org/abs/2606.30689)). `shallnot` moves the
citation from the code line to the test, where its placement is unambiguous:
the citation belongs on the test that verifies the requirement. A cited ID
that no spec declares is an `orphan_tag` finding.

`shallnot` serves three readers: a CI pipeline (exit code, job summary,
annotations), an engineer at a terminal, and an orchestrator of coding agents
that consumes the JSON report with no human in the loop.

**What a pass proves, and what it does not.** A pass proves that every
requirement in focus is cited by at least one test that ran and passed, at the
requirement's current revision, and that every citation resolves. It does not
prove that the test's assertions actually verify the requirement. A tag is
necessary evidence, not sufficient evidence; the rigour of the test remains a
review concern.

## How it works

1. **Specs** in Markdown or YAML declare requirements: `ID~REVISION: statement`.
   IDs take whatever shape the project uses (`REQ-042`, `ABC-101.AC3`), set by
   a regular expression. See [docs/spec-format.md](docs/spec-format.md).
2. **Tests** carry the tag `[verifies ID~REVISION]` where the test runner
   reports it: the test title (Jest, Vitest), `@DisplayName` (JUnit 5), a
   `verifies` property (pytest). See [docs/binding.md](docs/binding.md).
3. **Results** are JUnit XML, which pytest, Jest, Vitest, Maven, Gradle and
   most other runners can emit. A tag binds a test only if it appears in the
   results file, so the link between a tag and an outcome is read, never
   assumed. A scan of the test sources adds file and line, and catches tagged
   tests that never ran.
4. **The verdict** classifies every gap as a finding with a category, a
   configurable severity, a file and a line. A requirement is *covered* when at
   least one bound test passed; *failed*, *skipped*, *not run* and *uncovered*
   are distinct states. Bumping a requirement's revision invalidates every tag
   citing the old one, which forces the tests to be re-verified when the
   meaning of a requirement changes.

| Exit code | Meaning |
|---|---|
| `0` | Clean: no finding reached the blocking severity (or the run is advisory). |
| `1` | Blocked: at least one finding reached the blocking severity. |
| `2` | Tool failure: no verdict was produced. Never a statement about coverage. |

## Install

Download the archive for your platform from the
[releases page](https://github.com/RachidChabane/shallnot/releases), verify it
against `checksums.txt`, and put the binary on your `PATH`:

```sh
os=linux arch=amd64   # linux|darwin|windows, amd64|arm64
base=https://github.com/RachidChabane/shallnot/releases/latest/download
curl -fsSLO "$base/shallnot_${os}_${arch}.tar.gz"
curl -fsSL "$base/checksums.txt" | grep " shallnot_${os}_${arch}.tar.gz\$" | sha256sum -c -
tar -xzf "shallnot_${os}_${arch}.tar.gz" shallnot
sudo install shallnot /usr/local/bin/
```

On macOS, replace `sha256sum -c -` with `shasum -a 256 -c -`. On Windows,
download `shallnot_windows_amd64.zip` or `shallnot_windows_arm64.zip`.

Or build from source with Go:

```sh
go install github.com/RachidChabane/shallnot/cmd/shallnot@latest
```

In GitHub Actions:

```yaml
- run: pytest --junitxml=junit.xml        # your test step, producing JUnit XML
  continue-on-error: true
- uses: RachidChabane/shallnot@v0.3.0
  with:
    args: --specs specs --tests tests --results junit.xml
```

The action downloads the binary, runs `shallnot check`, writes the Markdown
summary to the job summary, and annotates the offending lines.

## Five-minute example

The project in [`examples/quickstart`](examples/quickstart) has a spec, one
module and its pytest tests.

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
cd examples/quickstart
python -m pytest --junitxml=junit.xml
shallnot check --specs spec.md --tests tests --results junit.xml
```

The output is the report at the top of this page, with exit code `1`:
`PWD-2~1` has no test. Write one, tag it `PWD-2~1`, fix `password.py` until it
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

For a machine reader, add `--format json` or `--json-out report.json`; for a
configuration file instead of flags, see
[docs/configuration.md](docs/configuration.md).

## Agents use it without being asked

The person asking an agent for a feature does not care about traceability,
and should not have to mention it. A repository equipped with one command
tells every agent that works in it what to do, and holds the end of the
agent's turn until the gate passes:

```sh
shallnot init
```

`init` writes a starter `shallnot.yaml` for the test runners it finds, the
agent instructions (`AGENTS.md`, a `CLAUDE.md` import, project skills), and
an end-of-turn hook for Claude Code, and for Cursor when the project uses it.
The instructions are three skills, one per step of the work:

| Skill | The agent learns to |
|---|---|
| [`shallnot-plan`](plugin/skills/shallnot-plan/SKILL.md) | write the requested behaviour down as a requirement, with an ID and a revision, before building it |
| [`shallnot`](plugin/skills/shallnot/SKILL.md) | tag the tests it writes, run `shallnot gate` before it says it is done, and react to each finding |
| [`shallnot-review`](plugin/skills/shallnot-review/SKILL.md) | judge whether a tagged test really verifies its requirement, the half the gate cannot check |

With that in place, a request as plain as "passwords should need a digit"
leads the agent to add the requirement to the spec, implement it, tag its
tests, and run the gate; if it tries to finish on a blocked gate, the hook
hands it the blocking findings and sends it back to work. See
[docs/agents.md](docs/agents.md).

The same skills ship as a plugin in [`plugin/`](plugin): an
[Agent Plugins](https://github.com/agentplugins/agent-plugins-spec) package
and a Claude Code plugin in one directory.

```text
/plugin marketplace add RachidChabane/shallnot
/plugin install shallnot@shallnot
```

## Documentation

| Document | Content |
|---|---|
| [docs/spec-format.md](docs/spec-format.md) | The requirement model; the Markdown and YAML spec formats. |
| [docs/binding.md](docs/binding.md) | The tag convention and its syntax for pytest, Jest, Vitest, JUnit 5 (Java, Kotlin; Maven, Gradle), Go and other runners. |
| [docs/report.md](docs/report.md) | The JSON report: a versioned public API, with every field, state and finding category. |
| [docs/configuration.md](docs/configuration.md) | `shallnot.yaml`, every flag, severities, exit codes. |
| [docs/pipeline-gate.md](docs/pipeline-gate.md) | Using `shallnot` as a gate in an automated agent pipeline: focus, advisory mode, several repositories, exit codes. |
| [docs/agents.md](docs/agents.md) | Equipping coding agents: `shallnot init`, the end-of-turn hooks, the plugin package. |
| [plugin/skills](plugin/skills) | The agent skills: writing requirements, binding tests and passing the gate, reviewing tests against requirements; and what an agent must never do to get a green report. |
| [plugin/evals](plugin/evals) | The eval suite run by `claude plugin eval`, with the plugin and without it. |
| [schemas/](schemas) | JSON Schemas of the report, the config file and the YAML spec; also printed by `shallnot schema report\|config\|spec`. |

## Relation to other tools

Requirement tracing is an established field. `shallnot` exists because of one
combination that the existing tools do not offer: a tag bound to a specific
test and joined with that test's actual result, in a dependency-free binary
with a JSON report designed for programs.

| | Spec format | How coverage is declared | Uses test results | Runtime | Licence |
|---|---|---|---|---|---|
| **shallnot** | Markdown or YAML, IDs of any shape, revisions | Tag surfaced by the test runner into JUnit XML | Yes: a requirement is covered only by a test that ran and passed | None (static binary) | Apache-2.0 |
| [OpenFastTrace](https://github.com/itsallcode/openfasttrace) | Markdown/RST items `type~name~revision` with `Needs`/`Covers` | Comment tags `[utest->req~login~1]` found by a language-agnostic source scan | No: a tag in a file counts whether or not any test runs or passes | JVM | GPL-3.0 |
| [Doorstop](https://github.com/doorstop-dev/doorstop) | One YAML file per item, in a VCS tree | Explicit links between items | No | Python | LGPL-3.0 |
| [StrictDoc](https://github.com/strictdoc-project/strictdoc) | Its own `.sdoc` document format | `@relation` markers in source files | Partly: JUnit XML import into its documentation model | Python | Apache-2.0 |
| [sphinx-needs](https://github.com/useblocks/sphinx-needs) | Sphinx directives | Links between need objects | Through the separate sphinx-test-reports extension, as linkable objects in a Sphinx build | Python + Sphinx | MIT |

**Why this is not a contribution to OpenFastTrace.** OpenFastTrace is the
closest relative, and `shallnot` borrows two of its ideas deliberately: the
revision carried in every reference, so that a change of meaning invalidates
the coverage that cites the old revision; and tag discovery by a
language-agnostic scan, which costs no code per framework. The differences are
structural rather than incremental:

- OpenFastTrace traces *static* links across artifact types (requirement →
  design → implementation → test). Whether a test runs, passes, or is skipped
  is outside its model. `shallnot`'s model is the join between a tag and a
  test outcome; without that join it has nothing to say.
- OpenFastTrace requires a JVM and integrates through Maven and Gradle.
  `shallnot` is one static binary, usable alike in a Python, JavaScript or JVM
  project, and in a container that holds nothing else.
- `shallnot`'s primary output is a versioned JSON report consumed by programs
  and agents, with exit codes that separate a verdict from a tool failure.
- `shallnot` has an explicit, justified non-testable status, and accepts
  requirement IDs in whatever shape the project already uses rather than
  `type~name~revision`.

OpenFastTrace remains the better tool for multi-level tracing across
artifact types, which `shallnot` does not attempt. StrictDoc and sphinx-needs
are documentation systems: adopting them means authoring requirements in
their formats and building their documents, which suits a documentation-led
process and not a gate that reads the Markdown plan an agent wrote an hour
ago. [ReqToCode](https://arxiv.org/abs/2603.13999) embeds requirements as
language-native code elements validated at build time; it gives stronger
structural guarantees inside one language, where `shallnot` stays outside the
language and works from what every test runner can already emit.

## Development

```sh
script/build            # bin/shallnot
script/test             # all tests, writes build/test-results/go.xml
script/trace            # shallnot traces its own requirements (specs/) to its own tests
script/ci               # lint, then `shallnot gate` on this repository, then `shallnot init --check`
script/fuzz             # fuzz the parsers
script/regen-fixtures   # re-run pytest, Jest, Vitest, Maven and Gradle on the fixture projects
script/eval-plugin      # claude plugin eval on the plugin, with and without it (spends tokens)
script/demo             # record a real agent session (needs a capture-session script)
```

`make <verb>` runs the same scripts. `shallnot` gates itself: its
requirements are in [specs/shallnot.md](specs/shallnot.md), its Go tests carry
the tags in their subtest names, CI fails if a requirement loses its passing
test, and the repository is equipped by its own `shallnot init`. The design decisions are recorded in [docs/adr](docs/adr).

## Licence

[Apache-2.0](LICENSE)
