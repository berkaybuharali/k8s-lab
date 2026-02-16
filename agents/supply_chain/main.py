"""Magic Cake Supply Chain Intelligence (System B).

A2A server on port 8002 with three agents:
- Inventory: Stock management for ingredients
- Order Service: Order CRUD and image references
- Fulfillment: Route planning with Google Maps MCP
"""
import sys
from pathlib import Path

# Add shared package to path
sys.path.insert(0, str(Path(__file__).parent.parent / "shared"))

from google import adk
from .agents.inventory import inventory_agent
from .agents.order_service import order_service_agent
from .agents.fulfillment import fulfillment_agent


# Root orchestrator agent
supply_chain_root = adk.Agent(
    name="supply_chain",
    model="gemini-2.5-pro",
    instruction="""You are the Supply Chain Intelligence system for Magic Cake.
You coordinate three specialized agents:
- Inventory agent: manages ingredient stock (chocolate, ananas, banana, walnut, almond)
- Order Service agent: handles order storage and retrieval
- Fulfillment agent: plans delivery routes from Danzigerkade 4, Amsterdam

Route requests to the appropriate agent based on the task.""",
    sub_agents=[inventory_agent, order_service_agent, fulfillment_agent],
)


def main():
    """Start A2A server on port 8002."""
    # Create Flask app with ADK
    app = adk.create_app(agent=supply_chain_root)

    # Run the server
    app.run(host="0.0.0.0", port=8002, debug=False)


if __name__ == "__main__":
    main()
