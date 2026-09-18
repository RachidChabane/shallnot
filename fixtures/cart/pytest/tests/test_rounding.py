import pytest

from cart import Cart


@pytest.mark.skip(reason="rounding mode pending finance decision")
@pytest.mark.verifies("CART-4~1")
def test_total_rounds_half_up_to_two_decimals():
    cart = Cart()
    cart.add_item("item", "3.335", 1)
    assert cart.total() == pytest.approx(3.34)
