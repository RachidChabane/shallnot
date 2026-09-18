package com.example.cart;

import java.math.BigDecimal;
import java.math.RoundingMode;
import java.util.ArrayList;
import java.util.List;

public class Cart {

    private final List<LineItem> items = new ArrayList<>();
    private DiscountCode discountCode;

    public void addItem(LineItem item) {
        items.add(item);
    }

    public void applyDiscount(DiscountCode discountCode) {
        this.discountCode = discountCode;
    }

    public BigDecimal total() {
        BigDecimal subtotal = items.stream()
                .map(LineItem::lineTotal)
                .reduce(BigDecimal.ZERO, BigDecimal::add);

        if (discountCode != null) {
            subtotal = discountCode.applyTo(subtotal);
        }

        return subtotal.setScale(2, RoundingMode.HALF_UP);
    }

    public BigDecimal checkout() {
        return total();
    }
}
