"""Bearer token check for every request."""
import time

TOKENS = {
    "t-ana": {"user": "ana", "expires": time.time() + 3600},
    "t-old": {"user": "bo", "expires": time.time() - 60},
}


def user_for(token):
    """Return the user for a token, or None."""
    entry = TOKENS.get(token)
    if entry is None:
        return None
    return entry["user"]
