"""Inventory management tools using Redis."""
import logging
import redis

# Use absolute import for shared module when running from root
try:
    from agents.shared.redis_client import get_redis
except ImportError:
    # Fallback for local testing if needed
    from ...shared.redis_client import get_redis

logger = logging.getLogger(__name__)

# Constants
MAX_STOCK = 5
INGREDIENTS = ["chocolate", "ananas", "banana", "walnut", "almond"]
INVENTORY_KEY_PREFIX = "inventory:"
INVENTORY_LOG_KEY = "inventory:log"

def _get_inventory_key(item: str) -> str:
    """Get the Redis key for an inventory item."""
    return f"{INVENTORY_KEY_PREFIX}{item.lower()}"

def check_stock(item: str) -> int:
    """
    Check the current stock level of an ingredient.
    
    Args:
        item: The name of the ingredient (chocolate, ananas, banana, walnut, almond).
        
    Returns:
        int: The current quantity in stock (0-5). Returns 0 if item not found.
    """
    if item.lower() not in INGREDIENTS:
        logger.warning(f"Check stock requested for unknown item: {item}")
        return 0
        
    r = get_redis()
    try:
        # Use HGET to get the quantity field from the hash
        qty = r.hget(_get_inventory_key(item), "quantity")
        return int(qty) if qty is not None else 0
    except (redis.RedisError, ValueError) as e:
        logger.error(f"Error checking stock for {item}: {e}")
        return 0

def update_stock(item: str, quantity_change: int, reason: str = "manual_update") -> int:
    """
    Update the stock level of an ingredient.
    
    Args:
        item: The name of the ingredient.
        quantity_change: The amount to change (positive to add, negative to remove).
        reason: The reason for the update (e.g., "order_123", "restock").
        
    Returns:
        int: The new quantity in stock.
        
    Raises:
        ValueError: If item is unknown or resulting stock would be invalid (<0 or >5).
    """
    item = item.lower()
    if item not in INGREDIENTS:
        raise ValueError(f"Unknown ingredient: {item}")
        
    r = get_redis()
    key = _get_inventory_key(item)
    
    # specific logic for atomic update with constraints
    # We use a Lua script or simple check-then-set for simplicity in this PoC
    # standard Redis transaction (pipeline) is better for concurrency
    
    pipe = r.pipeline()
    max_retries = 20
    attempts = 0
    
    while attempts < max_retries:
        try:
            pipe.watch(key)
            # Use HGET for hash
            current_qty = pipe.hget(key, "quantity")
            current_qty = int(current_qty) if current_qty is not None else 0
            
            new_qty = current_qty + quantity_change
            
            if new_qty < 0:
                pipe.unwatch()
                raise ValueError(f"Insufficient stock for {item}. Current: {current_qty}, Requested change: {quantity_change}")
                
            if new_qty > MAX_STOCK:
                pipe.unwatch()
                raise ValueError(f"Stock limit exceeded for {item}. Current: {current_qty}, Requested change: {quantity_change}, Max: {MAX_STOCK}")
                
            pipe.multi()
            # Use HSET for hash
            pipe.hset(key, "quantity", new_qty)
            # Log the change
            log_entry = f"{item}:{quantity_change}:{reason}:{new_qty}"
            pipe.rpush(INVENTORY_LOG_KEY, log_entry)
            # Trim log to keep last 1000 entries
            pipe.ltrim(INVENTORY_LOG_KEY, -1000, -1)
            
            pipe.execute()
            logger.info(f"Updated stock for {item}: {current_qty} -> {new_qty} ({reason})")
            return new_qty
            
        except redis.WatchError:
            # Retry if key changed during watch
            attempts += 1
            continue
        except redis.RedisError as e:
            logger.error(f"Redis error updating stock for {item}: {e}")
            raise
            
    raise RuntimeError(f"Failed to update stock for {item} after {max_retries} retries due to contention")

def list_all_stock() -> dict[str, int]:
    """
    Get the stock levels for all ingredients.
    
    Returns:
        dict[str, int]: A dictionary mapping ingredient names to their quantities.
    """
    r = get_redis()
    result = {}
    
    try:
        # Pipeline the HGET calls for efficiency since MGET doesn't work on hashes easily
        # Alternatively we could iterate, but pipeline is better
        pipe = r.pipeline()
        for item in INGREDIENTS:
            pipe.hget(_get_inventory_key(item), "quantity")
        
        values = pipe.execute()
        
        for item, value in zip(INGREDIENTS, values):
            result[item] = int(value) if value is not None else 0
            
    except redis.RedisError as e:
        logger.error(f"Error listing all stock: {e}")
        # Return what we can or empty? Let's return defaults (0) for safety
        for item in INGREDIENTS:
            if item not in result:
                result[item] = 0
                
    return result

def list_low_stock(threshold: int = 2) -> dict[str, int]:
    """
    List ingredients with stock at or below the threshold.
    
    Args:
        threshold: The quantity threshold (inclusive). Defaults to 2.
        
    Returns:
        dict[str, int]: A dictionary of low-stock ingredients and their quantities.
    """
    all_stock = list_all_stock()
    return {item: qty for item, qty in all_stock.items() if qty <= threshold}
