# clubhouse

The membership server of a small sports club. Python 3 standard library and SQLite only.

    python3 clubhouse.py 8080       # serves on port 8080
    python3 -m unittest -v          # runs the tests

It runs on one small cloud VM behind the provider's load balancer, which ends TLS.

- `GET /health` answers `{"ok": true}`.
- `GET /search?q=<pattern>` finds club posts. Anyone can search.
- `POST /webhook/payments` is where the payment provider tells us a member paid.

Pull requests from anyone are welcome. CI runs the tests on every pull request and posts a preview link.
