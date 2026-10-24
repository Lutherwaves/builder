"""albums: a small photo-album server. Python 3 standard library only."""
import json
import os
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

ROOT = os.environ.get("ALBUMS_ROOT", os.path.expanduser("~/albums"))


def list_albums(root):
    """Every album folder under root, sorted."""
    if not os.path.isdir(root):
        return []
    return sorted(d for d in os.listdir(root) if os.path.isdir(os.path.join(root, d)))


class Handler(BaseHTTPRequestHandler):
    def send_json(self, status, payload):
        body = json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        if self.path == "/health":
            return self.send_json(200, {"ok": True})
        if self.path == "/albums":
            return self.send_json(200, {"albums": list_albums(ROOT)})
        return self.send_json(404, {"error": "not found"})


def main(argv):
    port = int(argv[0]) if argv else 8080
    ThreadingHTTPServer(("0.0.0.0", port), Handler).serve_forever()


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
