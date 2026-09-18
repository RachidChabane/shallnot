from decimal import ROUND_HALF_UP, Decimal

MAX_LINE_ITEM_QUANTITY = 99


class LineItem:
    def __init__(self, name, unit_price, quantity):
        if quantity > MAX_LINE_ITEM_QUANTITY:
            raise ValueError(
                f"quantity {quantity} exceeds the maximum of {MAX_LINE_ITEM_QUANTITY}"
            )
        self.name = name
        self.unit_price = Decimal(str(unit_price))
        self.quantity = quantity

    @property
    def subtotal(self):
        return self.unit_price * self.quantity


class Cart:
    def __init__(self):
        self.items = []
        self.discount_percent = Decimal("0")

    def add_item(self, name, unit_price, quantity):
        self.items.append(LineItem(name, unit_price, quantity))

    def apply_discount_code(self, percent):
        self.discount_percent = Decimal(str(percent))

    def _raw_total(self):
        total = sum((item.subtotal for item in self.items), Decimal("0"))
        discount = total * (self.discount_percent / Decimal("100"))
        return total - discount

    def total(self):
        raw = self._raw_total()
        return raw.quantize(Decimal("0.01"), rounding=ROUND_HALF_UP)

    def checkout(self):
        if not self.items:
            return Decimal("0")
        return self.total()
