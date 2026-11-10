"""The contact store and the list the Contacts page shows."""

_CONTACTS = []


def add(name, email, added):
    _CONTACTS.append({"name": name, "email": email, "added": added})


def list_contacts():
    return sorted(_CONTACTS, key=lambda c: c["name"].lower())
