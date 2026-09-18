import pytest

from cart import Cart


@pytest.mark.verifies("CART-3~1")
def test_checkout_rejects_empty_cart():
    cart = Cart()
    with pytest.raises(ValueError):
        cart.checkout()
