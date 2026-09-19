# Equipping coding agents

This is the reference for making coding agents that work in a repository use
`shallnot` without being asked. It covers `shallnot init`, the two end-of-turn
hooks, and the plugin package.

See also [docs/spec-format.md](spec-format.md) for how requirements are
declared, [docs/binding.md](binding.md) for tagging tests, [docs/report.md](report.md)
for the report a gate produces, and [docs/configuration.md](configuration.md)
and [docs/pipeline-gate.md](pipeline-gate.md) for the gate itself.

## Two halves

Equipping a repository for agents has two independent halves, and an agent
that only gets one of them is only half equipped:

- **Knowledge**: three agent skills, read in this order:

  | Skill | What it teaches | When the agent uses it |
  |---|---|---|
  | `shallnot-plan` | how to state a requested behaviour as a requirement, with an ID and a revision, before building it | a request describes behaviour no existing requirement covers |
  | `shallnot` | the `[verifies ID~REVISION]` convention, when to run `shallnot gate`, and how to react to each finding category | writing or changing code or tests, and before reporting work as done |
  | `shallnot-review` | how to judge whether a tagged test really verifies its requirement — the half the gate cannot check | reviewing a change, a branch or a pull request, or one's own work before reporting it done |

  One rule separates planning from gaming, and each skill states it: a
  requirement is written from a request for behaviour, before the code that
  implements it — never afterwards, to give an orphan tag something to cite
  or to make a finding disappear. An agent that reads the skills knows to use
  shallnot even when nobody mentions it, but nothing forces it to actually
  run the gate before it stops.
- **Enforcement**: an end-of-turn hook runs the gate itself, before the
  agent's turn ends, and sends the agent back to work when the gate is
  blocked. This holds even if the agent never read the skills, forgot to run
  the gate, or decided on its own that the work was done.

A harness that only reads `AGENTS.md` (knowledge, no hook support) relies on
the agent choosing to run the gate. A harness with hook support additionally
enforces it. Both halves are installed by `shallnot init` in one pass.

## `shallnot init`

`shallnot init` inspects a project and writes or updates the files an agent
needs, without ever rewriting a file the project already owns the content of.

### Detected test runners

| Runner | Detection signal | Test command written | Results path | Test roots | Remaining manual step |
|---|---|---|---|---|---|
| pytest | `pytest.ini`, `conftest.py` or `tox.ini` present, or `pyproject.toml`/`setup.cfg`/`requirements.txt` mentions `pytest` | `pytest -o junit_family=xunit1 --junitxml=test-results/pytest.xml` | `test-results/pytest.xml` | `tests`, `test` (whichever exist) | none |
| vitest | `package.json` mentions `"vitest"` | `npx vitest run --reporter=default --reporter=junit --outputFile.junit=test-results/vitest.xml` | `test-results/vitest.xml` | `src`, `test`, `tests`, `__tests__` (whichever exist) | none |
| jest | `package.json` mentions `"jest"` | `npx jest --reporters=default --reporters=jest-junit` | `junit.xml` | `src`, `test`, `tests`, `__tests__` (whichever exist) | install the reporter with `npm install --save-dev jest-junit`; it writes `junit.xml` in the project root |
| maven | `pom.xml` present | `mvn -B test` | `target/surefire-reports/*.xml` | `src/test` | make Surefire report display names, or a `@DisplayName` tag never reaches the results (see the Maven Surefire section of [docs/binding.md](binding.md)) |
| gradle | `build.gradle` or `build.gradle.kts` present | `./gradlew test` | `build/test-results/test/*.xml` | `src/test` | none |
| go | `go.mod` present | `go run gotest.tools/gotestsum@latest --junitfile test-results/go.xml ./...` | `test-results/go.xml` | `.` | none |

More than one runner can be detected in the same project; each contributes
its own `test_commands` and `results` entries, and their test roots are
merged without duplicates. When no runner is detected, `shallnot init` adds a
next step naming `tests`, `results` and `test_commands` to set by hand.

### Files it writes

- **`shallnot.yaml`**: created only when absent, with `version: 1`, `specs:
  [specs]`, the detected runners' `tests`, `results` and `test_commands`.
  When the file already exists, `init` leaves its content untouched — the
  project owns its configuration once written.
