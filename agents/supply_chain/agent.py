"""Supply Chain root agent definition."""
from google import adk
from .agents.inventory import inventory_agent
from .agents.order_service import order_service_agent
from .agents.fulfillment import fulfillment_agent


# Root orchestrator agent
root_agent = adk.Agent(
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
