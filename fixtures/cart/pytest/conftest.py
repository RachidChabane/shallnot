def pytest_collection_modifyitems(items):
    for item in items:
        for marker in item.iter_markers(name="verifies"):
            item.user_properties.append(("verifies", ", ".join(marker.args)))
