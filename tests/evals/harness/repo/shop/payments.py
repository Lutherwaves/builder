class Declined(Exception):
    pass


def charge(card_token, amount_cents):
    if card_token.startswith("tok_declined"):
        raise Declined(card_token)
    return {"id": "ch_" + card_token[-4:], "amount_cents": amount_cents}


def refund(charge_id, amount_cents):
    return {"id": "re_" + charge_id[-4:], "amount_cents": amount_cents}
