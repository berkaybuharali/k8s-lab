"""A2A client for communicating with Supply Chain Intelligence system.

Uses direct HTTP A2A calls (JSON-RPC over httpx) instead of ADK's RemoteA2aAgent.
RemoteA2aAgent participates in ADK session management and corrupts the active
agent turn's session when called from within a tool. Direct HTTP bypasses that
while still respecting Supply Chain as the source of truth via A2A protocol.
"""
import os
import uuid
import httpx

VALID_ITEMS = ["chocolate", "ananas", "banana", "walnut", "almond"]
_DEFAULT_SUPPLY_CHAIN_URL = "http://supply-chain.agents.svc.cluster.local:8002"


def _supply_chain_url() -> str:
    return os.getenv("SUPPLY_CHAIN_URL", _DEFAULT_SUPPLY_CHAIN_URL)


def _call_supply_chain(message: str) -> str:
    """Send a message to Supply Chain via A2A JSON-RPC and return the response text."""
    payload = {
        "jsonrpc": "2.0",
        "method": "message/send",
        "id": "1",
        "params": {
            "message": {
                "role": "user",
                "parts": [{"kind": "text", "text": message}],
                "messageId": str(uuid.uuid4()),
            }
        },
    }
    response = httpx.post(f"{_supply_chain_url()}/", json=payload, timeout=15.0)
    response.raise_for_status()
    data = response.json()
    status = data.get("result", {}).get("status", {})
    if status.get("state") == "failed":
        raise RuntimeError(f"A2A call failed: {status}")
    result = data.get("result", {})
    return _extract_text(result)


def _extract_text(result: dict) -> str:
    """Extract agent response text from A2A result (artifacts first, then history)."""
    for artifact in result.get("artifacts", []):
        for part in artifact.get("parts", []):
            if part.get("kind") == "text" and part.get("text"):
                return part["text"]

    history = result.get("history", [])
    last_user_idx = next(
        (i for i in range(len(history) - 1, -1, -1) if history[i].get("role") == "user"),
        -1,
    )
    for msg in history[last_user_idx + 1:]:
        if msg.get("role") == "agent":
            for part in msg.get("parts", []):
                if part.get("kind") == "text" and part.get("text"):
                    return part["text"]
    return ""


def check_ingredient_available(item: str) -> bool:
    """Check if ingredient is in stock via A2A call to Supply Chain Inventory agent.

    Args:
        item: Ingredient name (chocolate, ananas, banana, walnut, almond)

    Returns:
        True if item is in stock (quantity > 0), False otherwise
    """
    if item.lower() not in VALID_ITEMS:
        return False

    try:
        text = _call_supply_chain(f"Check stock for {item}").lower()
        if "out of stock" in text or "unavailable" in text or "quantity: 0" in text:
            return False
        return True
    except Exception:
        return True


# Tool wrappers for Checkout agent
def deduct_inventory(items: list[str]) -> dict:
    """Deduct ingredients from inventory via A2A call to Supply Chain Inventory agent.

    Args:
        items: List of items to deduct (e.g., ["chocolate", "walnut"])

    Returns:
        Dict with success status and message
    """
    if not items:
        raise ValueError("Items list cannot be empty")

    items_str = ", ".join(items)
    try:
        message = _call_supply_chain(f"Deduct 1 unit of each: {items_str}")
        return {"success": True, "message": message, "items_deducted": items}
    except Exception as e:
        return {"success": False, "message": str(e), "items_deducted": []}


def create_order_remote(
    customer_name: str,
    flavors: list[str],
    nuts_choices: list[str],
    people_counts: list[int],
    concepts: list[str],
    address: str,
    postcode: str,
    delivery_date: str,
    image_paths: list[str],
) -> dict:
    """Create order via A2A call to Supply Chain Order Service agent.

    Args:
        customer_name: Customer full name
        flavors: Cake flavor per cake, e.g. ["ananas", "chocolate"]
        nuts_choices: Nut topping per cake, e.g. ["walnut", "none"]
        people_counts: Number of people per cake, e.g. [10, 8]
        concepts: Theme/concept per cake, e.g. ["birthday", "wedding"]
        address: Street name and house number, e.g. "Keizersgracht 123"
        postcode: Amsterdam postcode, e.g. "1015 CJ"
        delivery_date: YYYY-MM-DD
        image_paths: GCS path per cake, e.g. ["gs://bucket/cakes/..."]

    Returns:
        Order details including order_id
    """
    import re

    message = (
        f"Create order. "
        f"customer_name={customer_name}. "
        f"flavors={flavors}. "
        f"nuts_choices={nuts_choices}. "
        f"people_counts={people_counts}. "
        f"concepts={concepts}. "
        f"address={address}. "
        f"postcode={postcode}. "
        f"delivery_date={delivery_date}. "
        f"image_paths={image_paths}."
    )

    try:
        response = _call_supply_chain(message)
        order_id = None
        match = re.search(r'CAKE-\d{8}-[A-F0-9]{4}', response)
        if match:
            order_id = match.group(0)
        return {
            "success": True,
            "order_id": order_id,
            "message": response,
            "customer_name": customer_name,
            "address": address,
            "delivery_date": delivery_date,
            "image_paths": image_paths,
        }
    except Exception as e:
        return {"success": False, "message": str(e), "order_id": None}
