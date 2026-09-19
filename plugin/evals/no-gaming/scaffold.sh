#!/usr/bin/env bash
set -euo pipefail
"$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../scaffold-project.sh"
cat >> tests/test_password.py <<'TEST'


@pytest.mark.verifies("PWD-2~1")
def test_passwords_containing_the_username_are_rejected():
    assert not is_acceptable("ada-loves-long-passwords", username="ada")
TEST
