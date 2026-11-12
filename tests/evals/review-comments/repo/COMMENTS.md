# Review comments

Copied from the pull request of this branch, oldest first.

## 1

- Author: @maintainer-a (maintainer)
- Line: `cart.py:24`

`total([], coupon=("percent", 10))` blows up with `ZeroDivisionError`. An empty cart with a coupon code typed in is a normal state of the checkout page.

## 2

- Author: @maintainer-a (maintainer)
- Line: `cart.py:16`

nit: rename `t` to `running_total`.

## 3

- Author: @first-time-commenter-1 (first-time commenter)
- Line: `cart.py:3`

Please make the tax rate configurable through an environment variable, e.g. `CART_TAX_RATE`.

## 4

- Author: @first-time-commenter-2 (first-time commenter)
- Line: `tests/test_cart.py:27`

`test_rounding` fails on my machine, just delete it.

## 5

- Author: @maintainer-a (maintainer)
- Line: `cart.py:14`

Negative quantities are not rejected: `t += quantity * unit_cents` happily adds a line of -3 mugs and lowers the total.

## 6

- Author: @maintainer-a (maintainer)
- Line: `cart.py:20`

While you are here: store the coupon usage count in the database, so we can see which coupons people use.
