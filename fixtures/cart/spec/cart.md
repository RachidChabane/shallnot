# Cart pricing

The cart service computes what a customer pays at checkout. Prices are
decimal amounts; quantities are whole numbers.

## Totals

- **CART-1~1**: WHEN a cart holds line items THE SYSTEM SHALL compute the
  total as the sum of each line item's price times its quantity.

- **CART-4~1**: WHEN a total has more than two decimals THE SYSTEM SHALL round
  it to two decimals, half-up.

## Discounts

### CART-2~2: Percentage discount codes

WHEN a customer applies a percentage discount code THE SYSTEM SHALL reduce the
total by that percentage.

Revision 2 replaced flat-amount codes with percentage codes.

## Checkout

- **CART-3~1**: WHEN a customer checks out an empty cart THE SYSTEM SHALL
  reject the checkout with an error.
- **CART-5~1**: WHEN a line item quantity exceeds 99 THE SYSTEM SHALL reject
  the line item.

## Presentation

- **CART-6~1**: WHEN a total is displayed THE SYSTEM SHALL show it in the
  customer's currency.
- **CART-7~1**: THE checkout page SHALL feel uncluttered on a phone screen.
  - Non-testable: judged in the quarterly design review with customer panels;
    no automated check can stand in for it.
