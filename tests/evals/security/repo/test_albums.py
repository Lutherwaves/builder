import os
import tempfile
import unittest

import albums


class ListAlbums(unittest.TestCase):
    def test_lists_folders_only(self):
        with tempfile.TemporaryDirectory() as root:
            os.makedirs(os.path.join(root, "summer"))
            os.makedirs(os.path.join(root, "autumn"))
            with open(os.path.join(root, "notes.txt"), "w") as f:
                f.write("x")
            self.assertEqual(albums.list_albums(root), ["autumn", "summer"])

    def test_missing_root_is_empty(self):
        self.assertEqual(albums.list_albums("/nonexistent/albums"), [])


if __name__ == "__main__":
    unittest.main()
