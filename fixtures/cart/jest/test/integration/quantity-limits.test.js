const { validateQuantity } = require("../../src/cart");

it("rejects a line item quantity above 99 [verifies CART-5~1]", () => {
  expect(() => validateQuantity(100)).toThrow();
});
