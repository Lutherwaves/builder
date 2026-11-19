# A shop owner downloads one month of orders as a file

## Who, and the moment

A shop owner, at their desk on the first evening of the month. They do the books for the month that ended.
They have the Orders page open and the spreadsheet for their accountant next to it.
Evidence: the README says that owners "copy each order from the Orders page into the spreadsheet they send to their accountant".

## Why it matters

Owners with more than 200 orders a month say that the copy takes an evening. The number is from the README and is not measured.

## The experience

1. The owner opens the Orders page and picks a month.
2. The owner presses "Download this month".
3. The browser saves one file. The owner opens it in their spreadsheet program.
4. The file has one row for each order: the order number, the date, the customer name, the total and the VAT.
5. The last row is the month total. It is the same number as the total at the foot of the Orders page.

The export follows the cut-off rule.

When the month has no orders, the page says: "No orders in this month. Pick another month." No file downloads.
When the download fails, the page says: "The download did not finish. Press the button again."

## Non-goals

- No export of more than one month at a time.
- No export for more than one shop.
- No schedule and no email of the file.

## Anti-goals (failure even if it works)

- The file holds an order of a different shop.
- The file holds a customer email address.
- The month total in the file is different from the total on the Orders page.

## Surfaces

- Personal data — a customer email address leaves the product in a file. Its anti-goal is above.
- No other sensitive surface touched. Checked: authentication, money, uploads, webhooks, secrets, CI, an agent's tools.

## Rulings

- 2026-01-08 — the file is built in `orders.py`, next to the monthly list, and not in a new service — because the monthly list already holds the rows. — the owner.
- 2026-01-09 — refunds are left out — because of the decision in docs/decisions/refunds.md. — the owner.
- 2026-01-09 — customer names stay in the file and customer email addresses do not — because the accountant needs the name only. — the owner.

## Left to the builder

- The file format, if the owner's spreadsheet program opens it with no import step.
- The name of the file.
- The order of the columns.

## Done when

- [ ] The owner picks a month with orders, presses "Download this month", and opens the file in a spreadsheet program.
- [ ] The file has no customer email address.
- [ ] The month total in the file is equal to the total at the foot of the Orders page for that month.
- [ ] The export feels fast.