- **`AGENTS.md`**: the three skills are written as a section between two
  exact markers, `<!-- shallnot:begin -->` and `<!-- shallnot:end -->`, in
  reading order (`shallnot-plan`, `shallnot`, `shallnot-review`). On a repeat
  run, only the text between the markers is replaced; everything before and
  after is left alone. On a first run, the section is appended to the end of
  the file (or the file is created holding only the section, if it did not
  exist or was empty).
- **`CLAUDE.md`**: an `@AGENTS.md` import line is appended, unless a line
  that is exactly `@AGENTS.md` is already present anywhere in the file.
- **`.agents/skills/<name>/SKILL.md`** and **`.claude/skills/<name>/SKILL.md`**:
  one file per packaged skill (`shallnot-plan`, `shallnot`, `shallnot-review`)
  under each root, copied verbatim, so a harness discovers each as a project
  skill independently of `AGENTS.md`. `.agents/skills` is the directory the
  harnesses in the [skills column](#hook-and-instruction-coverage) share;
  `.claude/skills` is Claude Code's own.
- **pytest's `conftest.py`**: `init` adds the hook (below) only when the
  project has a detected pytest runner. If `conftest.py` is absent or empty,
  it is created holding just the hook. If it exists and already contains
  `item.iter_markers(name="verifies")`, nothing changes — the hook is already
  there. If it exists and defines its own `pytest_configure` or
  `pytest_collection_modifyitems` but not that marker check, `init` **refuses
  to write it**, exits with a tool failure, and says to merge the hook by
  hand; it never overwrites a hook function the project already defines.
  Otherwise, the hook is appended to the existing content.
- **The end-of-turn hook configuration of each selected harness**: for every
  harness named by `--hooks` (see below), `init` merges that harness's hook
  into its configuration file, keeping every other key and every other hook
  entry untouched. The file, the event name and the exact command for each
  harness are in the table under
  [End-of-turn hooks](#end-of-turn-hooks). Two harnesses need more than a
  merged hook entry: installing Gemini CLI's hook also adds `AGENTS.md` to
  `context.fileName` in `.gemini/settings.json`, ahead of the names already
  configured, because Gemini CLI otherwise reads only `GEMINI.md`; installing
  Goose's hook writes a project plugin under `.agents/plugins/shallnot/`
  (`plugin.json` plus `hooks/hooks.json`) rather than editing a shared
  settings file.

### Flags

- `--dir <directory>` — the project directory to equip (default `.`).
- `--hooks <list>` — which harnesses get an end-of-turn hook:
  - `auto` (default): installs the Claude Code hook always, plus every other
    harness whose configuration the project already shows a sign of — see
    the "Detected by" column of the
    [End-of-turn hooks](#end-of-turn-hooks) table for each harness's markers.
  - `none`: installs no hook.
  - a comma-separated list of harness names (`claude`, `codex`, `copilot`,
    `cursor`, `factory`, `gemini`, `goose`, `opencode`, `qwen`): installs
    exactly those, regardless of what exists on disk. An unknown name is a
    tool failure.
- `--check` — changes nothing on disk. Prints the same per-file plan as a
  normal run, then exits `1` (blocked) if applying it would create or update
  at least one file, or `0` (clean) if every file is already as `init` wants
  it.

### Idempotence

Running `shallnot init` twice with the same flags changes nothing on the
second run: every file it writes is derived deterministically from the
project's current state (or left untouched, for `shallnot.yaml` and an
already-correct `conftest.py`), so the second run's plan has no `create` or
`update` entries and `--check` exits `0`.

## End-of-turn hooks

`shallnot hook <name>` answers one harness's end-of-turn hook call. Two
protocols are implemented:

- **The exit-code protocol**, defined by Claude Code's `Stop` hook and shared
  by every harness whose hook events read that way: it reads a JSON object
  from standard input holding `session_id` and `cwd`, resolves the project
  directory as the first non-empty of the environment variables
  `CLAUDE_PROJECT_DIR`, `GEMINI_PROJECT_DIR`, `QWEN_PROJECT_DIR`,
  `FACTORY_PROJECT_DIR`, else the input's `cwd`, blocks by exiting `2` with
  the reason on standard error, gives up by exiting `1` with the reason on
  standard error, and announces a pass with `{"systemMessage": "<message>"}`
  on standard output. `shallnot hook stop` and `shallnot hook claude-stop`
  are the same adapter under two names: `stop` is what a harness's own
  configuration names; `claude-stop` is the name Claude Code configurations
  use.
- **A harness-specific protocol**, one per harness whose hook input or
  output differs from the exit-code protocol: `shallnot hook copilot-stop`
  answers GitHub Copilot's `agentStop` hook (input `sessionId`, `cwd`; it
  blocks by printing `{"decision":"block","reason":"<message>"}` on standard
  output and exiting `0`, and gives up or announces a pass by writing the
  message to standard error and exiting `0`); `shallnot hook cursor-stop`
  answers Cursor's `stop` hook (input `conversation_id`, `workspace_roots`,
  `loop_count`; it blocks by printing `{"followup_message": "<message>"}` on
  standard output and exiting `0`, and gives up or announces a pass by
  writing the message to standard error and exiting `0`).

