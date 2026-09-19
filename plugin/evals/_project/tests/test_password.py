import pytest

from password import is_acceptable


@pytest.mark.verifies("PWD-1~1")
def test_short_passwords_are_rejected():
    assert not is_acceptable("short", username="ada")


@pytest.mark.verifies("PWD-1~1")
def test_long_passwords_are_accepted():
    assert is_acceptable("correct horse battery", username="ada")
