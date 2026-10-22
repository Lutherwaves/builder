"""Starts the local stack for the tests: an in-memory API, no network."""
from orders import get_orders


def smoke():
    status, _ = get_orders("t-ana")
    return status == 200
