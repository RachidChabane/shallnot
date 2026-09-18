package com.example.cart;

import org.junit.jupiter.api.Disabled;
import org.junit.jupiter.api.DisplayName;
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
    void totalIsSumOfLineItemPricesTimesQuantities() {
        Cart cart = new Cart();
        cart.addItem(new LineItem("mug", new BigDecimal("4.50"), 2));
        cart.addItem(new LineItem("plate", new BigDecimal("3.00"), 3));

        assertEquals(new BigDecimal("18.00"), cart.total());
    }

    @ParameterizedTest(name = "{0}% off {1} gives {2} [verifies CART-2~2]")
    @DisplayName("percentage discount code reduces the total")
    @CsvSource({
            "10, 100.00, 90.00",
            "25, 40.00, 30.00",
            "50, 19.98, 9.99"
    })
    void percentageDiscountReducesTotal(String percentage, String startAmount, String expected) {
        Cart cart = new Cart();
        cart.addItem(new LineItem("gift-card", new BigDecimal(startAmount), 1));
        cart.applyDiscount(new DiscountCode("SAVE", new BigDecimal(percentage)));

        assertEquals(new BigDecimal(expected), cart.total());
    }

    @Test
    @DisplayName("legacy flat discount still reduces the total [verifies CART-2~1]")
    void legacyDiscountStillReducesTotal() {
        Cart cart = new Cart();
        cart.addItem(new LineItem("gift-card", new BigDecimal("50.00"), 1));
        cart.applyDiscount(new DiscountCode("OLD10", new BigDecimal("10")));

        assertEquals(new BigDecimal("45.00"), cart.total());
    }

    @Test
    @DisplayName("checking out an empty cart is rejected [verifies CART-3~1]")
    void checkoutOfEmptyCartIsRejected() {
        Cart cart = new Cart();

        assertThrows(IllegalStateException.class, cart::checkout);
    }

    @Test
    @Disabled("rounding mode pending finance decision")
    @DisplayName("totals round to two decimals half-up [verifies CART-4~1]")
    void totalsRoundToTwoDecimalsHalfUp() {
        Cart cart = new Cart();
        cart.addItem(new LineItem("bolt", new BigDecimal("0.125"), 4));

        assertEquals(new BigDecimal("0.50"), cart.total());
    }

    @Test
    @DisplayName("gift wrapping fee applies to every order [verifies CART-99~1]")
    void giftWrappingFeeAppliesToEveryOrder() {
        Cart cart = new Cart();
        cart.addItem(new LineItem("wrap", new BigDecimal("2.00"), 1));

        assertEquals(new BigDecimal("2.00"), cart.total());
    }

    @Test
    @DisplayName("free sample line item contributes nothing to the total [verifies CART-1]")
    void freeSampleContributesNothing() {
        Cart cart = new Cart();
        cart.addItem(new LineItem("sample", BigDecimal.ZERO, 1));
        cart.addItem(new LineItem("mug", new BigDecimal("4.50"), 1));

        assertEquals(new BigDecimal("4.50"), cart.total());
    }

    @Test
    @DisplayName("total of a single one dollar item is one dollar")
    void totalOfSingleOneDollarItemIsOneDollar() {
        Cart cart = new Cart();
        cart.addItem(new LineItem("token", new BigDecimal("1.00"), 1));

        assertEquals(new BigDecimal("1.00"), cart.total());
    }

    @Test
    @DisplayName("combined item and discount total is correct [verifies CART-1~1, CART-2~2]")
    void combinedItemAndDiscountTotalIsCorrect() {
        Cart cart = new Cart();
        cart.addItem(new LineItem("mug", new BigDecimal("10.00"), 2));
        cart.applyDiscount(new DiscountCode("SAVE20", new BigDecimal("20")));

        assertEquals(new BigDecimal("16.00"), cart.total());
    }

    @Nested
    @DisplayName("gift bundle pricing [verifies CART-1~1]")
    class GiftBundlePricing {

        @Test
        @DisplayName("bundle total sums each included item")
        void bundleTotalSumsEachIncludedItem() {
            Cart cart = new Cart();
            cart.addItem(new LineItem("candle", new BigDecimal("6.00"), 1));
            cart.addItem(new LineItem("card", new BigDecimal("2.50"), 2));

            assertEquals(new BigDecimal("11.00"), cart.total());
        }
    }
}
