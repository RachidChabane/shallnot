package cart;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import java.math.BigDecimal;

import static org.junit.jupiter.api.Assertions.assertThrows;

class BulkOrderTest {

    @Test
    @DisplayName("rejects a line item quantity above the per-order limit [verifies CART-5~1]")
    void rejectsQuantityAboveLimit() {
        Cart cart = new Cart();

        assertThrows(IllegalArgumentException.class, () -> cart.addItem("crate", new BigDecimal("2.00"), 100));
    }
}
