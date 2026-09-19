# ADR 0002: Equipping agents: init, end-of-turn hooks, and a plugin package

Status: accepted

## Context

The person who asks an agent for a feature is not interested in
traceability. If the gate only runs when someone types "run shallnot", it
fails at its purpose. An agent needs two things that a command-line tool alone
does not give it: the knowledge that the repository is gated and what the
convention is, and a mechanism that does not depend on the model remembering.

The project started as a CLI with a Markdown skill and a deliberate absence of
harness-specific plugins, so that the tool would not be tied to one agent
product. That left the skill uninstalled and the gate unenforced. This ADR
replaces that position.

The Agent Plugins specification (1.0.0) defines a vendor-neutral package: a
root `plugin.json`, skills under `skills/<name>/SKILL.md`, MCP servers in
`mcp.json`. It excludes hooks, commands and rules as "too client-specific for a
stable portable contract" and confines them to client-owned extension
directories with no portable semantics.

## Decisions

### Knowledge travels as a skill; enforcement is a hook

The skill is the single source of what an agent must know. Instruction files
are advice a model can skip; a step that must always happen belongs in a
hook. Both are shipped, and the hook repeats nothing: it runs the gate and
hands over the findings.

### `shallnot init` is the primary vehicle, not the plugin

A plugin equips one person's agent. `init` equips the repository, which covers
every teammate, every harness that reads `AGENTS.md`, and unattended pipeline
agents. It writes files and nothing else: a starter `shallnot.yaml` (never
rewritten once it exists), a marked section of `AGENTS.md`, an `@AGENTS.md`
import in `CLAUDE.md`, the project skill, the pytest `conftest.py` hook, and
the end-of-turn hook configuration. It is idempotent and has a `--check` mode,
so a repository can verify in CI that its agent files are current.

### One hook command, one adapter per harness

`shallnot hook <harness>` reads the harness's hook input, gates the project
and answers in the harness's protocol: exit code 2 with the reason on standard
error for Claude Code's `Stop`; a `followup_message` for Cursor's `stop`.
Adapters exist only for protocols verified against the harness's own
documentation. Rules common to all adapters:

- A project without a `shallnot.yaml` is ignored, so a globally installed
  plugin is inert elsewhere.
- Advisory mode never holds the agent.
- A missing verdict (tests did not write their results) is sent back to the
  agent like a blocked gate, with the instruction not to write requirements
  itself when the specs are what is missing.
- After three consecutive blocks the turn is allowed to end and the user is
  told. The count is the harness's own when it provides one, otherwise a
  per-session counter file in the temporary directory.
- The hook's own failures exit 1, never 2: harnesses read 2 as "hold the
  agent", and a broken hook must fail open.

### `shallnot gate` runs the tests; `check` stays read-only

A hook needs fresh results. `test_commands` in the config names how to produce
them; `gate` runs them through the platform shell in the config's directory,
ignores their exit status (failing tests are a result), and then checks. A
results file that the commands left untouched, compared with its state before
the run, is a tool failure: a runner that crashed before writing its report
must not leave an earlier verdict in place. `check` runs nothing and remains
the entry point for pipelines that run tests in another job.

### The plugin is packaging with no logic

`plugin/` is at once an Agent Plugins 1.0.0 package (`plugin.json`, `skills/`)
and a Claude Code plugin (`.claude-plugin/plugin.json`, `hooks/hooks.json`).
The Claude hooks are two shell scripts that call the binary, or do nothing
when it is absent apart from telling the agent how to install it in a gated
project. The skill file in the package is embedded in the binary, so `init`
and the plugin install the same text. The Claude Code hooks sit in Claude's
own layout rather than in an Agent Plugins extension directory because Claude
Code defines where it looks for them.

## Consequences

- The tool is still a CLI that any agent can run; nothing requires a plugin.
- Hook coverage is limited to harnesses with a verified blocking end-of-turn
  hook. Other harnesses get the instructions only.
- The repository dogfoods the mechanism: it is equipped by its own `init`, CI
  runs `shallnot gate` and `shallnot init --check`.
- The skill has one source (`plugin/skills/shallnot/SKILL.md`); the `AGENTS.md`
  section is derived from it at run time, so no generated copy is committed
  except the ones `init` writes into this repository, which `--check` guards.
