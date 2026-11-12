from datetime import date

import pytest

import contacts


@pytest.fixture(autouse=True)
def empty_store():
    contacts._CONTACTS.clear()
    yield
    contacts._CONTACTS.clear()


def test_list_is_sorted_by_name():
    contacts.add("bo", "bo@example.com", date(2026, 1, 2))
    contacts.add("Ana", "ana@example.com", date(2026, 1, 3))
    assert [c["name"] for c in contacts.list_contacts()] == ["Ana", "bo"]


def test_list_keeps_the_date_added():
    contacts.add("Ana", "ana@example.com", date(2026, 1, 3))
    assert contacts.list_contacts()[0]["added"] == date(2026, 1, 3)
