"""Fulfillment agent for delivery route planning."""
import os
import logging
from google import adk
from google.adk.tools.mcp_tool import McpToolset, StdioConnectionParams
from mcp import StdioServerParameters
from supply_chain.tools.maps import get_orders_for_date

logger = logging.getLogger(__name__)

# Define tools list — always include order fetching tool
tools = [get_orders_for_date]

# MCP integration for Google Maps route optimization.
# Requires GOOGLE_API_KEY to be set and Node.js + npx available.
# The @modelcontextprotocol/server-google-maps package is used as the MCP server.
# If GOOGLE_API_KEY is not set, the agent still works but without Maps tools —
# it can fetch orders and describe routes textually but cannot compute real distances.
_google_api_key = os.getenv("GOOGLE_API_KEY")

if _google_api_key:
    # Use McpToolset (preferred over deprecated MCPToolset) to connect to the
    # Google Maps MCP server via stdio. The server is invoked with npx so no
    # separate installation step is needed — npx downloads it on first run.
    maps_toolset = McpToolset(
        connection_params=StdioConnectionParams(
            server_params=StdioServerParameters(
                command="npx",
                args=["-y", "@modelcontextprotocol/server-google-maps"],
                env={"GOOGLE_MAPS_API_KEY": _google_api_key},
            ),
            timeout=30.0,
        ),
        # Limit to route-relevant tools; the package also exposes place search, etc.
        tool_filter=["directions", "distance_matrix", "geocode"],
    )
    tools.append(maps_toolset)
    logger.info("Google Maps MCP toolset enabled for fulfillment agent")
else:
    logger.warning(
        "GOOGLE_API_KEY not set — fulfillment agent will operate without "
        "Google Maps tools. Route optimization will be text-only."
    )

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

When Google Maps tools are available, use them for accurate distances and driving times.
When they are not available, provide estimated routes based on Amsterdam geography.

Tools:
- get_orders_for_date: Get all orders for a specific delivery date (YYYY-MM-DD)
- directions: Get turn-by-turn directions between two points (MCP, if enabled)
- distance_matrix: Calculate travel time/distance for multiple origins/destinations (MCP, if enabled)
- geocode: Convert an address to coordinates (MCP, if enabled)""",
    tools=tools,
)
