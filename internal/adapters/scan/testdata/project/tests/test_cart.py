import pytest

pytestmark = pytest.mark.verifies("REQ-9~1")


@pytest.mark.verifies("REQ-1~1")
def test_total_sums_items():
    assert True


@pytest.mark.parametrize("code", ["A", "B"])
@pytest.mark.verifies(
    "REQ-2~2",
    "REQ-3~1",
)
async def test_discount(code):
    assert code


def test_checkout(record_property):
    record_property("verifies", "REQ-4~1, REQ-5~1")
    assert True


@pytest.mark.verifies("REQ-6~1")
class TestRounding:
    def test_half_up(self):
        assert True


# @pytest.mark.verifies("REQ-7~1")
def test_disabled_marker():
    assert True


@pytest.mark.verifies(REQUIREMENT)
def test_variable_argument():
    assert True


@pytest.mark.verifies("REQ-8")
def test_no_revision():
    assert True
