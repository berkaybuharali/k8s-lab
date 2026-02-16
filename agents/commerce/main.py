"""Magic Cake Commerce Concierge (System A).

A2A server on port 8001 + UCP endpoints with three agents:
- Translation: Language selection (EN/DE/NL/TR)
- Cake Designer: Cake preferences + Imagen generation
- Checkout: Address, delivery, payment, order creation
"""
import os
from starlette.responses import JSONResponse
from starlette.routing import Route
from google.adk.a2a.utils.agent_to_a2a import to_a2a
from commerce.agent import root_agent

# Expose agent via A2A protocol
# Creates endpoints: /.well-known/agent-card.json and A2A RPC
app = to_a2a(root_agent, port=8001)

# Add health check endpoint for Kubernetes probes
async def health(request):
    return JSONResponse({"status": "healthy"})

app.add_route("/health", health, methods=["GET"])

# Phase 3: Add UCP endpoints - app.add_api_route("/ucp/...", ...)
# Phase 4: Update agent.py to consume supply-chain via RemoteA2aAgent


if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("PORT", "8001"))
    uvicorn.run(app, host="0.0.0.0", port=port)
