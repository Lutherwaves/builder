# orderbook

A small order book for independent shops. Each shop owner logs in and sees the orders of their own shop.

Most owners do their books once a month. They copy each order from the Orders page into the spreadsheet they send to their accountant. Owners with more than 200 orders a month say this takes an evening.

## Layout

- `orders.py` — the order store and the monthly list the Orders page shows.
- `docs/specs/` — one file per piece of work, named `YYYY-MM-DD-<slug>.md`.

## Tests

    python3 -m unittest
