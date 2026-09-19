#!/usr/bin/env bash
# End-of-turn gate. Does nothing when the shallnot binary is not installed:
# session-start.sh has already told the agent how to install it.
command -v shallnot > /dev/null 2>&1 || exit 0
exec shallnot hook claude-stop
