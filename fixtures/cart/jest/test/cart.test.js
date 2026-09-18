const {
  calculateSubtotal,
  calculateTotal,
  applyDiscount,
  applyGiftWrap,
  checkout,
  roundHalfUp,
  validateQuantity,
} = require("../src/cart");

it("sums line items [verifies CART-1~1]", () => {
  const items = [
    { price: 10, quantity: 2 },
    { price: 5, quantity: 3 },
  ];
  expect(calculateTotal(items)).toBe(35);
});

describe("discount codes", () => {
  it.each([
    [10, 100, 90],
    [20, 50, 40],
    [25, 200, 150],
  ])(
    "applies %i%% off a %i total to get %i [verifies CART-2~2]",
    (percentOff, total, expected) => {
      expect(applyDiscount(total, percentOff)).toBe(expected);
    }
  );

  it("applies a flat 5% welcome code [verifies CART-2~1]", () => {
    expect(applyDiscount(100, 5)).toBe(95);
  });
});

it("rejects checking out an empty cart [verifies CART-3~1]", () => {
  const emptyCart = { items: [] };
  expect(() => checkout(emptyCart)).toThrow();
});

// Rounding mode pending finance decision; revisit once accounting confirms
// the display precision for fractional cents.
it.skip("rounds totals to two decimals using half-up [verifies CART-4~1]", () => {
  expect(roundHalfUp(19.995)).toBe(20.0);
  expect(roundHalfUp(19.994)).toBe(19.99);
});

it("adds a gift wrap fee to the order total [verifies CART-99~1]", () => {
  expect(applyGiftWrap(50, 2)).toBe(52);
});

it("computes the subtotal for a single line item [verifies CART-1]", () => {
  expect(calculateSubtotal({ price: 7.5, quantity: 4 })).toBe(30);
});

it("keeps quantities editable after adding an item", () => {
  const items = [{ price: 12, quantity: 1 }];
  items[0].quantity = 2;
  expect(calculateTotal(items)).toBe(24);
});

it("stays consistent between total and discounted total [verifies CART-1~1, CART-2~2]", () => {
  const items = [
    { price: 10, quantity: 1 },
    { price: 20, quantity: 1 },
  ];
  const total = calculateTotal(items);
  const discounted = applyDiscount(total, 10);
  expect(discounted).toBe(total - 3);
});

describe("discount codes [verifies CART-2~2]", () => {
  describe("percentage codes", () => {
    it("applies a 10% code to a 100 total", () => {
      expect(applyDiscount(100, 10)).toBe(90);
    });

    it("applies a 20% code to a 50 total", () => {
      expect(applyDiscount(50, 20)).toBe(40);
    });
  });
});
