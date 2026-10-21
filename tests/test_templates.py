"""The shape and canon templates keep the section order their SKILL.md lists.

Eval agents cannot read the templates (they sit outside the eval's working
directory), so the order written in each SKILL.md must match the template.
"""
import os
import re
import unittest

SKILLS = os.path.join(os.path.dirname(__file__), "..", "skills")


def read(*parts):
    with open(os.path.join(SKILLS, *parts), encoding="utf-8") as f:
        return f.read()


def listed_order(skill_md, marker):
    """The ' · ' list that follows `marker` on one line of SKILL.md."""
    m = re.search(re.escape(marker) + r"\s*(.+?)\.\s*(?:$|[A-Z])", skill_md, re.M)
    if not m:
        raise AssertionError(f"SKILL.md has no line with {marker!r}")
    return [p.strip() for p in m.group(1).split("·")]


class TemplateOrder(unittest.TestCase):
    def test_issue_template_matches_shape_skill(self):
        headings = re.findall(r"^## (.+?)(?: \(.*\))?$", read("shape", "issue-template.md"), re.M)
        self.assertEqual(headings, listed_order(read("shape", "SKILL.md"), "Its sections, in order:"))

    def test_rule_template_matches_canon_skill(self):
        lines = re.findall(r"^\*\*(\w+)\.\*\*", read("canon", "rule-template.md"), re.M)
        self.assertEqual(lines, listed_order(read("canon", "SKILL.md"), "Its lines, in order:"))


if __name__ == "__main__":
    unittest.main()
