"""Payment processing tools (fake for PoC)."""
from datetime import datetime


def calculate_price(people_counts: list[int]) -> dict:
    """Calculate order price for one or more cakes.

    Pricing: 5 EUR per person per cake. Delivery: 5 EUR if subtotal < 50 EUR, free otherwise.

    Args:
        people_counts: Number of people for each cake, e.g. [8] for one cake or [8, 12] for two.
                       Each value must be between 6 and 50.

    Returns:
        {"subtotal": float, "delivery_fee": float, "total": float}
    """
    if not people_counts:
        raise ValueError("At least one cake is required")

    subtotal = 0.0
    for people_count in people_counts:
        if not isinstance(people_count, int) or people_count < 6 or people_count > 50:
            raise ValueError(f"Invalid people_count: {people_count}. Must be 6-50")
        subtotal += people_count * 5.0

    delivery_fee = 0.0 if subtotal >= 50.0 else 5.0
    return {
        "subtotal": subtotal,
        "delivery_fee": delivery_fee,
        "total": subtotal + delivery_fee,
    }


def process_payment(order_id: str, amount: float, customer_name: str) -> dict:
    """Process payment (fake - always succeeds).

    Args:
        order_id: Order ID (e.g., CAKE-20260217-A3F2)
        amount: Payment amount in EUR
        customer_name: Customer name

    Returns:
        {
            "success": bool,
            "transaction_id": str,
            "amount": float,
            "message": str
        }
    """
    if amount <= 0:
        raise ValueError(f"Invalid amount: {amount}. Must be > 0")

    # Generate fake transaction ID
    timestamp = datetime.now().strftime("%Y%m%d%H%M%S")
    transaction_id = f"PAY-{timestamp}"

    return {
        "success": True,
        "transaction_id": transaction_id,
        "amount": amount,
        "message": f"Payment processed for {customer_name}. Total: €{amount:.2f}"
    }
