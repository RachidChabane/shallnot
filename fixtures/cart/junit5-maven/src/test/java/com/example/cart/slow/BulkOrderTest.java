package com.example.cart.slow;

import com.example.cart.Cart;
import com.example.cart.LineItem;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import java.math.BigDecimal;

import static org.junit.jupiter.api.Assertions.assertThrows;

class BulkOrderTest {

    @Test
    @DisplayName("a line item quantity above 99 is rejected [verifies CART-5~1]")
    void quantityAboveNinetyNineIsRejected() {
        assertThrows(IllegalArgumentException.class,
                () -> new LineItem("pallet", new BigDecimal("1.00"), 150));
    }
}
