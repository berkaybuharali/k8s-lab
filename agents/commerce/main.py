"""Magic Cake Commerce Concierge (System A).

A2A server on port 8001 + UCP endpoints with three agents:
- Translation: Language selection (EN/DE/NL/TR)
- Cake Designer: Cake preferences + Imagen generation
- Checkout: Address, delivery, payment, order creation
"""
import sys
from pathlib import Path

# Add shared package to path
sys.path.insert(0, str(Path(__file__).parent.parent / "shared"))

from google import adk
from .agents.translation import translation_agent
from .agents.cake_designer import cake_designer_agent
from .agents.checkout import checkout_agent


# Root orchestrator agent
commerce_root = adk.Agent(
    name="commerce",
    model="gemini-2.5-pro",
    instruction="""You are the Commerce Concierge system for Magic Cake.
You coordinate three specialized agents:
- Translation agent: handles language selection (English, German, Dutch, Turkish)
- Cake Designer agent: helps customers design cakes and generates images
- Checkout agent: manages delivery, payment, and order finalization

Start with the Translation agent for new customers, then hand off to Cake Designer,
and finally to Checkout for order completion.

Pricing: 5 EUR per slice (per person). Minimum 6 people. Delivery: 5 EUR if order < 50 EUR.
Multiple cakes per order are allowed.""",
    sub_agents=[translation_agent, cake_designer_agent, checkout_agent],
)


def main():
    """Start A2A server on port 8001 + UCP endpoints."""
    # Phase 3 will add UCP endpoints here
    adk.run_server(
        app_name="commerce",
        root_agent=commerce_root,
        host="0.0.0.0",
        port=8001,
    )


if __name__ == "__main__":
    main()
