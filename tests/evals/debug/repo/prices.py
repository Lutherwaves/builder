from decimal import Decimal


def parse_price(text):
    """Turn a price from the shop export into a Decimal."""
    return Decimal(text.strip().split(",")[0])
