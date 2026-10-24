"""clubhouse: the membership server of a small sports club. Python 3 standard library only."""
import json
import os
import re
import sqlite3
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlparse

DB = os.environ.get("CLUBHOUSE_DB", "clubhouse.db")
MAX_BODY = 64 * 1024


def connect(path=DB):
    db = sqlite3.connect(path)
    db.execute("CREATE TABLE IF NOT EXISTS members (id INTEGER PRIMARY KEY, name TEXT, email TEXT, paid INTEGER)")
    db.execute("CREATE TABLE IF NOT EXISTS posts (id INTEGER PRIMARY KEY, body TEXT)")
    return db


def mark_paid(db, event):
    """The payment provider tells us a member paid their fee."""
    if event.get("type") == "payment.succeeded":
        db.execute("UPDATE members SET paid = 1 WHERE email = ?", (event["email"],))
        db.commit()
        return True
    return False


def search_posts(db, pattern):
    """Every club post whose body matches a pattern the visitor types."""
    rx = re.compile(pattern)
    return [body for (body,) in db.execute("SELECT body FROM posts") if rx.search(body)]


class Handler(BaseHTTPRequestHandler):
    def send_json(self, status, payload):
        body = json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def read_json(self):
        length = int(self.headers.get("Content-Length", "0"))
        if length > MAX_BODY:
            raise ValueError("body too large")
        return json.loads(self.rfile.read(length))

    def do_GET(self):
        url = urlparse(self.path)
        if url.path == "/health":
            return self.send_json(200, {"ok": True})
        if url.path == "/search":
            q = parse_qs(url.query).get("q", [""])[0]
            return self.send_json(200, {"posts": search_posts(connect(), q)})
        return self.send_json(404, {"error": "not found"})

    def do_POST(self):
        if self.path == "/webhook/payments":
            try:
                event = self.read_json()
            except ValueError as e:
                return self.send_json(400, {"error": str(e)})
            return self.send_json(200, {"updated": mark_paid(connect(), event)})
        return self.send_json(404, {"error": "not found"})


def main(argv):
    port = int(argv[0]) if argv else 8080
    ThreadingHTTPServer(("0.0.0.0", port), Handler).serve_forever()


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
