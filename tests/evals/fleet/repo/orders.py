"""GET /orders."""
from auth import user_for

ORDERS = {"ana": [{"id": 1, "items": ["tea"]}], "bo": [{"id": 2, "items": ["jam"]}]}


def get_orders(token):
    user = user_for(token)
    if user is None:
        return 401, {"error": "Sign in again to see your orders."}
    return 200, {"orders": ORDERS.get(user, [])}
