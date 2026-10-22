import unittest

from shop.api import post_cancel, post_order


class ShopTest(unittest.TestCase):
    def test_order_then_cancel(self):
        order = post_order({"card_token": "tok_visa_4242", "lines": [{"sku": "a", "qty": 2, "unit_cents": 150}]})
        self.assertEqual(order["total_cents"], 300)
        self.assertEqual(post_cancel(order)["status"], "cancelled")


if __name__ == "__main__":
    unittest.main()
