"""UCP manifest - capability declaration for external agents."""


def get_ucp_manifest() -> dict:
    """Get UCP capability declaration for Magic Cake.

    Returns:
        UCP manifest dict per /.well-known/ucp spec
    """
    return {
        "name": "Magic Cake Amsterdam",
        "description": "Custom cake ordering and delivery in Amsterdam. AI-generated cake designs with Gemini.",
        "capabilities": ["dev.ucp.shopping", "dev.ucp.checkout"],
        "services": {
            "catalog": {
                "endpoint": "/ucp/catalog",
                "version": "1.0",
                "description": "Browse available cake flavors, nuts, and pricing"
            },
            "checkout": {
                "endpoint": "/ucp/checkout-sessions",
                "version": "1.0",
                "description": "Create and manage cake orders"
            }
        },
        "delivery_area": "Amsterdam, Netherlands (postcodes 1000-1109)",
        "languages": ["en", "de", "nl", "tr"]
    }
