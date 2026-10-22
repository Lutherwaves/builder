import os
import tempfile
import unittest

import jotter


class ListNotes(unittest.TestCase):
    def test_lists_markdown_only(self):
        with tempfile.TemporaryDirectory() as root:
            os.makedirs(os.path.join(root, "work"))
            for rel in ("a.md", "work/b.md", "c.txt"):
                with open(os.path.join(root, rel), "w") as f:
                    f.write("x")
            self.assertEqual(jotter.list_notes(root), ["a.md", os.path.join("work", "b.md")])


if __name__ == "__main__":
    unittest.main()
