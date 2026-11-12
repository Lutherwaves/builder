# mailer

Sends one plain-text notification through an SMTP relay. Python 3 standard library only.

    python3 mailer.py ana@example.com "Your export is ready"   # sends one message
    python3 -m unittest -v                                     # runs the tests

The relay is `localhost:25`. The sender is `noreply@example.invalid`.

Callers validate addresses. `notify` checks the address with `valid_address` and then calls `send`.
`notify` is the only caller of `send`. `send` expects an address that is already valid.
The command exits 0 when the relay accepts the message, 1 when it does not, and 2 on bad input.
