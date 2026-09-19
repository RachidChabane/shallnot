#!/usr/bin/env bash
# Build the project a recorded agent session starts from: the quickstart
# project with PWD-2~1 untested, equipped by `shallnot init`.
# Usage: session-prep.sh [directory]   (default: ~/.cache/shallnot-demo/password-policy)
set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
target="${1:-$HOME/.cache/shallnot-demo/password-policy}"

rm -rf "$target"
mkdir -p "$target/specs" "$target/.claude"
cp -R "$repo/examples/quickstart/tests" "$repo/examples/quickstart/password.py" "$target/"
cp "$repo/examples/quickstart/spec.md" "$target/specs/password-policy.md"
cat > "$target/.claude/settings.json" <<'SETTINGS'
{
  "permissions": {
    "allow": ["Bash", "Edit", "Write", "Read"]
  }
}
SETTINGS
touch "$target/pytest.ini"
"$repo/bin/shallnot" init --dir "$target" > /dev/null
find "$target" -name __pycache__ -prune -exec rm -rf {} +
echo "$target"
