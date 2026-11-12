"""CSV export of the Contacts page."""

import csv
import io
from pathlib import Path


def export_csv(rows):
    out = io.StringIO()
    writer = csv.writer(out)
    writer.writerow(["name", "email", "added"])
    for row in rows:
        writer.writerow([row["name"], row["email"], row["added"].isoformat()])
    return out.getvalue()


def save_export(rows, owner_id, folder="exports"):
    path = Path(folder) / owner_id / "contacts.csv"
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(export_csv(rows), encoding="utf-8")
    return path
