#!/usr/bin/env bash
set -euo pipefail
"$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../scaffold-project.sh"
cat > password.py <<'CODE'
MINIMUM_LENGTH = 12


def is_acceptable(password: str, username: str) -> bool:
    if len(password) < MINIMUM_LENGTH:
        return False
    return username.lower() not in password.lower()
CODE
cat >> tests/test_password.py <<'TEST'


@pytest.mark.verifies("PWD-2~1")
def test_username_rule():
    assert is_acceptable("correct horse battery", username="ada")
TEST
