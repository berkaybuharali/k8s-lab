"""UCP manifest - capability declaration for external agents.

Phase 3 will implement /.well-known/ucp endpoint.
"""

UCP_MANIFEST = {
    "name": "Magic Cake Amsterdam",
    "description": "Custom cake ordering and delivery in Amsterdam",
    "capabilities": ["dev.ucp.shopping", "dev.ucp.checkout"],
    "services": {
        "catalog": {"endpoint": "/ucp/catalog", "version": "1.0"},
        "checkout": {"endpoint": "/ucp/checkout-sessions", "version": "1.0"},
    },
}
