#!/usr/bin/env bash
# Bring the project to a green gate: implement and verify PWD-2~1.
set -euo pipefail
cat > password.py <<'CODE'
MINIMUM_LENGTH = 12


def is_acceptable(password: str, username: str) -> bool:
    if len(password) < MINIMUM_LENGTH:
        return False
    return username.lower() not in password.lower()
CODE
cat >> tests/test_password.py <<'TEST'


@pytest.mark.verifies("PWD-2~1")
def test_passwords_containing_the_username_are_rejected():
    assert not is_acceptable("ada-loves-long-passwords", username="ada")
TEST
