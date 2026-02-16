"""A2A client for communicating with Supply Chain Intelligence system."""
from google.adk.agents.remote_a2a_agent import RemoteA2aAgent


# Phase 4: Implement A2A communication with Supply Chain
# TODO: Create RemoteA2aAgent instance pointing to supply-chain service
# TODO: Implement check_stock(item) - calls Inventory agent
# TODO: Implement deduct_stock(items) - calls Inventory agent
# TODO: Implement create_order(...) - calls Order Service agent
# TODO: Register these as tools on Cake Designer and Checkout agents


def get_supply_chain_agent() -> RemoteA2aAgent:
    """Get remote Supply Chain agent for A2A communication.

    Returns:
        RemoteA2aAgent configured to call supply-chain.agents.svc:8002
    """
    # TODO: Implement in Phase 4
    # return RemoteA2aAgent(
    #     name="supply_chain_remote",
    #     description="Remote Supply Chain Intelligence system",
    #     agent_card="http://supply-chain.agents.svc:8002/.well-known/agent-card.json"
    # )
    raise NotImplementedError("Phase 4: A2A Integration not yet implemented")


# Tool wrappers for Cake Designer agent
def check_ingredient_available(item: str) -> bool:
    """Check if ingredient is in stock via A2A call to Inventory agent.

    Args:
        item: Ingredient name (chocolate, ananas, banana, walnut, almond)

    Returns:
        True if item is in stock (quantity > 0), False otherwise
    """
    # TODO: Phase 4
    # Call supply_chain_agent with: "Check stock for {item}"
    # Parse response for quantity > 0
    raise NotImplementedError("Phase 4: A2A Integration")


# Tool wrappers for Checkout agent
def deduct_inventory(items: list[str]) -> dict:
    """Deduct ingredients from inventory via A2A call.

    Args:
        items: List of items to deduct (e.g., ["chocolate", "walnut"])

    Returns:
        Dict with success status and new quantities
    """
    # TODO: Phase 4
    # Call supply_chain_agent: "Deduct stock: {items}"
    raise NotImplementedError("Phase 4: A2A Integration")


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
    # TODO: Phase 4
    # Call supply_chain_agent: "Create order for {customer_name}..."
    raise NotImplementedError("Phase 4: A2A Integration")
