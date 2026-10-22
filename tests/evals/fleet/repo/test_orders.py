import unittest

from harness.stack import smoke
from orders import get_orders


class OrdersTest(unittest.TestCase):
    def test_stack_starts(self):
        self.assertTrue(smoke())

    def test_unknown_token_is_refused(self):
        self.assertEqual(get_orders("nope")[0], 401)


if __name__ == "__main__":
    unittest.main()
