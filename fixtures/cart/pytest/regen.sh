#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

if [ ! -d .venv ]; then
    uv venv .venv
fi
uv pip install -r requirements.txt -p .venv

mkdir -p results

.venv/bin/python -m pytest -o junit_family=xunit1 --junitxml=results/junit.xml -q || true
.venv/bin/python -m pytest -o junit_family=xunit2 --junitxml=results/junit-xunit2.xml -q || true

"$SCRIPT_DIR/../sanitize-results.sh" "$SCRIPT_DIR" results/junit.xml results/junit-xunit2.xml
