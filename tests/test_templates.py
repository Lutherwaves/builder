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


REPORT_ORDER = ["state", "▶ You", "⚠ Not proven", "✓ Proven", "↗ Outside", "Calls", "↻ Next"]
STATES = ("✓ Done", "▶ Needs you", "✗ Blocked")


def report_line(label):
    """The REPORT_ORDER name of a bold report label, or None."""
    if label.lstrip("<").startswith(STATES):
        return "state"
    return next((name for name in REPORT_ORDER[1:] if label.startswith(name)), None)


class ReportShape(unittest.TestCase):
    """Every place that shows or lists the builder report keeps the template's lines and order."""

    def template(self):
        skill = read("builder", "SKILL.md")
        start = skill.index("```markdown")
        return skill[start:skill.index("```", start + 3)]

    def test_template_has_every_line_in_order(self):
        labels = re.findall(r"^\*\*(.+?):?\*\*", self.template(), re.M)
        self.assertEqual([report_line(l) for l in labels], REPORT_ORDER)

    def test_template_lines_are_apart(self):
        lines = [l for l in self.template().splitlines()[1:]]
        self.assertTrue(all(l == "" for l in lines[1::2]), "each report line needs a blank line after it")

    def test_soul_lists_the_lines_in_order(self):
        soul = read("builder", "soul-compact.md")
        listed = re.search(r"in this order: (.+?)\.\n", soul).group(1)
        names = re.sub(r" \(.*?\)", "", listed).split(", ")
        self.assertEqual(names, ["the state"] + REPORT_ORDER[1:])
        self.assertIn("not in a code block", soul)

    def test_readme_example_follows_the_order(self):
        with open(os.path.join(SKILLS, "..", "README.md"), encoding="utf-8") as f:
            labels = re.findall(r"^> \*\*(.+?):?\*\*", f.read(), re.M)
        order = [REPORT_ORDER.index(report_line(l)) for l in labels]
        self.assertTrue(labels and order == sorted(order), labels)

    def test_no_skill_uses_an_old_report_line(self):
        # A review or security finding keeps its own "Evidence:" field; only the old report's aligned labels and names fail.
        old = re.compile(r"^(For|Evidence|Outside):\s{2,}|\bunder Evidence\b|\b(For|Evidence) line\b|(?<!↗ )\bOutside(:| line)")
        for root, _, files in os.walk(SKILLS):
            for name in files:
                if name.endswith(".md"):
                    with open(os.path.join(root, name), encoding="utf-8") as f:
                        for n, line in enumerate(f, 1):
                            self.assertIsNone(old.search(line), f"{name}:{n}: {line.strip()}")

    def test_report_asks_with_an_action_every_harness_maps(self):
        self.assertIn('the "ask the person to choose" action of your harness', read("builder", "SKILL.md"))
        for harness in ["claude", "codex", "cursor", "opencode"]:
            self.assertIn("| ask the person to choose |", read("builder", "harnesses", harness + ".md"))


class SendRuleWording(unittest.TestCase):
    """reach.py lets a send rule allow a message. Every text that names the gate says so."""

    def test_reach_teammate_row_names_the_send_rule(self):
        row = next(l for l in read("reach", "SKILL.md").splitlines() if l.startswith("| **teammate**"))
        self.assertIn("unless a send rule allows the kind", row)
        self.assertIn("the person's yes", row)

    def test_builder_gate_texts_name_the_send_rule(self):
        skill = read("builder", "SKILL.md")
        self.assertIn("without a send rule or the person's yes in this session", skill)
        self.assertIn("- [ ] Nothing left the repository without a send rule or the person's yes.", skill)
        self.assertIn("sent by rule: <n>, logged.", skill)
        self.assertNotIn("Nothing left the repository without a yes.", skill)


if __name__ == "__main__":
    unittest.main()
