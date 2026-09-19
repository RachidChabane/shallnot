def pytest_configure(config):
    config.addinivalue_line("markers", "verifies(*refs): requirements the test verifies, as ID~REVISION")


def pytest_collection_modifyitems(items):
    for item in items:
        for marker in item.iter_markers(name="verifies"):
            item.user_properties.append(("verifies", ", ".join(marker.args)))
