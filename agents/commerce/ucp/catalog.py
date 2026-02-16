"""UCP catalog endpoint - flavor discovery for external agents."""
from typing import Dict, List
from ..tools.image_gen import check_ingredient_available
from ..tools.address import get_available_delivery_dates


def get_catalog() -> Dict:
    """Get product catalog for UCP agents.

    Returns availability based on current inventory.
    In Phase 3, uses stub check_ingredient_available.
    In Phase 4, will use real A2A calls to Supply Chain.

    Returns:
        Catalog dict with available flavors, nuts, pricing, delivery info
    """
    # Check availability for each item
    flavors = []
    for flavor in ["chocolate", "ananas", "banana"]:
        if check_ingredient_available(flavor):
            flavors.append({
                "id": flavor,
                "name": flavor.capitalize(),
                "available": True
            })

    nuts = []
    for nut in ["almond", "walnut"]:
        if check_ingredient_available(nut):
            nuts.append({
                "id": nut,
                "name": nut.capitalize(),
                "available": True
            })
    nuts.append({"id": "none", "name": "No nuts", "available": True})

    # Get available delivery dates
    delivery_dates = get_available_delivery_dates()

    return {
        "flavors": flavors,
        "nuts": nuts,
        "pricing": {
            "price_per_person": 5.0,
            "currency": "EUR",
            "minimum_people": 6,
            "maximum_people": 50,
            "delivery_fee": {
                "threshold": 50.0,
                "below_threshold": 5.0,
                "above_threshold": 0.0
            }
        },
        "delivery": {
            "area": "Amsterdam (postcodes 1000-1109)",
            "available_dates": delivery_dates
        },
        "languages": ["en", "de", "nl", "tr"]
    }
