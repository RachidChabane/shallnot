# Contributing

Bug reports, questions and pull requests are welcome. For anything larger
than a fix, open an issue first so we can agree on the shape of it.

## Building and testing

You need Go (the version in `go.mod`). Nothing else for the core.

```sh
script/build            # bin/shallnot
script/test             # all tests, writes build/test-results/go.xml
script/lint             # gofmt and go vet
script/ci               # what CI runs: lint, build, shallnot gate, shallnot init --check
script/fuzz             # fuzz the parsers
script/regen-fixtures   # re-run pytest, Jest, Vitest, Maven and Gradle on the fixture projects
script/eval-plugin      # claude plugin eval on the plugin (spends model tokens)
```

`make <verb>` runs the same scripts. Run `script/ci` before you push.

## The repository gates itself

shallnot's own requirements are in [specs/shallnot.md](specs/shallnot.md). A
change in behaviour comes with a requirement (new, or a bumped revision) and a
Go test that cites it in a subtest name:

```go
t.Run("exits 0 when clean and 1 when blocked [verifies SN-52~1]", ...)
```

CI fails if a requirement loses its passing test.

## Layout

- `internal/domain`, `internal/analysis`: the model and the verdict. Pure, no I/O.
- `internal/adapters`: spec parsers, source scan, JUnit XML, reports, config, agent hooks.
- `internal/app`, `internal/cli`: use cases and the command line.
- `plugin/`: the agent skills, embedded in the binary and packaged as a plugin.
- `fixtures/`: real projects with results from their real test runners. Regenerate them, never edit the XML by hand.

Design decisions are recorded in [docs/adr](docs/adr).

## Pull requests

Keep them small and about one thing. Code, comments and commit messages are
in English. Contributions are licensed under Apache-2.0, like the rest of the
repository.
