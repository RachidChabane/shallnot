#!/usr/bin/env bash
# Build the project of the "catch" recording: the quickstart project with
# PWD-2~1 untested, gated by the end-of-turn hook, and no agent instructions.
# The agent in it knows nothing about shallnot; only the hook does.
# Usage: catch-prep.sh [directory]   (default: ~/.cache/shallnot-demo/catch)
set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
target="${1:-$HOME/.cache/shallnot-demo/catch}"

rm -rf "$target"
mkdir -p "$target/specs" "$target/.claude"
cp -R "$repo/examples/quickstart/tests" "$repo/examples/quickstart/password.py" \
  "$repo/examples/quickstart/conftest.py" "$repo/examples/quickstart/pytest.ini" "$target/"
cp "$repo/examples/quickstart/spec.md" "$target/specs/password-policy.md"
cat > "$target/shallnot.yaml" <<'CONFIG'
version: 1
id_pattern: 'PWD-[0-9]+'
specs:
  - specs
tests:
  - tests
results:
  - test-results/pytest.xml
test_commands:
  - pytest -q -o junit_family=xunit1 --junitxml=test-results/pytest.xml
CONFIG
cat > "$target/.claude/settings.json" <<'SETTINGS'
{
  "verbose": true,
  "enabledPlugins": { "pyright-lsp@claude-plugins-official": true },
  "permissions": { "allow": ["Bash", "Edit", "Write", "Read"] },
  "hooks": {
    "Stop": [
      { "hooks": [ { "type": "command", "command": "shallnot hook claude-stop", "timeout": 600 } ] }
    ]
  }
}
SETTINGS
find "$target" -name __pycache__ -prune -exec rm -rf {} +
echo "$target"
