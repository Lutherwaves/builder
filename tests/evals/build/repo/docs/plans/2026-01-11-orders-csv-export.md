# Plan: a shop owner sends a month of orders to their accountant in one download

Spec: `docs/specs/2026-01-10-orders-csv-export.md` (approved by the owner on 2026-01-10).
Test command: `python3 -m pytest`.

## File map

- `export.py` (create) — turns one shop's month of orders into CSV text.
- `tests/test_export.py` (create) — the tests of `export.py`.
- `orders.py` (no change) — the export reads the order store through it.

## Task 1: one order becomes one CSV row, refunds marked

**Files:** `export.py`, `tests/test_export.py`.

**Test first:** three tests in `tests/test_export.py`.

```python
from datetime import date

import export
import orders


def order(id=1, shop_id=1, placed_on=date(2026, 3, 2), name="Ana Petrova", refunded=False):
    return orders.Order(id, shop_id, placed_on, name, "ana@example.com",
                        [("MUG", 2, 900)], 360, refunded)


def test_row_has_the_header_columns():
    assert export.HEADER == ["order_id", "placed_on", "customer_name", "total_cents", "refunded"]
    assert export.order_row(order()) == [1, "2026-03-02", "Ana Petrova", 2160, "no"]


def test_refunded_row_is_marked():
    assert export.order_row(order(refunded=True))[-1] == "yes"


def test_no_customer_email():
    assert "ana@example.com" not in export.order_row(order())
    assert not any("email" in column for column in export.HEADER)
```

**Run:** `python3 -m pytest tests/test_export.py`.
Before the code: the run fails with `ModuleNotFoundError: No module named 'export'`.
After the code: 3 passed.

**Do:**
1. Create `export.py` with the list `HEADER` above.
2. Add `order_row(order)`. It returns `[order.id, order.placed_on.isoformat(), order.customer_name, order.total_cents, "yes" or "no"]`. The last cell is `"yes"` when `order.refunded` is true.
3. Leave `order.customer_email` out of the row.

**Done when:** the three tests pass, and `python3 -m pytest` passes.

**Commit:** `feat(export): one order becomes one CSV row with refunds marked`

## Task 2: the file holds one shop's month, and its total matches the page

**Files:** `export.py`, `tests/test_export.py`.

**Test first:** three tests in `tests/test_export.py`. Clear `orders._ORDERS` before and after each test with an autouse fixture.

```python
import csv
import io


def rows(text):
    return list(csv.reader(io.StringIO(text)))


def test_month_csv_holds_one_shop_and_one_month():
    orders.add(order(id=1, shop_id=1, placed_on=date(2026, 3, 2)))
    orders.add(order(id=2, shop_id=2, placed_on=date(2026, 3, 2)))
    orders.add(order(id=3, shop_id=1, placed_on=date(2026, 4, 1)))
    lines = rows(export.month_csv(1, 2026, 3))
    assert lines[0] == export.HEADER
    assert [line[0] for line in lines[1:]] == ["1"]


def test_file_total_matches_the_page_total():
    orders.add(order(id=1, placed_on=date(2026, 3, 2)))
    orders.add(order(id=2, placed_on=date(2026, 3, 9), refunded=True))
    lines = rows(export.month_csv(1, 2026, 3))[1:]
    file_total = sum(int(line[3]) for line in lines if line[4] == "no")
    assert file_total == orders.month_total_cents(1, 2026, 3)


def test_empty_month_gives_no_file():
    orders.add(order(id=1, shop_id=2, placed_on=date(2026, 3, 2)))
    assert export.month_csv(1, 2026, 3) is None
```

**Run:** `python3 -m pytest tests/test_export.py`.
Before the code: the three new tests fail with `AttributeError: module 'export' has no attribute 'month_csv'`.
After the code: 6 passed.

**Do:**
1. Add `month_csv(shop_id, year, month_no)` to `export.py`.
2. Read the orders with `orders.load_page(shop_id, year, month_no)`. It returns the orders the Orders page lists for that shop and month, refunds included.
3. When the list is empty, return `None`. The page then shows "No orders in March 2026. There is nothing to download."
4. Otherwise write `HEADER`, then `order_row(o)` for each order, with `csv.writer` into an `io.StringIO`. Return the text.

**Done when:** the six tests pass, and `python3 -m pytest` passes.

**Commit:** `feat(export): a shop owner gets one month of their own orders as CSV text`

## Task 3: a customer name never runs as a formula

Surface: personal data

**Files:** `export.py`, `tests/test_export.py`.

**Test first:** the hostile-input test for the anti-goal "a customer name that starts with `=`, `+`, `-` or `@` runs as a formula".

```python
import pytest


@pytest.mark.parametrize("name", ['=HYPERLINK("http://x","y")', "+1+1", "-2+3", "@SUM(A1)"])
def test_formula_name_is_plain_text(name):
    assert export.order_row(order(name=name))[2] == "'" + name


def test_plain_name_is_unchanged():
    assert export.order_row(order(name="Ana Petrova"))[2] == "Ana Petrova"
```

**Run:** `python3 -m pytest tests/test_export.py`.
Before the code: the four `test_formula_name_is_plain_text` cases fail; the cell holds the name with no `'`.
After the code: 11 passed.

**Do:**
1. In `order_row`, when `order.customer_name` starts with `=`, `+`, `-` or `@`, put `'` before it.
2. Leave every other name as it is.

**Done when:** the hostile-input test passes for all four names, and `python3 -m pytest` passes.

**Commit:** `fix(export): a customer name never runs as a formula in the spreadsheet`

## The gate

After Task 3, the builder skill `review` reviews the whole change against the spec, with `python3 -m pytest`.

## Review

Engineering lens:
- What exists: `orders.month_total_cents` gives the page total. Task 2 tests the file total against it. No change.
- Failure mode: the store is a module list, shared by every test. Task 2 clears it around each test with an autouse fixture.
- Authentication: the shop id comes from the signed-in session in the web layer, which is not in this repository. Task 2 pins that the file holds only the asked shop's orders. No `Surface:` tag, because no code here reads the session.

Design lens:
- The owner opens the file in a spreadsheet. The header names are plain words, and a refund reads "yes" or "no". No change.
- The empty month and the other shop give the same result, `None`, so the page shows the same words for both. No change.

## Check against the spec

| Spec line | Task |
|---|---|
| One row per order of their shop in March 2026, refunds marked | Task 1, Task 2 |
| The file total, without refunds, equals the page total | Task 2: the file holds one shop's month, and its total matches the page |
| No customer email in the file | Task 1: one order becomes one CSV row, refunds marked |
| `=HYPERLINK("http://x","y")` shows as plain text | Task 3: a customer name never runs as a formula |
| An owner of shop 1 who asks for shop 2 gets no file | Task 2 (`None`, the same as an empty month) |
| A month with no orders downloads no file | Task 2: `test_empty_month_gives_no_file` |

## Calls

- The button, the file name `orders-YYYY-MM.csv` and the page messages live in the web layer, which is not in this repository. This plan builds the export that the page calls.
