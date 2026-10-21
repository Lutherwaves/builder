from shop.cart import Cart
from shop.orders import cancel_order, place_order


def post_order(body):
    cart = Cart()
    for line in body["lines"]:
        cart.add(line["sku"], line["qty"], line["unit_cents"])
    return place_order(cart, body["card_token"])


def post_cancel(order):
    return cancel_order(order)
