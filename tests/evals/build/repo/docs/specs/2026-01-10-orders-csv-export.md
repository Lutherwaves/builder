# A shop owner sends a month of orders to their accountant in one download

## Who, and the moment

A shop owner at the end of the month, at a laptop, with the Orders page open and the accountant's monthly email waiting.
They copy each order from the Orders page into a spreadsheet, row by row, and send it to the accountant.
Evidence: the README says owners with more than 200 orders a month spend an evening on this.

## Why it matters

An owner with 200 or more orders loses one evening each month to copying (estimated, from the README).
A typed amount can be wrong, and the accountant then asks the owner to check every row (estimated).

## The experience

1. On the Orders page, the owner picks a month and clicks "Download CSV".
2. The browser saves `orders-YYYY-MM.csv`. The file opens in a spreadsheet with one row per order. Without the refunded rows, its total is the total at the foot of the Orders page.
3. The owner attaches the file to the accountant's email.

When it goes wrong:
- The month has no orders. The page says "No orders in March 2026. There is nothing to download." The button is off.
- The owner is signed out. The page says "Sign in again to download your orders." After sign-in, the owner is back on the same month.
- The download fails. The page says "The download did not finish. Try again." The owner clicks the button again.

## Non-goals

- No export of other date ranges than one calendar month.
- No Excel, PDF or other file format.
- No email of the file from the product. The owner sends it.
- No change to the Orders page list or its total.

## Anti-goals (failure even if it works)

- The file holds customer email addresses. The accountant does not need them, and the file leaves the shop.
- The file holds an order of another shop.
- A customer name that starts with `=`, `+`, `-` or `@` runs as a formula when the owner opens the file.
- The file total, without the refunded rows, differs from the total at the foot of the Orders page.

## Surfaces

- Personal data — each order holds `customer_name` and `customer_email`. The file goes to a third party, the accountant. Its anti-goals are above: no customer email in the file, and no formula from a customer name.
- Authentication — the download shows one shop's orders to a signed-in owner. Its anti-goal is above: no order of another shop.
- Checked and not touched: money movement, uploads, webhooks, secrets, CI, an agent's tools.

## Rulings

- 2026-01-10 — The file is CSV, one row per order, refunds included and marked — because the accountant's spreadsheet takes CSV and the Orders page also lists refunds. — the owner.
- 2026-01-10 — The file keeps the customer name and leaves out the customer email — because the accountant matches invoices by name and needs no way to contact a customer. — the owner.
- 2026-01-10 — One calendar month per file — because owners do their books once a month. — the owner.
- 2026-01-10: spec approved. Decided by: the owner.

## Left to the builder

- The column order and the header names, as long as the total column matches the page total.
- The CSV dialect, the line endings and the content type.
- Where the export function lives, and how it reuses `orders.month`.

## Done when

- [ ] An owner picks March 2026, clicks "Download CSV", and opens `orders-2026-03.csv` in a spreadsheet: one row per order of their shop in March 2026, refunds marked.
- [ ] The sum of the file's total column, over the rows not marked refunded, equals the total at the foot of the Orders page for the same month.
- [ ] The file has no column and no cell with a customer email address.
- [ ] A customer named `=HYPERLINK("http://x","y")` shows as plain text in the spreadsheet, not as a link.
- [ ] An owner of shop 1 who asks for the file of shop 2 gets no file and sees "No orders in March 2026. There is nothing to download.", the same words as an empty month.
- [ ] A month with no orders shows "No orders in March 2026. There is nothing to download." and no file downloads.
