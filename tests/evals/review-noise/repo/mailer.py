import logging
import smtplib
import sys
from email.message import EmailMessage

log = logging.getLogger("mailer")

RELAY_HOST = "localhost"
RELAY_PORT = 25
SENDER = "noreply@example.invalid"


def valid_address(address):
    if not isinstance(address, str) or address.count("@") != 1:
        return False
    local_part, domain = address.split("@")
    return bool(local_part) and "." in domain


def build_message(recipient, body):
    message = EmailMessage()
    message["From"] = SENDER
    message["To"] = recipient
    message["Subject"] = "Notification"
    message.set_content(body)
    return message


def smtp_deliver(message):
    with smtplib.SMTP(RELAY_HOST, RELAY_PORT, timeout=10) as relay:
        relay.send_message(message)


def send(to, body, deliver=smtp_deliver):
    """Send body to one address. Returns True when the relay accepted the message."""
    message = build_message(to, body)
    try:
        deliver(message)
    except OSError as error:
        log.warning("delivery failed: %s", error)
        return False
    return True


def notify(recipient, body):
    if not valid_address(recipient):
        raise ValueError(f"not an email address: {recipient!r}")
    return send(recipient, body)


def main(argv):
    logging.basicConfig(level=logging.INFO)
    if len(argv) != 2:
        print("usage: mailer.py <address> <message>")
        return 2
    recipient, body = argv
    try:
        delivered = notify(recipient, body)
    except ValueError as error:
        print(error)
        return 2
    print("Sent." if delivered else "Not sent. The relay did not accept the message.")
    return 0 if delivered else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
