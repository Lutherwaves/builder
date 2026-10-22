"""jotter: a tiny command-line notebook."""
import json
import logging
import os
import sys

log = logging.getLogger("jotter")


def load_config(path="config.json"):
    with open(path) as f:
        cfg = json.load(f)
    cfg["notes_dir"] = os.path.expanduser(cfg.get("notes_dir", "~/notes"))
    return cfg


def list_notes(root):
    """Every note under root, as a clean path relative to root."""
    out = []
    for dirpath, _, files in os.walk(root):
        for name in files:
            if name.endswith(".md"):
                rel = os.path.relpath(os.path.join(dirpath, name), root)
                out.append(os.path.normpath(rel))
    return sorted(out)


def main(argv):
    logging.basicConfig(filename="jotter.log", level=logging.INFO)
    cfg = load_config()
    if argv[:1] == ["list"]:
        for rel in list_notes(cfg["notes_dir"]):
            print(rel)
        return 0
    print("usage: jotter.py list")
    return 2


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
