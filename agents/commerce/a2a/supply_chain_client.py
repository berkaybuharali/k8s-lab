"""A2A client for communicating with Supply Chain Intelligence system."""
import os
from google.adk.agents.remote_a2a_agent import RemoteA2aAgent


# Singleton instance
_supply_chain_agent = None


def get_supply_chain_agent() -> RemoteA2aAgent:
    """Get remote Supply Chain agent for A2A communication.

    Returns:
        RemoteA2aAgent configured to call supply-chain.agents.svc:8002
    """
    global _supply_chain_agent

    if _supply_chain_agent is None:
        # Get URL from environment (defaults to K8s service DNS)
        supply_chain_url = os.getenv(
            "SUPPLY_CHAIN_URL",
            "http://supply-chain.agents.svc.cluster.local:8002"
        )

        _supply_chain_agent = RemoteA2aAgent(
            name="supply_chain_remote",
            description="Remote Supply Chain Intelligence system for Magic Cake",
            agent_card=f"{supply_chain_url}/.well-known/agent-card.json"
        )

    return _supply_chain_agent


# Tool wrappers for Cake Designer agent
def check_ingredient_available(item: str) -> bool:
    """Check if ingredient is in stock via A2A call to Inventory agent.

    Args:
        item: Ingredient name (chocolate, ananas, banana, walnut, almond)

    Returns:
        True if item is in stock (quantity > 0), False otherwise
    """
    valid_items = ["chocolate", "ananas", "banana", "walnut", "almond"]
    if item not in valid_items:
        raise ValueError(f"Invalid item: {item}. Must be one of {valid_items}")

    # Call Supply Chain via A2A
    agent = get_supply_chain_agent()
    response = agent.run(f"Check stock for {item}")

    # Parse response - looking for quantity information
    # The Inventory agent will respond with stock info
    response_text = response.lower()

    # Check if the response indicates availability
    # Patterns: "in stock", "available", "quantity: X" where X > 0
    if "out of stock" in response_text or "unavailable" in response_text or "quantity: 0" in response_text:
        return False

    return True


# Tool wrappers for Checkout agent
def deduct_inventory(items: list[str]) -> dict:
    """Deduct ingredients from inventory via A2A call.

    Args:
        items: List of items to deduct (e.g., ["chocolate", "walnut"])

    Returns:
        Dict with success status and new quantities
    """
    if not items:
        raise ValueError("Items list cannot be empty")

    # Build natural language request
    items_str = ", ".join(items)
    message = f"Deduct 1 unit of each: {items_str}"

    # Call Supply Chain via A2A
    agent = get_supply_chain_agent()
    response = agent.run(message)

    # Return simplified response
    return {
        "success": True,
        "message": response,
        "items_deducted": items
    }


def create_order_remote(
    customer_name: str,
    cakes: list[dict],
    address: str,
    postcode: str,
    delivery_date: str,
    image_paths: list[str],
) -> dict:
    """Create order via A2A call to Order Service agent.

    Args:
        customer_name: Customer name
        cakes: List of cake dicts with flavor, nuts, people_count, concept
        address: Delivery address
        postcode: Postcode
        delivery_date: YYYY-MM-DD
        image_paths: GCS paths to cake images

    Returns:
        Order details including order_id and pricing
    """
    # Build cake descriptions
    cake_descriptions = []
    for i, cake in enumerate(cakes, 1):
        desc = (
            f"Cake {i}: {cake['flavor']} cake for {cake['people_count']} people, "
            f"{cake['nuts']} nuts, {cake['concept']} theme"
        )
        cake_descriptions.append(desc)

    # Build natural language request
    cakes_text = "; ".join(cake_descriptions)
    images_text = ", ".join(image_paths)

    message = (
        f"Create order for {customer_name}. "
        f"Cakes: {cakes_text}. "
        f"Delivery: {address}, {postcode} on {delivery_date}. "
        f"Images: {images_text}"
    )

    # Call Supply Chain via A2A
    agent = get_supply_chain_agent()
    response = agent.run(message)

    # Parse order ID from response
    # The Order Service agent should return order ID in response
    order_id = None
    if "CAKE-" in response:
        # Extract order ID pattern: CAKE-YYYYMMDD-XXXX
        import re
        match = re.search(r'CAKE-\d{8}-[A-F0-9]{4}', response)
        if match:
            order_id = match.group(0)

    return {
        "success": True,
        "order_id": order_id,
        "message": response,
        "customer_name": customer_name,
        "cakes": cakes,
        "address": address,
        "delivery_date": delivery_date,
        "image_paths": image_paths
    }
