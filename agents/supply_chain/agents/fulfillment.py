"""Fulfillment agent for delivery route planning."""
import os
from google import adk
from agents.supply_chain.tools.maps import get_orders_for_date

# Define tools list
tools = [get_orders_for_date]

# Add MCP Toolset if configured
if os.getenv("GOOGLE_MAPS_API_KEY"):
    from google.adk.toolsets import MCPToolset
    maps_mcp = MCPToolset(
        server_config={
            "command": "npx",
            "args": ["-y", "@modelcontextprotocol/server-google-maps"],
            "env": {"GOOGLE_MAPS_API_KEY": os.getenv("GOOGLE_MAPS_API_KEY")}
        }
    )
    tools.append(maps_mcp)

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
