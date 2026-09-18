import pytest

from cart import Cart


@pytest.mark.verifies("CART-2~2")
@pytest.mark.parametrize(
    "unit_price,quantity,discount_percent,expected",
    [
        ("10.00", 2, "10", 18.00),
        ("50.00", 1, "20", 40.00),
        ("15.00", 4, "5", 57.00),
    ],
)
def test_discount_code_reduces_total(unit_price, quantity, discount_percent, expected):
    cart = Cart()
    cart.add_item("item", unit_price, quantity)
    cart.apply_discount_code(discount_percent)
    assert cart.total() == pytest.approx(expected)


@pytest.mark.verifies("CART-2~1")
def test_discount_code_applies_before_rounding():
    cart = Cart()
    cart.add_item("item", "10.00", 3)
    cart.apply_discount_code("10")
    assert cart.total() == pytest.approx(27.00)


@pytest.mark.verifies("CART-2~2")
class TestDiscounts:
    def test_full_discount_zeroes_total(self):
        cart = Cart()
        cart.add_item("item", "20.00", 2)
        cart.apply_discount_code("100")
        assert cart.total() == pytest.approx(0.00)

    def test_partial_discount_on_multiple_items(self):
        cart = Cart()
        cart.add_item("item-a", "10.00", 1)
        cart.add_item("item-b", "10.00", 1)
        cart.apply_discount_code("50")
        assert cart.total() == pytest.approx(10.00)

    def test_zero_discount_leaves_total_unchanged(self):
        cart = Cart()
        cart.add_item("item", "30.00", 1)
        cart.apply_discount_code("0")
        assert cart.total() == pytest.approx(30.00)
