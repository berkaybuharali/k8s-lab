"""Maps and routing tools."""
import logging
from typing import Any

try:
    from agents.supply_chain.tools.redis_orders import list_orders
except ImportError:
    from .redis_orders import list_orders

logger = logging.getLogger(__name__)

def get_orders_for_date(date: str) -> list[dict[str, Any]]:
    """
    Get all orders scheduled for delivery on a specific date.
    
    Args:
        date: The delivery date in YYYY-MM-DD format.
        
    Returns:
        list[Dict]: List of orders with address details.
    """
    logger.info(f"Fetching orders for delivery on {date}")
    orders = list_orders(delivery_date=date)
    
    # Filter/Clean data for the agent if needed?
    # The agent needs address, postcode, and maybe order ID to reference.
    # We return the full order object for context.
    
    return orders
