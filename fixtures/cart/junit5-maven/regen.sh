#!/usr/bin/env bash
set -euo pipefail

fixture_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$fixture_dir"

mvn -q -DskipTests=false dependency:go-offline

mvn test || true

reports_dir="target/surefire-reports"
if ! compgen -G "$reports_dir/TEST-*.xml" > /dev/null; then
  echo "no surefire reports were produced" >&2
  exit 1
fi

results_dir="results"
rm -rf "$results_dir"
mkdir -p "$results_dir"
cp "$reports_dir"/TEST-*.xml "$results_dir"/

"$fixture_dir/../sanitize-results.sh" "$fixture_dir" "$results_dir"/TEST-*.xml
