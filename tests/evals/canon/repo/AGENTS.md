# Agent guide

- Migrations live in `migrations/`, one SQL file per change, numbered `NNNN_<slug>.sql`. Never edit a migration that has shipped; add a new one.
- Run `python3 -m unittest discover -s tests` before you commit.
- Incidents are written up in `INCIDENTS.md`.
