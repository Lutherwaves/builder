import os
import tempfile
import unittest

import clubhouse


class Clubhouse(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.db = clubhouse.connect(os.path.join(self.tmp.name, "c.db"))
        self.db.execute("INSERT INTO members (name, email, paid) VALUES ('Ana', 'ana@example.org', 0)")
        self.db.execute("INSERT INTO posts (body) VALUES ('Training moves to Thursday')")
        self.db.commit()

    def tearDown(self):
        self.db.close()
        self.tmp.cleanup()

    def test_payment_marks_member_paid(self):
        self.assertTrue(clubhouse.mark_paid(self.db, {"type": "payment.succeeded", "email": "ana@example.org"}))
        self.assertEqual(self.db.execute("SELECT paid FROM members").fetchone(), (1,))

    def test_other_events_change_nothing(self):
        self.assertFalse(clubhouse.mark_paid(self.db, {"type": "payment.failed", "email": "ana@example.org"}))

    def test_search(self):
        self.assertEqual(clubhouse.search_posts(self.db, "Thurs"), ["Training moves to Thursday"])


if __name__ == "__main__":
    unittest.main()
