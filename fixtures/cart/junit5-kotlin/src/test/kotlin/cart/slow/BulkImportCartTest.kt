package cart.slow

import org.junit.jupiter.api.Assertions.assertThrows
import org.junit.jupiter.api.DisplayName
import org.junit.jupiter.api.Test
import cart.Cart
import java.math.BigDecimal

class BulkImportCartTest {

    @Test
    @DisplayName("importing a quantity above ninety nine is rejected [verifies CART-5~1]")
    fun importingQuantityAboveNinetyNineIsRejected() {
        val cart = Cart()
        assertThrows(IllegalArgumentException::class.java) {
            cart.addItem(BigDecimal("1.00"), 150)
        }
    }
}
