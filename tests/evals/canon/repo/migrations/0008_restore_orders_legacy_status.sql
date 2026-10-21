-- Revert of 0007. See INCIDENTS.md, 2026-10-20. Data restored from the 2026-10-19 backup by hand.
ALTER TABLE orders ADD COLUMN legacy_status TEXT;
