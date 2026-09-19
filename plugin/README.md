# shallnot plugin

This directory is a plugin package for coding agents. It carries three skills
that make an agent do requirement-to-test traceability without the user asking
for it:

| Skill | Teaches the agent to |
|---|---|
| `shallnot-plan` | state a requested behaviour as a requirement, with an ID and a revision, before building it |
| `shallnot` | bind the tests it writes to requirements, run the gate before reporting work as done, and react to each finding |
| `shallnot-review` | judge whether a tagged test really verifies its requirement, which the gate cannot do |

It is two packages in one directory:

- An [Agent Plugins 1.0.0](https://github.com/agentplugins/agent-plugins-spec)
  package: `plugin.json` and `skills/*/SKILL.md`. Any client that implements
  the specification loads the skills.
- A Claude Code plugin: `.claude-plugin/plugin.json`, the same `skills/`, and
  `hooks/hooks.json`, whose Stop hook runs `shallnot hook claude-stop` and
  sends Claude back to work while the gate is blocked.

The plugin holds no logic. Everything it does goes through the `shallnot`
binary, which must be on `PATH`
([install](https://github.com/RachidChabane/shallnot#install)).

Install in Claude Code:

```text
/plugin marketplace add RachidChabane/shallnot
/plugin install shallnot@shallnot
```

[`evals/`](evals) holds the suite that `claude plugin eval` runs against the
plugin, with and without it.

A repository can also carry the same skills and hook itself, for every agent
and every teammate, with `shallnot init`; see
[docs/agents.md](../docs/agents.md).
