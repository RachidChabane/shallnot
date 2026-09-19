#!/usr/bin/env bash
set -euo pipefail
"$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../scaffold-project.sh"
rm -rf shallnot.yaml specs conftest.py
sed -i.bak '/pytest.mark.verifies/d' tests/test_password.py && rm -f tests/test_password.py.bak
