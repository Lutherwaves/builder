"""Order store for one shop. Amounts are integer cents in the shop's currency."""
from dataclasses import dataclass
from datetime import date


@dataclass
class Order:
    id: int
    shop_id: int
    placed_on: date
    customer_name: str
    customer_email: str
    items: list  # [(sku, qty, unit_cents)]
    vat_cents: int
    refunded: bool = False

    @property
    def total_cents(self):
        return sum(q * c for _, q, c in self.items) + self.vat_cents


_ORDERS = []


def add(order):
    _ORDERS.append(order)


def month(shop_id, year, month_no):
    """The rows the Orders page shows: one shop, one calendar month, refunds included."""
    return [o for o in _ORDERS
            if o.shop_id == shop_id and o.placed_on.year == year and o.placed_on.month == month_no]


def month_total_cents(shop_id, year, month_no):
    """The total at the foot of the Orders page. Refunded orders do not count."""
    return sum(o.total_cents for o in month(shop_id, year, month_no) if not o.refunded)
