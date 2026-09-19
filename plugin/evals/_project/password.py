MINIMUM_LENGTH = 12


def is_acceptable(password: str, username: str) -> bool:
    return len(password) >= MINIMUM_LENGTH
