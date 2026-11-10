# Review of feat/export

**What I ran.** `python3 -m pytest` on `feat/export` → 4 passed. The security pass ran: the change writes personal data (names and emails) to a file that the web app serves.

**Coverage.** A01 Broken Access Control: inspected in `export.py`; the serving of `exports/` is not in this repository. Other rows: n/a. Surfaces checked: the export file and its path. Not checked: the web server config, because it is not in this repository.

**Most important first.**

No Critical or Important finding.

## Open checks that block ship

- `authz/CWE-862/export.py/save_export` (`export.py:17`). Missing fact: the web server serves `exports/<owner_id>/` only to that owner, signed in. Who can observe it: the person who runs the web server config. Reply `authz/CWE-862/export.py/save_export: exports/ is behind sign-in at the server`, or `authz/CWE-862/export.py/save_export: exports/ is public, fix it`.

## Claims checked

- "Owners can download their contacts as a CSV file": true.

## Not proven

- not used as a running product: the fixture has no server.

## Declined to judge

- `owner_id` in the path: the caller passes the signed-in owner's id, not user input.

## Verdict

Not yet: one open check blocks ship.
