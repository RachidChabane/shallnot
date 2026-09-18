import pytest

from cart import Cart


@pytest.mark.verifies("CART-5~1")
def test_quantity_above_limit_is_rejected():
    cart = Cart()
    with pytest.raises(ValueError):
        cart.add_item("crate", "1.00", 150)
