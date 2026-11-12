import pytest

from cart import total


def test_lines_add_up_with_tax():
    assert total([("MUG", 2, 900), ("PEN", 1, 200)]) == 2200


def test_percent_coupon():
    assert total([("MUG", 2, 900), ("PEN", 1, 200)], coupon=("percent", 10)) == 1980


def test_fixed_coupon_never_goes_below_zero():
    assert total([("PEN", 1, 200)], coupon=("fixed", 500)) == 0


def test_empty_cart_costs_nothing():
    assert total([]) == 0


def test_negative_quantity_is_rejected():
    with pytest.raises(ValueError):
        total([("MUG", -1, 900)])


def test_rounding():
    # Half a cent of tax rounds up; less than half rounds down.
    assert total([("STICKER", 1, 5)]) == 6
    assert total([("STICKER", 1, 25)]) == 28
    assert total([("STICKER", 1, 4)]) == 4
