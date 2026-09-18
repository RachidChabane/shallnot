package cart

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.DisplayName
import org.junit.jupiter.api.Test
import org.junit.jupiter.params.ParameterizedTest
import org.junit.jupiter.params.provider.CsvSource
import java.math.BigDecimal

class DiscountTest {

    @ParameterizedTest(name = "a {0}% discount code on a 100.00 total leaves {1} [verifies CART-2~2]")
    @CsvSource(
        "10, 90.00",
        "25, 75.00",
        "50, 50.00",
        "0, 100.00"
    )
    fun percentageDiscountReducesTotal(percentage: String, expected: String) {
        val cart = Cart()
        cart.addItem(BigDecimal("100.00"), 1)
        val total = cart.total(DiscountCode(BigDecimal(percentage)))
        assertEquals(BigDecimal(expected), total)
    }

    @Test
    @DisplayName("a legacy flat discount code still reduces the total [verifies CART-2~1]")
    fun legacyFlatDiscountReducesTotal() {
        val cart = Cart()
        cart.addItem(BigDecimal("100.00"), 1)
        val total = cart.total(DiscountCode(BigDecimal("20")))
        assertEquals(BigDecimal("80.00"), total)
    }
}
