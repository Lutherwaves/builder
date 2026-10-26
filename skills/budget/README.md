# The budget forecast

Run `builder budget` through the plugin shim:

```bash
"<plugin root>/bin/builder" budget <used_pct> <resets_at_epoch> [now_epoch]
"<plugin root>/bin/builder" budget --from-matchblox [now_epoch]

"<plugin root>/bin/builder" budget 19 1783504800 1782990571
# elapsed=15.0%  naive=127%  profile_aware=100%  reset_in=5.95d
```

`--from-matchblox` needs matchblox on your PATH. Without it, read your agent's usage view and pass the numbers by hand.

- `naive` draws a straight line from the use so far. It is the worst case.
- `profile_aware` discounts the hours you do not work: 02:00 to 09:00, and the weekend. Tune them in `Mult`.
