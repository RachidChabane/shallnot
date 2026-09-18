#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$script_dir"

mvn -B -q dependency:resolve dependency:resolve-plugins || true

set +e
mvn -B test
mvn_exit=$?
set -e

if [ "$mvn_exit" -ne 0 ] && [ ! -d target/surefire-reports ]; then
  echo "mvn test failed before producing any surefire reports" >&2
  exit "$mvn_exit"
fi

rm -rf results
mkdir -p results

cp target/surefire-reports/TEST-*.xml results/

"$script_dir/../sanitize-results.sh" "$script_dir" results/TEST-*.xml
