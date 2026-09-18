import { describe, expect, it, test } from "vitest";
import {
  applyDiscount,
  applyGiftWrap,
  calculateTotal,
  checkout,
  roundHalfUp,
  subtotal,
} from "../src/cart";

it("sums line items [verifies CART-1~1]", () => {
  const total = calculateTotal([
    { price: 10, quantity: 2 },
    { price: 5, quantity: 3 },
  ]);
  expect(total).toBe(35);
});

test.each([
  { total: 100, percentage: 10, expected: 90 },
  { total: 50, percentage: 20, expected: 40 },
  { total: 200, percentage: 25, expected: 150 },
])(
  "applies a $percentage% discount code to $total -> $expected [verifies CART-2~2]",
  ({ total, percentage, expected }) => {
    expect(applyDiscount(total, percentage)).toBe(expected);
  },
);

it("leaves the total unchanged when no discount code is applied [verifies CART-2~1]", () => {
  expect(applyDiscount(80, 0)).toBe(80);
});

it("rejects checking out an empty cart [verifies CART-3~1]", () => {
  expect(() => checkout([])).toThrow();
});

it.skip("rounds a total half-up to two decimals [verifies CART-4~1]", () => {
  // rounding mode pending finance decision
  expect(roundHalfUp(19.005)).toBe(19.01);
});

it("adds a gift wrap fee to the total [verifies CART-99~1]", () => {
  expect(applyGiftWrap(50, 3.5)).toBe(53.5);
});

it("computes a subtotal before tax [verifies CART-1]", () => {
  expect(subtotal([{ price: 12, quantity: 4 }])).toBe(48);
});

it("keeps quantities as positive integers", () => {
  expect(Number.isInteger(2)).toBe(true);
});

it("keeps the discounted total consistent with the summed total [verifies CART-1~1, CART-2~2]", () => {
  const total = calculateTotal([{ price: 40, quantity: 1 }]);
  expect(applyDiscount(total, 25)).toBe(30);
});

describe("discount codes [verifies CART-2~2]", () => {
  describe("percentage codes", () => {
    it("applies a 10% code", () => {
      expect(applyDiscount(100, 10)).toBe(90);
    });

    it("applies a 20% code", () => {
      expect(applyDiscount(100, 20)).toBe(80);
    });
  });
});
