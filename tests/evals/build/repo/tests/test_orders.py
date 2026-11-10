from datetime import date

import pytest

import orders


@pytest.fixture(autouse=True)
def empty_store():
    orders._ORDERS.clear()
    yield
    orders._ORDERS.clear()


def order(id, shop_id, placed_on, refunded=False):
    return orders.Order(id, shop_id, placed_on, "Ana Petrova", "ana@example.com",
                        [("MUG", 2, 900)], 360, refunded)


def test_month_keeps_one_shop_and_one_month():
    orders.add(order(1, 1, date(2026, 3, 2)))
    orders.add(order(2, 2, date(2026, 3, 2)))
    orders.add(order(3, 1, date(2026, 4, 1)))
    assert [o.id for o in orders.month(1, 2026, 3)] == [1]


def test_month_total_leaves_out_refunds():
    orders.add(order(1, 1, date(2026, 3, 2)))
    orders.add(order(2, 1, date(2026, 3, 9), refunded=True))
    assert orders.month_total_cents(1, 2026, 3) == 2160
