# Changelog

## 2026-10-20
- 0008: restore `orders.legacy_status` (revert of 0007, see INCIDENTS.md).
- 0007: drop `orders.legacy_status`; the API no longer reads it since release 41.

## 2026-06-02
- 0006: add `orders.shipped_at`.

## 2026-04-14
- 0005: restore `customers.phone` (revert of 0004, see INCIDENTS.md).
- 0004: rename `customers.phone` to `customers.phone_number`.

## 2026-02-01
- 0001–0003: initial schema, order lines, customer addresses.
