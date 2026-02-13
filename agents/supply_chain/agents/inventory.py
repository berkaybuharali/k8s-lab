"""Inventory agent for ingredient stock management."""
from google import adk

inventory_agent = adk.Agent(
    name="inventory",
    model="gemini-2.5-flash",
    instruction="""You manage ingredient inventory for Magic Cake.

Available ingredients: chocolate, ananas, banana, walnut, almond.
Maximum stock per ingredient: 5.

Your responsibilities:
- Check stock availability for specific ingredients
- Update stock levels when ingredients are used
- Alert when stock is low (threshold: 2 or below)
- Maintain stock logs

Tools will be added in Phase 2.""",
    tools=[],  # Tools added in Phase 2
)
