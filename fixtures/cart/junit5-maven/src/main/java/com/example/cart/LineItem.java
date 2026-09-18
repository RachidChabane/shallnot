package com.example.cart;

import java.math.BigDecimal;

public record LineItem(String sku, BigDecimal unitPrice, int quantity) {

    public static final int MAX_QUANTITY = 99;

    public LineItem {
        if (quantity > MAX_QUANTITY) {
            throw new IllegalArgumentException("quantity above " + MAX_QUANTITY + " is not allowed: " + quantity);
        }
    }

    public BigDecimal lineTotal() {
        return unitPrice.multiply(BigDecimal.valueOf(quantity));
    }
}
