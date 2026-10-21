"""User records. The store is in memory; the app seeds it on start."""

from dataclasses import dataclass


@dataclass
class User:
    id: int
    name: str
    email: str


class UserStore:
    def __init__(self):
        self._users = {}

    def add(self, user):
        self._users[user.id] = user
        return user

    def get(self, user_id):
        return self._users.get(user_id)

    def by_email(self, email):
        for user in self._users.values():
            if user.email == email:
                return user
        return None


def seed():
    store = UserStore()
    store.add(User(1, "Alex", "alex@example.com"))
    store.add(User(2, "Sam", "sam@example.com"))
    return store
