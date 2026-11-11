import unittest

import mailer


class ValidAddress(unittest.TestCase):
    def test_accepts_a_plain_address(self):
        self.assertTrue(mailer.valid_address("ana@example.com"))

    def test_rejects_what_is_not_an_address(self):
        for address in (None, "", "ana", "ana@", "@example.com", "ana@localhost", "a@b@example.com"):
            self.assertFalse(mailer.valid_address(address), address)


class Send(unittest.TestCase):
    def test_hands_the_message_to_the_relay(self):
        delivered = []
        accepted = mailer.send("ana@example.com", "hello", deliver=delivered.append)
        self.assertTrue(accepted)
        self.assertEqual(len(delivered), 1)
        self.assertEqual(delivered[0]["To"], "ana@example.com")
        self.assertEqual(delivered[0].get_content().strip(), "hello")


class Notify(unittest.TestCase):
    def test_rejects_a_bad_address_before_sending(self):
        with self.assertRaises(ValueError):
            mailer.notify("not an address", "hello")


if __name__ == "__main__":
    unittest.main()
