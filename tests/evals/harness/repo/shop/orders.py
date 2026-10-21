from shop.payments import charge, refund


def place_order(cart, card_token):
    total = cart.total_cents()
    receipt = charge(card_token, total)
    return {"total_cents": total, "charge": receipt["id"], "status": "paid"}


def cancel_order(order):
    refund(order["charge"], order["total_cents"])
    return dict(order, status="cancelled")
