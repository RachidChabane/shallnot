#!/usr/bin/env bash
# End-of-turn gate. Does nothing when the shallnot binary is not installed:
# session-start.sh has already told the agent how to install it.
command -v shallnot > /dev/null 2>&1 || exit 0
# A project equipped by `shallnot init` carries the same hook itself: gate once.
project="${CLAUDE_PROJECT_DIR:-$PWD}"
grep -q "shallnot hook claude-stop" "$project/.claude/settings.json" 2> /dev/null && exit 0
exec shallnot hook claude-stop
