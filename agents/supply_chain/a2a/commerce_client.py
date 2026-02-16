"""A2A client for communicating with Commerce Concierge system."""
from google.adk.agents.remote_a2a_agent import RemoteA2aAgent


# Phase 4: Implement A2A communication with Commerce
# TODO: Create RemoteA2aAgent instance pointing to commerce service
# TODO: Implement notify_out_of_stock(item) - alerts Commerce when item hits 0
# TODO: Register as tool on Inventory agent


def get_commerce_agent() -> RemoteA2aAgent:
    """Get remote Commerce agent for A2A communication.

    Returns:
        RemoteA2aAgent configured to call commerce.agents.svc:8001
    """
    # TODO: Implement in Phase 4
    # return RemoteA2aAgent(
    #     name="commerce_remote",
    #     description="Remote Commerce Concierge system",
    #     agent_card="http://commerce.agents.svc:8001/.well-known/agent-card.json"
    # )
    raise NotImplementedError("Phase 4: A2A Integration not yet implemented")


# Tool wrapper for Inventory agent
def notify_out_of_stock(item: str) -> None:
    """Notify Commerce system that an item is out of stock.

    Args:
        item: The ingredient that is out of stock

    This is called by Inventory agent when stock reaches 0 after deduction.
    Commerce can then update its UI or notify customers.
    """
    # TODO: Phase 4
    # Call commerce_agent: "Item out of stock: {item}"
    # Commerce may want to:
    # - Update catalog to hide this ingredient
    # - Notify active shoppers
    # - Log for backoffice dashboard
    raise NotImplementedError("Phase 4: A2A Integration")
