package cart;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Disabled;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.CsvSource;

import java.math.BigDecimal;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

class CartTotalTest {

    @Test
    @DisplayName("sums line item prices times quantities [verifies CART-1~1]")
    void sumsLinePricesTimesQuantities() {
        Cart cart = new Cart();
        cart.addItem("mug", new BigDecimal("9.50"), 2);
        cart.addItem("plate", new BigDecimal("4.25"), 3);

        assertEquals(new BigDecimal("31.75"), cart.checkoutTotal());
    }

    @ParameterizedTest(name = "{0}% off leaves {2}")
    @CsvSource({
            "10, 100.00, 90.00",
            "25, 40.00, 30.00",
            "50, 19.98, 9.99"
    })
    @DisplayName("applies a percentage discount code [verifies CART-2~2]")
    void appliesPercentageDiscount(int percent, BigDecimal subtotal, BigDecimal expected) {
        Cart cart = new Cart();
        cart.addItem("bulk item", subtotal, 1);

        assertEquals(expected, cart.applyDiscount(BigDecimal.valueOf(percent)));
    }

    @Test
    @DisplayName("applies the legacy discount code format [verifies CART-2~1]")
    void appliesLegacyDiscountFormat() {
        Cart cart = new Cart();
        cart.addItem("bulk item", new BigDecimal("50.00"), 1);

        assertEquals(new BigDecimal("45.00"), cart.applyDiscount(BigDecimal.valueOf(10)));
    }

    @Test
    @DisplayName("rejects checkout of an empty cart [verifies CART-3~1]")
    void rejectsEmptyCartCheckout() {
        Cart cart = new Cart();

        assertThrows(EmptyCartException.class, cart::checkoutTotal);
    }

    @Test
    @Disabled("rounding mode pending finance decision")
    @DisplayName("rounds totals to two decimals half-up [verifies CART-4~1]")
    void roundsTotalsHalfUp() {
        Cart cart = new Cart();
        cart.addItem("widget", new BigDecimal("3.335"), 1);

        assertEquals(new BigDecimal("3.34"), cart.checkoutTotal());
    }

    @Test
    @DisplayName("keeps totals in whole currency units [verifies CART-99~1]")
    void keepsTotalsInWholeCurrencyUnits() {
        Cart cart = new Cart();
        cart.addItem("token", new BigDecimal("2.00"), 5);

        assertEquals(new BigDecimal("10.00"), cart.checkoutTotal());
    }

    @Test
    @DisplayName("keeps the total stable across repeated reads [verifies CART-1]")
    void keepsTotalStableAcrossRepeatedReads() {
        Cart cart = new Cart();
        cart.addItem("token", new BigDecimal("1.00"), 1);

        BigDecimal first = cart.checkoutTotal();
        BigDecimal second = cart.checkoutTotal();

        assertEquals(first, second);
    }

    @Test
    @DisplayName("treats zero-price items as free")
    void treatsZeroPriceItemsAsFree() {
        Cart cart = new Cart();
        cart.addItem("sample", BigDecimal.ZERO, 1);

        assertEquals(new BigDecimal("0.00"), cart.checkoutTotal());
    }

    @Test
    @DisplayName("combines a base total with a discount code [verifies CART-1~1, CART-2~2]")
    void combinesBaseTotalWithDiscountCode() {
        Cart cart = new Cart();
        cart.addItem("bundle", new BigDecimal("20.00"), 2);

        assertEquals(new BigDecimal("36.00"), cart.applyDiscount(BigDecimal.valueOf(10)));
    }

    @Nested
    @DisplayName("checkout summary [verifies CART-1~1]")
    class CheckoutSummary {

        @Test
        @DisplayName("reports the subtotal before any discount")
        void reportsSubtotalBeforeDiscount() {
            Cart cart = new Cart();
            cart.addItem("charger", new BigDecimal("15.00"), 1);

            assertEquals(new BigDecimal("15.00"), cart.subtotal());
        }
    }
}
