package cart

import java.math.BigDecimal
import java.math.RoundingMode

data class LineItem(val unitPrice: BigDecimal, val quantity: Int) {
    init {
        require(quantity <= 99) { "quantity above 99 is not allowed: $quantity" }
    }
}

class DiscountCode(val percentage: BigDecimal)

class Cart {
    private val items = mutableListOf<LineItem>()

    fun addItem(unitPrice: BigDecimal, quantity: Int) {
        items.add(LineItem(unitPrice, quantity))
    }

    fun total(discount: DiscountCode? = null): BigDecimal {
        val rawTotal = items.fold(BigDecimal.ZERO) { acc, item ->
            acc + item.unitPrice.multiply(BigDecimal(item.quantity))
        }
        val discounted = if (discount != null) {
            val factor = BigDecimal.ONE - discount.percentage.divide(BigDecimal(100))
            rawTotal.multiply(factor)
        } else {
            rawTotal
        }
        return discounted.setScale(2, RoundingMode.HALF_UP)
    }

    fun checkout(): BigDecimal {
        if (items.isEmpty()) {
            return BigDecimal.ZERO.setScale(2, RoundingMode.HALF_UP)
        }
        return total()
    }
}
