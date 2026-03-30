"""A2A client for communicating with Commerce Concierge system."""
import os
from google.adk.agents.remote_a2a_agent import RemoteA2aAgent


# Singleton instance
_commerce_agent = None


def get_commerce_agent() -> RemoteA2aAgent:
    """Get remote Commerce agent for A2A communication.

    Returns:
        RemoteA2aAgent configured to call commerce.agents.svc:8001
    """
    global _commerce_agent

    if _commerce_agent is None:
        # Get URL from environment (defaults to K8s service DNS)
        commerce_url = os.getenv(
            "COMMERCE_URL",
            "http://commerce.agents.svc.cluster.local:8001"
        )

        _commerce_agent = RemoteA2aAgent(
            name="commerce_remote",
            description="Remote Commerce Concierge system for Magic Cake",
            agent_card=f"{commerce_url}/.well-known/agent-card.json"
        )

    return _commerce_agent


# Tool wrapper for Inventory agent
def notify_out_of_stock(item: str) -> str:
    """Notify Commerce system that an item is out of stock.

    Args:
        item: The ingredient that is out of stock

    Returns:
        Response from Commerce system

    This is called by Inventory agent when stock reaches 0 after deduction.
    Commerce can then update its UI or notify customers.
    """
    valid_items = ["chocolate", "ananas", "banana", "walnut", "almond"]
    if item not in valid_items:
        raise ValueError(f"Invalid item: {item}. Must be one of {valid_items}")

    # Call Commerce via A2A
    agent = get_commerce_agent()
    message = f"Alert: {item} is now out of stock. Please update catalog and notify customers."

    try:
        response = agent.run(message)
        return response
    except Exception as e:
        # Log but don't fail - this is a notification, not critical path
        return f"Failed to notify Commerce: {e}"
