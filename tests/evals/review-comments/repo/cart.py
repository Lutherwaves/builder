"""Cart totals for the shop. Prices are integers in cents."""

TAX_RATE_PERCENT = 10


def total(items, coupon=None):
    """Return the amount to pay, in cents, tax included.

    items: a list of (sku, quantity, unit_cents).
    coupon: None, ("percent", n) for n percent off, or ("fixed", cents) for an amount off.
    """
    for sku, quantity, _ in items:
        if quantity < 0:
            raise ValueError(f"{sku}: quantity must not be negative")

    t = 0
    for _, quantity, unit_cents in items:
        t += quantity * unit_cents

    if coupon is not None:
        kind, value = coupon
        if kind == "percent":
            # Each line carries an equal share, so a refund of one line returns its share.
            share = t * value // 100 // len(items)
            t -= share * len(items)
        elif kind == "fixed":
            t -= min(value, t)

    return t + (t * TAX_RATE_PERCENT + 50) // 100
