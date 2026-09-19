# shallnot plugin

This directory is a plugin package for coding agents. It carries the shallnot
skill, which teaches an agent to bind tests to requirements and to pass the
gate before it reports work as done, without the user asking for it.

It is two packages in one directory:

- An [Agent Plugins 1.0.0](https://github.com/agentplugins/agent-plugins-spec)
  package: `plugin.json` and `skills/shallnot/SKILL.md`. Any client that
  implements the specification loads the skill.
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

A repository can also carry the same skill and hook itself, for every agent
and every teammate, with `shallnot init`; see
[docs/agents.md](../docs/agents.md).
