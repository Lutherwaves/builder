# Incidents

## 2026-04-14 — checkout returned 500 for 11 minutes

- Trigger: `migrations/0004_rename_customer_phone.sql` renamed `customers.phone` to `customers.phone_number`.
- CI ran the migration, then started the rollout. The old API instances still selected `customers.phone`. Every checkout on an old instance failed until the rollout finished.
- Impact: 11 minutes of failed checkouts, about 340 orders lost.
- Fix: `0005_restore_customer_phone.sql` renamed `phone_number` back to `phone`.
- Follow-up: "be careful with renames" (no check added).

## 2026-10-20 — no invoices were sent for one night

- Trigger: `migrations/0007_drop_orders_legacy_status.sql` dropped `orders.legacy_status`.
- The API had stopped reading the column in release 41. The nightly invoice job was still on release 39, and it reads `legacy_status`. The job crashed at 02:00 on the first query.
- Impact: 1,900 invoices were not sent. Finance re-ran the job by hand the next morning.
- Fix: `0008_restore_orders_legacy_status.sql` added the column back.
- Follow-up: none yet.
