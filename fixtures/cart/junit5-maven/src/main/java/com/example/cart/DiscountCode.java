package com.example.cart;

import java.math.BigDecimal;

public record DiscountCode(String code, BigDecimal percentage) {

    public BigDecimal applyTo(BigDecimal amount) {
        BigDecimal factor = BigDecimal.ONE.subtract(percentage.movePointLeft(2));
        return amount.multiply(factor);
    }
}
