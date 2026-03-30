"""UCP catalog endpoint - flavor discovery for external agents."""
import logging

logger = logging.getLogger(__name__)

_ALL_FLAVORS = ["chocolate", "ananas", "banana"]
_ALL_NUTS = ["almond", "walnut"]


def _check_safe(item: str) -> dict:
    """Check a single ingredient and return availability dict with fallback."""
    try:
        from ..a2a.supply_chain_client import check_ingredient_available
        available = check_ingredient_available(item)
        return {"id": item, "name": item.capitalize(), "available": available}
    except Exception as exc:
        logger.warning("Inventory check failed for %s: %s — defaulting to unknown", item, exc)
        return {"id": item, "name": item.capitalize(), "available": True, "stock_status": "unknown"}


def get_catalog() -> dict:
    """Get product catalog for UCP agents.

    Calls Supply Chain via A2A for real-time stock levels.
    Falls back to available=True, stock_status='unknown' if A2A is unreachable.
    """
    from ..tools.address import get_available_delivery_dates

    flavors = [_check_safe(f) for f in _ALL_FLAVORS]
    nuts = [_check_safe(n) for n in _ALL_NUTS]
    nuts.append({"id": "none", "name": "No nuts", "available": True})

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