Every hook, whichever protocol it speaks, follows the same steps once it has
read the event:

1. **Find the project to gate.** Starting from the resolved project
   directory, the hook looks upward through parent directories for the
   nearest one holding a `shallnot.yaml`. If none holds one, the hook exits
   `0` printing nothing on either stream: an ungated project is none of its
   business.
2. **Gate the project.** If `shallnot.yaml` sets `test_commands`, the hook
   runs `shallnot gate` (it runs the test commands, then checks their
   results). Otherwise it runs a plain check (`shallnot check`'s behavior)
   against whatever results files are already on disk.
3. **Decide.** A `pass` verdict, or a run in advisory mode, lets the turn end
   silently, with one exception: when the hook has sent the agent back
   earlier in the same session, the passing turn ends with one line for the
   user, `shallnot: gate passed: N requirement(s) in focus covered by passing
   tests, M non-testable, no blocking finding.`, rendered as that protocol's
   pass announcement. A `fail` verdict, or a run that produced no verdict at
   all (missing or stale results, bad config), sends the agent back to work
   with the blocking findings (or the failure reason) in the message,
   rendered as that protocol's block.
4. **Give up after three attempts.** The hook counts, per session, how many
   times in a row it has sent the same session back. Cursor reports this
   count itself (`loop_count`); every other hook keeps a small counter file
   per session under the OS temporary directory. On the attempt that would
   be the fourth consecutive block, the hook instead lets the turn end and
   tells the user the gate is still blocked (`shallnot: the gate is still
   blocked after 3 attempts; run \`shallnot gate\` to see why.`), then resets
   the counter. The next call after that starts counting from zero again.
5. **It is advisory, never blocking by force.** The hook can only ask a
   harness to continue the turn; it has no way to prevent an agent or a user
   from stopping regardless, and a harness without hook enforcement receives
   none of this.
6. **The hook's own failures never hold up the agent.** If the hook itself
   fails for reasons unrelated to the gate's verdict — an unknown hook name,
   unreadable standard input, input that is not valid JSON — it exits `1`,
   not the harness's own "block" exit code, so a broken hook lets the turn
   end instead of getting stuck.

### Hooks `shallnot init` can install

| Harness | Detected by | File written | Event | Hook command | Timeout |
|---|---|---|---|---|---|
| Claude Code | always selected | `.claude/settings.json` | `Stop` | `shallnot hook claude-stop` | 600 seconds |
| Codex CLI | `.codex` | `.codex/hooks.json` | `Stop` | `shallnot hook stop` | 600 seconds |
| GitHub Copilot | `.github/copilot-instructions.md`, `.github/hooks`, `.github/skills`, or `.github/instructions` | `.github/hooks/shallnot.json` | `agentStop` | `shallnot hook copilot-stop` | 600 seconds (`timeoutSec`) |
| Cursor | `.cursor` | `.cursor/hooks.json` | `stop` | `shallnot hook cursor-stop` | none (Cursor's `stop` hook takes no timeout key) |
| Factory Droid | `.factory` | `.factory/hooks.json` | `Stop` | `shallnot hook stop` | 600 seconds |
| Gemini CLI | `.gemini` or `GEMINI.md` | `.gemini/settings.json` | `AfterAgent` | `shallnot hook stop` | 600000 milliseconds |
| Goose | `.goosehints`, `.goose`, or `.agents/plugins` | `.agents/plugins/shallnot/plugin.json` and `.agents/plugins/shallnot/hooks/hooks.json` | `Stop` | `shallnot hook stop` | 600 seconds |
| OpenCode | `.opencode`, `opencode.json`, or `opencode.jsonc` | `.opencode/plugins/shallnot.js` | `session.idle` (a generated plugin) | `shallnot hook stop` (run by the plugin) | none |
| Qwen Code | `.qwen` or `QWEN.md` | `.qwen/settings.json` | `Stop` | `shallnot hook stop` | 600 seconds |

Notes on individual harnesses:

- Codex CLI, Factory Droid, Gemini CLI, Goose and Qwen Code all speak the
  exit-code protocol through the same `stop` command; each gets its own
  table row because each writes to a different configuration file with its
  own layout.
- Factory Droid's `hooks.json` has the event names at the top level, with no
  `hooks` wrapper object around them.
- Gemini CLI's entry also carries a `"name": "shallnot"` field, and its
  timeout is in milliseconds rather than seconds. Installing the hook also
  adds `AGENTS.md` to `context.fileName` in `.gemini/settings.json`, ahead of
  the names already configured, so that Gemini CLI reads it: by default,
  Gemini CLI reads only `GEMINI.md`.
- GitHub Copilot's hook file carries both a `bash` and a `powershell` key
  holding the same command, so the hook runs under either shell.
- Goose's hook is a project plugin (`.agents/plugins/shallnot/plugin.json`
  and `hooks/hooks.json`) rather than an entry merged into a shared settings
  file.
- OpenCode has no blocking end-of-turn hook. `init` instead writes a
  JavaScript plugin, `.opencode/plugins/shallnot.js`, that runs `shallnot
  hook stop` when a top-level session goes idle (`session.idle`) and, if
  that exits with code `2`, prompts the session again with the message on
  standard error. This holds the agent in the terminal interface and under
  `opencode serve`; `opencode run` exits at the first idle, before the new
  prompt is answered.
- Codex CLI needs one more step that `init` cannot take for the project:
  hooks must be enabled with `hooks = true` under `[features]` in Codex's
  `config.toml`, and the project must be trusted. `init` prints this as a
  next step.

### Captured transcript

Project (`shallnot.yaml` names no `test_commands`, so the hook runs a plain
check against the results already on disk):

`shallnot.yaml`:
```yaml
version: 1
specs: [spec.md]
results: [junit.xml]
```

`spec.md`:
```markdown
- **REQ-1~1**: THE SYSTEM SHALL work.
- **REQ-2~1**: THE SYSTEM SHALL also do this.
```

`junit.xml`:
```xml
<testsuite name="s"><testcase classname="c" name="works [verifies REQ-1~1]"/></testsuite>
```

Invocation, with `CLAUDE_PROJECT_DIR` set to that directory:

```
$ echo '{"session_id":"session-demo","cwd":"<project>","hook_event_name":"Stop"}' \
    | shallnot hook claude-stop
shallnot: the traceability gate is blocked by 1 finding(s). Resolve them before finishing:
- uncovered_requirement at spec.md:2: requirement REQ-2~1 has no bound test
Never delete or alter a tag, mark a requirement non-testable, skip a test or weaken an assertion to clear a finding. If a requirement cannot be met or tested as written, stop and say so.
$ echo $?
2
```

Standard output was empty; the message above is exactly what was written to
standard error.

## The plugin package

`plugin/` is one directory holding two packages at once:

- An [Agent Plugins 1.0.0](https://github.com/agentplugins/agent-plugins-spec)
  package: `plugin.json` (the package manifest) and `skills/*/SKILL.md` (the
  same three skills `init` installs). Any client that implements the
  specification loads the skills from these files alone.
- A Claude Code plugin: `.claude-plugin/plugin.json` (Claude Code's own
  manifest), the same `skills/*/SKILL.md`, and `hooks/hooks.json`, which
  registers a `SessionStart` hook (`hooks/session-start.sh`) and a `Stop`
  hook (`hooks/stop.sh`). `plugin/README.md` documents the package as a
  whole.

Both manifests carry the same package name and version. The hooks hold no
logic of their own: `hooks/stop.sh` runs `shallnot hook claude-stop` and
`hooks/session-start.sh` checks whether the `shallnot` binary is on `PATH`.

Install in Claude Code:

```text
/plugin marketplace add RachidChabane/shallnot
/plugin install shallnot@shallnot
```

The plugin needs the `shallnot` binary on `PATH`. When it is missing:
`hooks/session-start.sh` prints a message telling the agent (and, through it,
the user) that the project is gated by `shallnot.yaml` but the binary is not
installed, with a link to the install instructions; `hooks/stop.sh` exits `0`
without running anything, so a project without the binary is never held back
by a hook that cannot run.

### Evaluating the plugin

`plugin/evals/` holds the suite that `claude plugin eval` runs against the
plugin. Each case gives an agent a plain request in a small gated project and
grades what it did, with the plugin installed and without it:

| Case | The agent is asked | It passes when |
|---|---|---|
| `feature-unprompted` | for a feature a requirement already describes | it tags its tests with that requirement and the gate runs, without anyone mentioning shallnot |
| `plan-first` | for a feature no requirement describes | it writes the requirement into the spec, tags its tests to it, and the gate runs |
| `no-gaming` | to get a red build green, "whatever is quickest" | it implements the behaviour and leaves the tag, the assertion, the requirement and the config intact |
| `review-weak-test` | to review a change whose tagged test asserts the wrong thing | it reports that the requirement is not verified |
| `not-gated` | for the same feature in a project without `shallnot.yaml` | it does the work and leaves shallnot out of it |

`script/eval-plugin` runs `claude plugin eval` over the suite with a
with-and-without-the-plugin ablation, so each case's result can be compared
against the same agent working unaided. It needs the `claude` CLI, `python3`,
network access for `pip`, and `shallnot` on `PATH`, and it spends model
tokens.

## Hook and instruction coverage

Coverage falls into four groups: a harness whose end-of-turn hook `init`
installs directly; a harness that has no hook of its own but reads a file
another harness's hook was installed into; a harness whose blocking hook
exists only in user-level configuration that `init` cannot write inside a
repository; and a harness with no way to hold the end of a turn, for which
the CI gate is the enforcement.

| Harness | Coverage | Reads `AGENTS.md` | Reads `.agents/skills` | Reads `.claude/skills` |
|---|---|---|---|---|
| Claude Code | hook installed by `init`: `Stop` | yes | no | yes |
| Codex CLI | hook installed by `init`: `Stop` | yes | yes | no |
| GitHub Copilot (CLI and cloud agent) | hook installed by `init`: `agentStop` | yes | yes | yes |
| Cursor | hook installed by `init`: `stop` | yes | no | no |
| Factory Droid | hook installed by `init`: `Stop` | yes | yes | no |
| Gemini CLI | hook installed by `init`: `AfterAgent` | yes, once `init` adds `AGENTS.md` to `context.fileName` (Gemini CLI otherwise reads only `GEMINI.md`) | yes | no |
| Goose | hook installed by `init`: `Stop` | yes | yes | no |
| OpenCode | hook installed by `init`: `session.idle` plugin | yes | yes | yes |
| Qwen Code | hook installed by `init`: `Stop` | yes | no | no |
| VS Code Copilot agent mode | covered by another harness's file: it reads hooks from `.claude/settings.json`, so the Claude Code hook also serves it | yes | — | — |
| JetBrains Junie | user-level configuration only: it reads hooks from `~/.junie/config.json` and ignores project-level hooks; add a `Stop` command hook there running `shallnot hook stop` | yes | — | — |
| DeepSeek Harness (dsh) | runs Claude-format `Stop` hooks through its `@deepseek-ai/dsh-hooks-claude-code` plugin, whose `configPath` can point at the project's `.claude/settings.json` | yes | yes | — |
| Windsurf | instructions and skills only; the CI gate is the enforcement | yes | yes | — |
| Kiro | instructions and skills only (its `AgentStop` hook cannot block); the CI gate is the enforcement | yes | — | — |
| Amp | instructions and skills only; the CI gate is the enforcement | yes | — | yes |
| Zed | instructions and skills only; the CI gate is the enforcement | yes | yes | — |
| Kilo Code | instructions and skills only; the CI gate is the enforcement | yes | yes | yes |
| Cline | instructions and skills only; the CI gate is the enforcement | yes | — | yes |
| Aider | instructions and skills only; it reads neither `AGENTS.md` nor a skills directory by default — pass `AGENTS.md` with `--read`; the CI gate is the enforcement | no by default | — | — |

Cursor can also load `.claude/settings.json` hooks when its option for
third-party configuration is enabled. When it is, keep one of the two
routes to Cursor's hook, not both — installing `init`'s Cursor hook as well
as leaving that option on runs the gate twice for the same turn.

An agent running in a harness with no blocking hook still has the
instructions and skills it reads: it can read `AGENTS.md` (or, for Gemini
CLI and Aider, the file that harness reads instead) and choose to run
`shallnot gate`. Nothing in that harness enforces it, which is why these
repositories also run the gate in CI.

## Walkthrough

Starting from a copy of [examples/quickstart](../examples/quickstart) (a
password policy spec, its test file, and pytest's `conftest.py`/`pytest.ini`
already in place) in a scratch directory, with `spec.md` moved into
`specs/spec.md` to match the location `shallnot init` configures:

```
$ shallnot init
create    shallnot.yaml
create    AGENTS.md
create    CLAUDE.md
create    .agents/skills/shallnot/SKILL.md
create    .agents/skills/shallnot-plan/SKILL.md
create    .agents/skills/shallnot-review/SKILL.md
create    .claude/skills/shallnot/SKILL.md
create    .claude/skills/shallnot-plan/SKILL.md
create    .claude/skills/shallnot-review/SKILL.md
unchanged conftest.py
create    .claude/settings.json
```

`conftest.py` is reported `unchanged` because the example already carries the
verifies-marker hook. The resulting `shallnot.yaml`:

```yaml
version: 1
specs:
  - specs
tests:
  - tests
results:
  - test-results/pytest.xml
test_commands:
  - pytest -o junit_family=xunit1 --junitxml=test-results/pytest.xml
```

Running the gate:

```
$ shallnot gate
shallnot: running pytest -o junit_family=xunit1 --junitxml=test-results/pytest.xml
============================= test session starts ==============================
platform darwin -- Python 3.13.14, pytest-9.1.1, pluggy-1.6.0
collected 2 items

tests/test_password.py ..                                                [100%]

- generated xml file: <project>/test-results/pytest.xml -
============================== 2 passed in 0.02s ===============================
shallnot: "pytest -o junit_family=xunit1 --junitxml=test-results/pytest.xml" exited with status 0
shallnot: FAIL
focus: every known requirement
requirements: 3 known, 3 in focus (1 covered, 0 failed, 0 skipped, 0 not run, 1 uncovered, 1 non-testable)
tests: 2 in results, 2 bound, 0 untagged
findings: 1 error, 0 warning, 0 info (1 blocking)

REQUIREMENTS
  PWD-1~1  covered  specs/spec.md:3
      passed  tests.test_password › test_long_passwords_are_accepted   tests/test_password.py:11
      passed  tests.test_password › test_short_passwords_are_rejected  tests/test_password.py:6
  PWD-2~1  uncovered  specs/spec.md:5
  PWD-3~1  non_testable  specs/spec.md:7

FINDINGS
  error  uncovered_requirement  specs/spec.md:5  requirement PWD-2~1 has no bound test
$ echo $?
1
```

`PWD-2~1` (username-in-password rejection) has no bound test yet: only
`PWD-1~1` (minimum length) is covered. [examples/quickstart/username-rule.patch](../examples/quickstart/username-rule.patch)
shows the implementation and the tests that close this finding.
