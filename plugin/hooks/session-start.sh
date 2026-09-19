#!/usr/bin/env bash
# In a gated project without the shallnot binary, tell the agent how to get it.
project="${CLAUDE_PROJECT_DIR:-$PWD}"
[ -f "$project/shallnot.yaml" ] || exit 0
command -v shallnot > /dev/null 2>&1 && exit 0
echo "This project is gated by shallnot (shallnot.yaml), but the shallnot CLI is not on PATH, so the gate cannot run. Tell the user to install it: https://github.com/RachidChabane/shallnot#install"
