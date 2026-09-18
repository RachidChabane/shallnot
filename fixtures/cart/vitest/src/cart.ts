export interface LineItem {
  price: number;
  quantity: number;
}

const MAX_QUANTITY = 99;

export function roundHalfUp(value: number, decimals = 2): number {
  const factor = 10 ** decimals;
  return Math.round(value * factor) / factor;
}

export function subtotal(items: LineItem[]): number {
  return items.reduce((acc, item) => acc + item.price * item.quantity, 0);
}

export function calculateTotal(items: LineItem[]): number {
  return roundHalfUp(subtotal(items));
}

export function applyDiscount(total: number, percentage: number): number {
  return roundHalfUp(total * (1 - percentage / 100));
}

export function validateQuantity(quantity: number): void {
  if (quantity > MAX_QUANTITY) {
    throw new Error(`quantity ${quantity} exceeds the maximum of ${MAX_QUANTITY}`);
  }
}

export function checkout(items: LineItem[]): number {
  return calculateTotal(items);
}

export function applyGiftWrap(total: number, fee: number): number {
  return roundHalfUp(total + fee);
}
