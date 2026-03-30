"""Magic Cake Supply Chain Intelligence (System B).

A2A server on port 8002 with three agents:
- Inventory: Stock management for ingredients
- Order Service: Order CRUD and image references
- Fulfillment: Route planning with Google Maps MCP
"""
import os
from starlette.responses import JSONResponse
from starlette.routing import Route
from google.adk.a2a.utils.agent_to_a2a import to_a2a
from supply_chain.agent import root_agent

# Expose agent via A2A protocol
# Creates endpoints: /.well-known/agent-card.json and A2A RPC
app = to_a2a(root_agent, port=8002)

# Add health check endpoint for Kubernetes probes
async def health(request):
    return JSONResponse({"status": "healthy"})

app.add_route("/health", health, methods=["GET"])


if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("PORT", "8002"))
    uvicorn.run(app, host="0.0.0.0", port=port)
