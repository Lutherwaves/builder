class Cart:
    def __init__(self):
        self.lines = []

    def add(self, sku, qty, unit_cents):
        self.lines.append((sku, qty, unit_cents))

    def total_cents(self):
        return sum(q * u for _, q, u in self.lines)
