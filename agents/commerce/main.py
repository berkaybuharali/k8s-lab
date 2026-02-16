"""Magic Cake Commerce Concierge (System A).

A2A server on port 8001 + UCP endpoints with three agents:
- Translation: Language selection (EN/DE/NL/TR)
- Cake Designer: Cake preferences + Gemini image generation
- Checkout: Address, delivery, payment, order creation
"""
import os
from starlette.responses import JSONResponse
from starlette.requests import Request
from google.adk.a2a.utils.agent_to_a2a import to_a2a
from commerce.agent import root_agent
from commerce.ucp.manifest import get_ucp_manifest
from commerce.ucp.catalog import get_catalog
from commerce.ucp.sessions import (
    create_session,
    update_session,
    get_session,
    complete_session
)

# Expose agent via A2A protocol
# Creates endpoints: /.well-known/agent-card.json and A2A RPC
app = to_a2a(root_agent, port=8001)


# Health check endpoint for Kubernetes probes
async def health(request: Request):
    return JSONResponse({"status": "healthy"})


# UCP Discovery endpoint
async def ucp_manifest(request: Request):
    return JSONResponse(get_ucp_manifest())


# UCP Catalog endpoint
async def ucp_catalog(request: Request):
    return JSONResponse(get_catalog())


# UCP Session endpoints
async def ucp_create_session(request: Request):
    try:
        data = await request.json()
        customer_name = data.get("customer_name")
        cakes = data.get("cakes", [])

        if not customer_name:
            return JSONResponse({"error": "customer_name is required"}, status_code=400)

        session = create_session(customer_name, cakes)
        return JSONResponse(session)
    except ValueError as e:
        return JSONResponse({"error": str(e)}, status_code=400)
    except Exception as e:
        return JSONResponse({"error": f"Internal error: {e}"}, status_code=500)


async def ucp_update_session(request: Request):
    try:
        session_id = request.path_params["session_id"]
        data = await request.json()

        session = update_session(
            session_id,
            delivery_date=data.get("delivery_date"),
            postcode=data.get("postcode"),
            house_number=data.get("house_number")
        )
        return JSONResponse(session)
    except ValueError as e:
        return JSONResponse({"error": str(e)}, status_code=400)
    except Exception as e:
        return JSONResponse({"error": f"Internal error: {e}"}, status_code=500)


async def ucp_get_session(request: Request):
    try:
        session_id = request.path_params["session_id"]
        session = get_session(session_id)
        return JSONResponse(session)
    except ValueError as e:
        return JSONResponse({"error": str(e)}, status_code=404)
    except Exception as e:
        return JSONResponse({"error": f"Internal error: {e}"}, status_code=500)


async def ucp_complete_session(request: Request):
    try:
        session_id = request.path_params["session_id"]
        result = complete_session(session_id)
        return JSONResponse(result)
    except ValueError as e:
        return JSONResponse({"error": str(e)}, status_code=400)
    except Exception as e:
        return JSONResponse({"error": f"Internal error: {e}"}, status_code=500)


# Register routes
app.add_route("/health", health, methods=["GET"])
app.add_route("/.well-known/ucp", ucp_manifest, methods=["GET"])
app.add_route("/ucp/catalog", ucp_catalog, methods=["GET"])
app.add_route("/ucp/checkout-sessions", ucp_create_session, methods=["POST"])
app.add_route("/ucp/checkout-sessions/{session_id:path}", ucp_update_session, methods=["PUT"])
app.add_route("/ucp/checkout-sessions/{session_id:path}", ucp_get_session, methods=["GET"])
app.add_route("/ucp/checkout-sessions/{session_id:path}/complete", ucp_complete_session, methods=["POST"])

# Phase 4: Update agent.py to consume supply-chain via RemoteA2aAgent


if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("PORT", "8001"))
    uvicorn.run(app, host="0.0.0.0", port=port)
