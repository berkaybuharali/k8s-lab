"""Order management tools using Redis."""
import datetime
import json
import logging
import uuid
from typing import Optional, Any
import redis

# Use absolute import for shared module
try:
    from agents.shared.redis_client import get_redis
    from agents.supply_chain.tools.gcs_images import delete_cake_images
except ImportError:
    from ...shared.redis_client import get_redis
    from .gcs_images import delete_cake_images

logger = logging.getLogger(__name__)

# Constants
PRICE_PER_PERSON = 5
MIN_ORDER_AMOUNT = 30 # Minimum cake order (6 people)
FREE_DELIVERY_THRESHOLD = 50
DELIVERY_FEE = 5
ORDER_KEY_PREFIX = "order:"
ORDER_ID_FORMAT = "CAKE-{date}-{suffix}"

def _generate_order_id() -> str:
    """Generate a unique order ID."""
    today = datetime.date.today().strftime("%Y%m%d")
    r = get_redis()
    
    # Retry loop for collision avoidance
    for _ in range(10):
        suffix = uuid.uuid4().hex[:4].upper()
        order_id = ORDER_ID_FORMAT.format(date=today, suffix=suffix)
        key = f"{ORDER_KEY_PREFIX}{order_id}"
        
        # Check if key exists
        if not r.exists(key):
            return order_id
            
    # Fallback to longer suffix if we keep colliding (unlikely)
    logger.warning("High collision rate for order IDs, using longer suffix")
    suffix = uuid.uuid4().hex[:8].upper()
    return ORDER_ID_FORMAT.format(date=today, suffix=suffix)

def _calculate_price(cakes: list[dict[str, Any]]) -> dict[str, float]:
    """
    Calculate the total price of the order including delivery fee.
    
    Args:
        cakes: List of cake dictionaries, each must have 'people_count'.
        
    Returns:
        Dict: {"total_cake_price": float, "delivery_fee": float, "total_price": float}
    """
    total_cake_price = 0.0
    for cake in cakes:
        people = cake.get("people_count", 0)
        # Ensure minimum people count logic if needed, but price is per person
        total_cake_price += people * PRICE_PER_PERSON
        
    delivery_fee = 0.0
    if total_cake_price < FREE_DELIVERY_THRESHOLD:
        delivery_fee = DELIVERY_FEE
        
    return {
        "total_cake_price": total_cake_price,
        "delivery_fee": delivery_fee,
        "total_price": total_cake_price + delivery_fee
    }

def create_order(
    customer_name: str,
    cakes: list[dict[str, Any]],
    address: str,
    postcode: str,
    delivery_date: str,
    image_paths: list[str] = None
) -> dict[str, Any]:
    """
    Create a new order and store it in Redis.
    
    Args:
        customer_name: Name of the customer.
        cakes: List of cake details (flavor, nuts, people_count, concept).
        address: Delivery address (street + number).
        postcode: Delivery postcode (NNNN XX).
        delivery_date: YYYY-MM-DD string.
        image_paths: List of GCS paths for cake images.
        
    Returns:
        Dict: The created order details including ID and price breakdown.
    """
    # Validate cakes
    for cake in cakes:
        people = cake.get("people_count", 0)
        if people < 6:
            raise ValueError(f"Minimum 6 people per cake (got {people})")
        if people > 50:
            raise ValueError(f"Maximum 50 people per cake (got {people})")

    order_id = _generate_order_id()
    pricing = _calculate_price(cakes)
    
    order_data = {
        "order_id": order_id,
        "customer_name": customer_name,
        "cakes": json.dumps(cakes), # Store complex objects as JSON string
        "address": address,
        "postcode": postcode,
        "delivery_date": delivery_date,
        "created_at": datetime.datetime.now().isoformat(),
        "total_cake_price": pricing["total_cake_price"],
        "delivery_fee": pricing["delivery_fee"],
        "total_price": pricing["total_price"],
        "status": "confirmed", # Default status
        "image_paths": json.dumps(image_paths or [])
    }
    
    r = get_redis()
    try:
        # Store as Hash
        key = f"{ORDER_KEY_PREFIX}{order_id}"
        r.hset(key, mapping=order_data)
        logger.info(f"Created order {order_id} for {customer_name}")
        return order_data
    except redis.RedisError as e:
        logger.error(f"Error creating order: {e}")
        raise

def get_order(order_id: str) -> Optional[dict[str, Any]]:
    """
    Retrieve an order by ID.
    
    Args:
        order_id: The ID of the order.
        
    Returns:
        Dict: The order details or None if not found.
    """
    r = get_redis()
    key = f"{ORDER_KEY_PREFIX}{order_id}"
    try:
        data = r.hgetall(key)
        if not data:
            return None
            
        # Parse JSON fields back to objects
        if "cakes" in data:
            try:
                data["cakes"] = json.loads(data["cakes"])
            except json.JSONDecodeError:
                pass # Keep as string if parsing fails
                
        if "image_paths" in data:
             try:
                data["image_paths"] = json.loads(data["image_paths"])
             except json.JSONDecodeError:
                pass

        # Convert numeric strings to numbers
        for field in ["total_cake_price", "delivery_fee", "total_price"]:
            if field in data:
                try:
                    data[field] = float(data[field])
                except ValueError:
                    pass
                    
        return data
    except redis.RedisError as e:
        logger.error(f"Error getting order {order_id}: {e}")
        return None

def list_orders(delivery_date: Optional[str] = None) -> list[dict[str, Any]]:
    """
    List all orders, optionally filtered by delivery date.
    
    Args:
        delivery_date: Optional YYYY-MM-DD string to filter by.
        
    Returns:
        list[Dict]: List of order details.
    """
    r = get_redis()
    orders = []
    
    try:
        # Use SCAN to find keys matching pattern
        cursor = '0'
        while cursor != 0:
            cursor, keys = r.scan(cursor=cursor, match=f"{ORDER_KEY_PREFIX}*", count=100)
            for key in keys:
                order_id = key.split(":", 1)[1] # remove prefix
                order = get_order(order_id)
                if order:
                    if delivery_date:
                        if order.get("delivery_date") == delivery_date:
                            orders.append(order)
                    else:
                        orders.append(order)
                        
    except redis.RedisError as e:
        logger.error(f"Error listing orders: {e}")
        
    return orders

def delete_order(order_id: str) -> bool:
    """
    Delete an order from Redis and its associated images from GCS.
    
    Args:
        order_id: The ID of the order to delete.
        
    Returns:
        bool: True if deleted, False otherwise.
    """
    # Delete images first (best effort)
    try:
        delete_cake_images(order_id)
    except Exception as e:
        logger.warning(f"Failed to delete images for order {order_id}: {e}")

    r = get_redis()
    key = f"{ORDER_KEY_PREFIX}{order_id}"
    try:
        return r.delete(key) > 0
    except redis.RedisError as e:
        logger.error(f"Error deleting order {order_id}: {e}")
        return False

def get_order_stats() -> dict[str, Any]:
    """
    Get basic statistics about orders.
    
    Returns:
        Dict: {"count": int, "total_revenue": float, "average_order_value": float}
    """
    orders = list_orders()
    count = len(orders)
    total_revenue = sum(o.get("total_price", 0) for o in orders)
    average = total_revenue / count if count > 0 else 0.0
    
    return {
        "count": count,
        "total_revenue": total_revenue,
        "average_order_value": average
    }
