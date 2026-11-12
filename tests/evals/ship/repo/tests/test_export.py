from datetime import date

import pytest

import contacts
import export


@pytest.fixture(autouse=True)
def empty_store():
    contacts._CONTACTS.clear()
    yield
    contacts._CONTACTS.clear()


def test_export_has_one_row_per_contact():
    contacts.add("Ana", "ana@example.com", date(2026, 1, 3))
    assert export.export_csv(contacts.list_contacts()) == (
        "name,email,added\r\nAna,ana@example.com,2026-01-03\r\n"
    )


def test_save_export_writes_one_file_per_owner(tmp_path):
    path = export.save_export([], "owner-7", folder=tmp_path)
    assert path == tmp_path / "owner-7" / "contacts.csv"
    assert path.read_text(encoding="utf-8").splitlines() == ["name,email,added"]
