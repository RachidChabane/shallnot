#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

./gradlew test --no-daemon || true

rm -rf results
mkdir -p results

cp build/test-results/test/*.xml results/

"$SCRIPT_DIR/../sanitize-results.sh" "$SCRIPT_DIR" results/*.xml
