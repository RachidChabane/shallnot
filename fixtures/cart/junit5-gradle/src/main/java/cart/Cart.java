package cart;

import java.math.BigDecimal;
import java.math.RoundingMode;
import java.util.ArrayList;
import java.util.List;

public final class Cart {

    private final List<CartItem> items = new ArrayList<>();

    public void addItem(String name, BigDecimal unitPrice, int quantity) {
        items.add(new CartItem(name, unitPrice, quantity));
    }

    public BigDecimal subtotal() {
        BigDecimal total = BigDecimal.ZERO;
        for (CartItem item : items) {
            total = total.add(item.lineTotal());
        }
        return total;
    }

    public BigDecimal applyDiscount(BigDecimal percentage) {
        BigDecimal factor = BigDecimal.ONE.subtract(percentage.movePointLeft(2));
        return subtotal().multiply(factor).setScale(2, RoundingMode.HALF_UP);
    }

    public BigDecimal checkoutTotal() {
        if (items.isEmpty()) {
            return BigDecimal.ZERO;
        }
        return subtotal().setScale(2, RoundingMode.HALF_UP);
    }
}
