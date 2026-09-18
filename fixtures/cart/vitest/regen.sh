#!/usr/bin/env bash
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$DIR"

npm ci

set +e
npx vitest run
set -e

if [ ! -f "results/junit.xml" ]; then
  echo "results/junit.xml was not produced" >&2
  exit 1
fi

"$DIR/../sanitize-results.sh" "$DIR" results/junit.xml
