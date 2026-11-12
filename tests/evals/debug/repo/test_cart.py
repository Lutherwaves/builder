from decimal import Decimal

from cart import total


def test_total_for_one_cheap_item():
    assert total([(2, "19.50")]) == Decimal("39.00")


def test_total_for_two_items():
    assert total([(1, "1,299.00"), (2, "19.50")]) == Decimal("1338.00")
