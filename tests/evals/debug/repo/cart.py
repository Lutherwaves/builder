from prices import parse_price


def total(lines):
    """Add up (quantity, price text) lines."""
    return sum(qty * parse_price(price) for qty, price in lines)
