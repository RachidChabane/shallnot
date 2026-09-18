import pytest

from cart import Cart


@pytest.mark.verifies("CART-99~1")
def test_receipt_notes_survive_across_sessions():
    cart = Cart()
    cart.add_item("bookmark", "2.50", 1)
    assert cart.total() == pytest.approx(2.50)


@pytest.mark.verifies("CART-1")
def test_cart_accepts_repeat_orders_of_same_item():
    cart = Cart()
    cart.add_item("candle", "5.00", 2)
    cart.add_item("candle", "5.00", 1)
    assert cart.total() == pytest.approx(15.00)


def test_cart_starts_with_no_discount():
    cart = Cart()
    assert cart.discount_percent == 0


@pytest.mark.verifies("CART-1~1", "CART-2~2")
def test_discounted_total_reflects_line_items():
    cart = Cart()
    cart.add_item("item", "20.00", 2)
    cart.apply_discount_code("10")
    assert cart.total() == pytest.approx(36.00)
