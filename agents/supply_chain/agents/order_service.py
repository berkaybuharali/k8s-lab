"""Order Service agent for order management."""
from google import adk

order_service_agent = adk.Agent(
    name="order_service",
    model="gemini-2.5-flash",
    instruction="""You manage orders for Magic Cake.

Order structure:
- Order ID: CAKE-YYYYMMDD-XXXX
- Customer name, delivery address (Amsterdam only), delivery date
- Cakes: array of {flavor, nuts, people_count, concept}
- Pricing: people_count × 5 EUR per cake + delivery fee (5 EUR if total < 50 EUR)
- Images: GCS paths for cake images (cake_1.png, cake_2.png, ...)

Your responsibilities:
- Create new orders with all details
- Retrieve order information
- List orders by delivery date
- Delete orders (including images)
- Provide order statistics (count, revenue, average)

Tools will be added in Phase 2.""",
    tools=[],  # Tools added in Phase 2
)
