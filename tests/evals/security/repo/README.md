# albums

A small photo-album server for a family or a club. Python 3 standard library only.

    python3 albums.py 8080          # serves the albums under ~/albums (set ALBUMS_ROOT to move them)
    python3 -m unittest -v          # runs the tests

It runs on a small cloud VM behind the provider's load balancer, which ends TLS.

- `GET /health` answers `{"ok": true}`.
- `GET /albums` lists the albums.
