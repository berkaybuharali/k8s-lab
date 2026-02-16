"""Inventory agent for ingredient stock management."""
from google import adk
from supply_chain.tools.redis_stock import (
    check_stock,
    update_stock,
    list_all_stock,
    list_low_stock,
)
from supply_chain.a2a.commerce_client import notify_out_of_stock

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
- Notify Commerce system via A2A when stock reaches 0
- Maintain stock logs

Tools:
- check_stock: Check quantity of an item
- update_stock: Change quantity of an item (positive/negative)
- list_all_stock: Get all stock levels
- list_low_stock: Get items with stock <= threshold
- notify_out_of_stock: Notify Commerce when item reaches 0 (via A2A)

IMPORTANT: When stock reaches 0 after deduction, call notify_out_of_stock.""",
    tools=[check_stock, update_stock, list_all_stock, list_low_stock, notify_out_of_stock],
)
