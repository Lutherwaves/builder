"""WSGI app: a home page per signed-in user."""

from html import escape
from http.cookies import SimpleCookie
from wsgiref.simple_server import make_server

from users import seed

STORE = seed()


def current_user(environ):
    cookie = SimpleCookie(environ.get("HTTP_COOKIE", ""))
    if "uid" not in cookie:
        return None
    try:
        return STORE.get(int(cookie["uid"].value))
    except ValueError:
        return None


def page(title, body):
    return f"<!doctype html><title>{escape(title)}</title><h1>{escape(title)}</h1>{body}"


def app(environ, start_response):
    user = current_user(environ)
    path = environ.get("PATH_INFO", "/")
    if user is None:
        start_response("401 Unauthorized", [("Content-Type", "text/html")])
        return [page("Sign in", "<p>Set the uid cookie to sign in.</p>").encode()]
    if path == "/":
        start_response("200 OK", [("Content-Type", "text/html")])
        return [page(f"Hello, {user.name}", f"<p>Signed in as {escape(user.email)}.</p>").encode()]
    start_response("404 Not Found", [("Content-Type", "text/html")])
    return [page("Not found", "<p>No such page.</p>").encode()]


if __name__ == "__main__":
    with make_server("", 8000, app) as server:
        print("serving on http://localhost:8000")
        server.serve_forever()
