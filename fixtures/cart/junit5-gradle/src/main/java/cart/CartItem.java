package cart;

import java.math.BigDecimal;

public final class CartItem {

    private final String name;
    private final BigDecimal unitPrice;
    private final int quantity;

    public CartItem(String name, BigDecimal unitPrice, int quantity) {
        if (quantity > 99) {
            throw new IllegalArgumentException("quantity exceeds the maximum allowed per line item");
        }
        this.name = name;
        this.unitPrice = unitPrice;
        this.quantity = quantity;
    }

    public String getName() {
        return name;
    }

    public BigDecimal lineTotal() {
        return unitPrice.multiply(BigDecimal.valueOf(quantity));
    }
}
