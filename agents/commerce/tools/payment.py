"""Payment processing tools (fake for PoC)."""
from typing import Dict
from datetime import datetime


def calculate_price(cakes: list[Dict]) -> Dict[str, float]:
    """Calculate order price.

    Pricing:
    - 5 EUR per slice (= per person)
    - Delivery fee: 5 EUR if total < 50 EUR, free otherwise
    - Multiple cakes allowed in single order

    Args:
        cakes: List of cake dicts with 'people_count' key
               e.g., [{"people_count": 8}, {"people_count": 12}]

    Returns:
        {
            "subtotal": float,
            "delivery_fee": float,
            "total": float
        }
    """
    if not cakes:
        raise ValueError("At least one cake is required")

    # Calculate subtotal (5 EUR per person)
    subtotal = 0.0
    for cake in cakes:
        people_count = cake.get("people_count", 0)

        if not isinstance(people_count, int) or people_count < 6 or people_count > 50:
            raise ValueError(f"Invalid people_count: {people_count}. Must be 6-50")

        subtotal += people_count * 5.0

    # Calculate delivery fee
    delivery_fee = 0.0 if subtotal >= 50.0 else 5.0

    # Total
    total = subtotal + delivery_fee

    return {
        "subtotal": subtotal,
        "delivery_fee": delivery_fee,
        "total": total
    }


def process_payment(order_id: str, amount: float, customer_name: str) -> Dict:
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
