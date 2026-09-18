function calculateSubtotal(item) {
  return item.price * item.quantity;
}

function calculateTotal(items) {
  return items.reduce((sum, item) => sum + calculateSubtotal(item), 0);
}

function applyDiscount(total, percentOff) {
  return total - (total * percentOff) / 100;
}

function applyGiftWrap(total, feePerItem) {
  return total + feePerItem;
}

function checkout(cart) {
  if (cart.items.length === 0) {
    return 0;
  }
  return calculateTotal(cart.items);
}

function roundHalfUp(value) {
  return Math.round(value * 100) / 100;
}

function validateQuantity(quantity) {
  if (quantity > 99) {
    throw new Error("Quantity exceeds the maximum allowed per line item");
  }
  return quantity;
}

module.exports = {
  calculateSubtotal,
  calculateTotal,
  applyDiscount,
  applyGiftWrap,
  checkout,
  roundHalfUp,
  validateQuantity,
};
