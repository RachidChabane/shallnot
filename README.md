<p align="center">
  <img src="docs/logo.svg" width="96" alt="shallnot logo">
</p>

<h1 align="center">shallnot</h1>

<p align="center">
  A gate that stops a coding agent from saying "done" while a requirement has no passing test.
</p>

<p align="center">
  <a href="https://github.com/RachidChabane/shallnot/actions/workflows/ci.yml"><img src="https://github.com/RachidChabane/shallnot/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/RachidChabane/shallnot/releases/latest"><img src="https://img.shields.io/github/v/release/RachidChabane/shallnot" alt="Latest release"></a>
  <a href="https://pkg.go.dev/github.com/RachidChabane/shallnot"><img src="https://pkg.go.dev/badge/github.com/RachidChabane/shallnot.svg" alt="Go reference"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/RachidChabane/shallnot" alt="License: Apache-2.0"></a>
</p>

![An agent reports a feature done with all tests passing. shallnot blocks the end of its turn because requirement PWD-2~1 has no bound test. The agent fixes its tags and the gate passes.](https://raw.githubusercontent.com/RachidChabane/shallnot-demo/main/media/catch.gif)

Your specs give each requirement an ID. Tests say which requirement they
verify. shallnot reads the specs, the test sources and the JUnit XML, and
fails unless every requirement is cited by a test that actually ran and
passed. Hooked into the agent, it sends the findings back and the agent keeps
working until the gate is green.

It is one static binary. It never touches the network or a model, so the same
inputs always give the same report.

## Use it with your agent

Install the binary:

```sh
go install github.com/RachidChabane/shallnot/cmd/shallnot@latest
```

or take a prebuilt one from the
[releases page](https://github.com/RachidChabane/shallnot/releases/latest)
(Linux, macOS, Windows; amd64 and arm64; `checksums.txt` alongside).

Then, in your project:

```sh
shallnot init
```

This writes:

- `shallnot.yaml`, filled in for the test runner it finds (pytest, Jest,
  Vitest, Maven, Gradle, Go)
- a section in `AGENTS.md`, a `CLAUDE.md` that imports it, and project skills
  (`.agents/skills`, `.claude/skills`): how to write a requirement, tag a
  test, run the gate, review a tagged test
- an end-of-turn hook for Claude Code, and for each other harness the project
  is set up for: Codex, Copilot, Cursor, Gemini CLI, OpenCode, Qwen Code,
  Factory Droid, Goose

Commit those files. From then on you ask for features the usual way and never
mention shallnot. The agent writes the requirement, the code and the tagged
tests. If it tries to finish on a blocked gate, the hook hands it the findings.
After three blocks in a row the hook lets go and tells you.

VS Code Copilot picks up the Claude Code hook. Harnesses that cannot hold the
end of a turn (Windsurf, Kiro, Amp, Zed, Cline) still read `AGENTS.md` and
the skills, and CI catches what they let through. The full table is in
[docs/agents.md](docs/agents.md).

There is also a Claude Code plugin, for using the skills in every project
without committing anything:

```text
/plugin marketplace add RachidChabane/shallnot
/plugin install shallnot@shallnot
```

## What it looks like

A requirement in a Markdown (or YAML) spec, as `ID~REVISION`:

```markdown
- **PWD-2~1**: WHEN a password contains the account's username THE SYSTEM
  SHALL reject it.
```

A test bound to it. The tag goes wherever the runner will report it:

```python
@pytest.mark.verifies("PWD-2~1")                      # pytest
```
```js
it("rejects the username [verifies PWD-2~1]", ...)    // Jest, Vitest
```
```java
@DisplayName("rejects the username [verifies PWD-2~1]")   // JUnit 5
```
```go
t.Run("rejects the username [verifies PWD-2~1]", ...)     // Go
```

The gate:

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

A tag only counts if it shows up in the results file, so a tag in a comment
or on a test that never ran binds nothing. When the meaning of a requirement
changes, bump its revision: every test citing the old one fails the gate
until someone re-checks it.

`shallnot gate` runs your test commands and then checks. `shallnot check`
only reads results that already exist.

| Exit code | Meaning |
|---|---|
| `0` | clean |
| `1` | blocked: some requirement lacks a passing test, or a tag is wrong |
| `2` | no verdict (missing results, bad config). Says nothing about coverage. |

Try it on a tiny project: [examples/quickstart](examples/quickstart).

## In CI

```yaml
- run: pytest --junitxml=junit.xml
  continue-on-error: true
- uses: RachidChabane/shallnot@v0.4.1
  with:
    args: --specs specs --tests tests --results junit.xml
```

The action writes a job summary and annotates the offending lines.
[shallnot-demo](https://github.com/RachidChabane/shallnot-demo) is a small
repository gated this way.

## What a pass does not prove

A pass means every requirement is cited by a test that ran and passed. It
does not mean the test is any good. A tagged test that asserts nothing still
passes the gate. Judging that is a review job, and the `shallnot-review`
skill tells the agent how to do it.

## Documentation

- [docs/agents.md](docs/agents.md): `init`, the hooks, the plugin
- [docs/spec-format.md](docs/spec-format.md): requirements in Markdown and YAML
- [docs/binding.md](docs/binding.md): tagging tests, per runner
- [docs/configuration.md](docs/configuration.md): `shallnot.yaml`, flags, severities
- [docs/report.md](docs/report.md): the JSON report, every field and finding category
- [docs/pipeline-gate.md](docs/pipeline-gate.md): gating an unattended agent pipeline, several repositories
- [docs/comparison.md](docs/comparison.md): OpenFastTrace, Doorstop, StrictDoc, sphinx-needs
- [schemas/](schemas): JSON Schemas for the report, the config and YAML specs
- [llms.txt](llms.txt): the same index, for agents

## Contributing

Bug reports and pull requests are welcome. [CONTRIBUTING.md](CONTRIBUTING.md)
has the build and test commands. Security issues: [SECURITY.md](SECURITY.md).

## License

[Apache-2.0](LICENSE)
