-- Revert of 0004. See INCIDENTS.md, 2026-04-14.
ALTER TABLE customers RENAME COLUMN phone_number TO phone;
