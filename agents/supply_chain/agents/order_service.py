"""Order Service agent for order management."""
from google import adk
from supply_chain.tools.redis_orders import (
    create_order,
    get_order,
    list_orders,
    delete_order,
    get_order_stats,
)
from supply_chain.tools.gcs_images import get_cake_image_urls

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

Tools:
- create_order: Create a new order
- get_order: Get order details by ID
- list_orders: List orders, optionally by date
- delete_order: Delete an order and its images
- get_order_stats: Get summary statistics
- get_cake_image_urls: Get signed URLs for order images""",
    tools=[
        create_order,
        get_order,
        list_orders,
        delete_order,
        get_order_stats,
        get_cake_image_urls,
    ],
)
