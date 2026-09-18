import pytest

from cart import Cart


@pytest.mark.verifies("CART-1~1")
def test_total_sums_price_times_quantity():
    cart = Cart()
    cart.add_item("mug", "9.50", 3)
    cart.add_item("plate", "4.00", 2)
    assert cart.total() == pytest.approx(36.50)


def test_total_reflects_each_line_item(record_property):
    record_property("verifies", "CART-1~1")
    cart = Cart()
    cart.add_item("candle", "12.00", 1)
    cart.add_item("vase", "18.25", 2)
    assert cart.total() == pytest.approx(48.50)


def test_total_with_no_discount_matches_subtotal():
    cart = Cart()
    cart.add_item("towel", "7.00", 4)
    assert cart.total() == pytest.approx(28.00)
