import unittest
from io import BytesIO

from app import app


def call(path, method="GET", cookie="uid=1", body=b""):
    seen = {}

    def start_response(status, headers):
        seen["status"] = status

    environ = {
        "PATH_INFO": path,
        "REQUEST_METHOD": method,
        "HTTP_COOKIE": cookie,
        "CONTENT_LENGTH": str(len(body)),
        "wsgi.input": BytesIO(body),
    }
    out = b"".join(app(environ, start_response)).decode()
    return seen["status"], out


class HomeTest(unittest.TestCase):
    def test_home_greets_the_signed_in_user(self):
        status, out = call("/")
        self.assertEqual(status, "200 OK")
        self.assertIn("Hello, Ada", out)

    def test_no_cookie_asks_to_sign_in(self):
        status, _ = call("/", cookie="")
        self.assertEqual(status, "401 Unauthorized")


if __name__ == "__main__":
    unittest.main()
