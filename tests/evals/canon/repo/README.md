# shopdb

The database schema and migrations for the shop API and the nightly invoice job.

## Deploy

CI runs every new file in `migrations/` against production first. Then it rolls the new API release out over about ten minutes. During the rollout, old and new API instances serve traffic side by side. The nightly invoice job is pinned to a release and is upgraded by hand, usually a few days later.

## Tests

    python3 -m unittest discover -s tests
