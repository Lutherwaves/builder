import os
import re
import unittest

MIGRATIONS = os.path.join(os.path.dirname(__file__), "..", "migrations")


class MigrationFiles(unittest.TestCase):
    def test_numbered_in_order_with_no_gaps(self):
        names = sorted(f for f in os.listdir(MIGRATIONS) if f.endswith(".sql"))
        numbers = [int(re.match(r"(\d{4})_", n).group(1)) for n in names]
        self.assertEqual(numbers, list(range(1, len(names) + 1)))


if __name__ == "__main__":
    unittest.main()
