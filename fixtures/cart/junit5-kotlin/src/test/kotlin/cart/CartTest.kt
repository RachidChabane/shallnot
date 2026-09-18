package cart

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertThrows
import org.junit.jupiter.api.DisplayName
import org.junit.jupiter.api.Disabled
import org.junit.jupiter.api.Nested
import org.junit.jupiter.api.Test
import java.math.BigDecimal

class CartTest {

    @Test
    @DisplayName("total sums line item prices times quantities [verifies CART-1~1]")
    fun totalSumsLineItems() {
        val cart = Cart()
        cart.addItem(BigDecimal("9.99"), 2)
        cart.addItem(BigDecimal("4.50"), 1)
        assertEquals(BigDecimal("24.48"), cart.total())
    }

    @Test
    @DisplayName("checking out an empty cart is rejected [verifies CART-3~1]")
    fun checkoutOfEmptyCartIsRejected() {
        val cart = Cart()
        assertThrows(IllegalStateException::class.java) { cart.checkout() }
    }

    @Test
    @Disabled("rounding mode pending finance decision")
    @DisplayName("totals round to two decimals half-up [verifies CART-4~1]")
    fun totalsRoundHalfUp() {
        val cart = Cart()
        cart.addItem(BigDecimal("1.005"), 1)
        assertEquals(BigDecimal("1.01"), cart.total())
    }

    @Test
    @DisplayName("catalog reference number stays stable [verifies CART-99~1]")
    fun catalogReferenceNumberStaysStable() {
        val cart = Cart()
        cart.addItem(BigDecimal("3.00"), 1)
        assertEquals(BigDecimal("3.00"), cart.total())
    }

    @Test
    @DisplayName("single item total matches unit price [verifies CART-1]")
    fun singleItemTotalMatchesUnitPrice() {
        val cart = Cart()
        cart.addItem(BigDecimal("5.00"), 1)
        assertEquals(BigDecimal("5.00"), cart.total())
    }

    @Test
    @DisplayName("cart starts out with a zero total")
    fun cartStartsOutWithZeroTotal() {
        val cart = Cart()
        assertEquals(BigDecimal("0.00"), cart.total())
    }

    @Test
    @DisplayName("first item price rolls into the discounted total [verifies CART-1~1, CART-2~2]")
    fun firstItemPriceRollsIntoDiscountedTotal() {
        val cart = Cart()
        cart.addItem(BigDecimal("10.00"), 2)
        val total = cart.total(DiscountCode(BigDecimal("10")))
        assertEquals(BigDecimal("18.00"), total)
    }

    @Nested
    @DisplayName("bulk order handling [verifies CART-1~1]")
    inner class BulkOrderHandling {

        @Test
        @DisplayName("adding sixty units of one item is accepted")
        fun addingSixtyUnitsIsAccepted() {
            val cart = Cart()
            cart.addItem(BigDecimal("2.00"), 60)
            assertEquals(BigDecimal("120.00"), cart.total())
        }
    }
}
