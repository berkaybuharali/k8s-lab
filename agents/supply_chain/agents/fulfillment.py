"""Fulfillment agent for delivery route planning."""
import os
from google import adk
from supply_chain.tools.maps import get_orders_for_date

# Define tools list
tools = [get_orders_for_date]

# TODO: MCP integration for Google Maps
# MCPToolset doesn't exist in ADK 1.25.0
# Phase 3 will implement proper MCP integration for route optimization
# For now, the agent can fetch orders but won't have Maps API access
if os.getenv("GOOGLE_MAPS_API_KEY"):
    # Placeholder for future MCP integration
    # See: https://github.com/modelcontextprotocol/python-sdk
    pass

fulfillment_agent = adk.Agent(
    name="fulfillment",
    model="gemini-2.5-pro",
    instruction="""You are a delivery route planner for Magic Cake.

Fulfillment center: Danzigerkade 4, 1013 AP Amsterdam

Your responsibilities:
- Plan optimal delivery routes for a given date
- Calculate delivery times
- Optimize multi-stop routes from our fulfillment center
- Provide route visualizations for the backoffice map

You use Google Maps MCP for route calculation, distance matrix, and geocoding.

Tools:
- get_orders_for_date: Get all orders for a specific delivery date (YYYY-MM-DD)
- (MCP tools for Maps will be available)""",
    tools=tools,
)
