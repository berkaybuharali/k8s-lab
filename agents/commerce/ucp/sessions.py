"""UCP checkout session management."""
from typing import Dict, List, Optional
from datetime import datetime
import uuid
from ..tools.payment import calculate_price
from ..tools.address import validate_amsterdam_address, get_available_delivery_dates
from ..tools.gemini_image import generate_cake_image
from ..a2a.supply_chain_client import deduct_inventory, create_order_remote


# In-memory session storage (replace with Redis in production)
_sessions: Dict[str, Dict] = {}


def create_session(customer_name: str, cakes: List[Dict]) -> Dict:
    """Create new checkout session.

    Args:
        customer_name: Customer name
        cakes: List of cake dicts with flavor, nuts, people_count, concept

    Returns:
        Session dict with session_id, pricing, available_dates
    """
    # Validate cakes
    if not cakes:
        raise ValueError("At least one cake is required")

    for cake in cakes:
        required_fields = ["flavor", "nuts", "people_count", "concept"]
        for field in required_fields:
            if field not in cake:
                raise ValueError(f"Missing required field: {field}")

    # Generate session ID
    session_id = str(uuid.uuid4())

    # Calculate pricing
    pricing = calculate_price(cakes)

    # Get available delivery dates
    available_dates = get_available_delivery_dates()

    # Create session
    session = {
        "session_id": session_id,
        "customer_name": customer_name,
        "cakes": cakes,
        "pricing": pricing,
        "delivery": {
            "available_dates": available_dates,
            "selected_date": None,
            "address": None,
            "postcode": None
        },
        "status": "created",
        "created_at": datetime.now().isoformat()
    }

    # Store session
    _sessions[session_id] = session

    return session


def update_session(
    session_id: str,
    delivery_date: Optional[str] = None,
    postcode: Optional[str] = None,
    house_number: Optional[str] = None
) -> Dict:
    """Update session with delivery details.

    Args:
        session_id: Session ID
        delivery_date: YYYY-MM-DD
        postcode: Dutch postcode
        house_number: House number

    Returns:
        Updated session dict
    """
    if session_id not in _sessions:
        raise ValueError(f"Session not found: {session_id}")

    session = _sessions[session_id]

    # Update delivery date
    if delivery_date:
        available_dates = session["delivery"]["available_dates"]
        if delivery_date not in available_dates:
            raise ValueError(f"Invalid delivery date. Must be one of: {available_dates}")
        session["delivery"]["selected_date"] = delivery_date

    # Update address
    if postcode and house_number:
        validation = validate_amsterdam_address(postcode, house_number)
        if not validation["valid"]:
            raise ValueError(validation["error"])

        session["delivery"]["address"] = validation["formatted_address"]
        session["delivery"]["postcode"] = postcode

    session["status"] = "updated"
    return session


def get_session(session_id: str) -> Dict:
    """Get session by ID.

    Args:
        session_id: Session ID

    Returns:
        Session dict

    Raises:
        ValueError: If session not found
    """
    if session_id not in _sessions:
        raise ValueError(f"Session not found: {session_id}")

    return _sessions[session_id]


def complete_session(session_id: str) -> Dict:
    """Complete session and create order.

    Phase 4: Real A2A integration with Supply Chain

    Args:
        session_id: Session ID

    Returns:
        Order confirmation dict
    """
    if session_id not in _sessions:
        raise ValueError(f"Session not found: {session_id}")

    session = _sessions[session_id]

    # Validate session is ready
    if not session["delivery"]["selected_date"]:
        raise ValueError("Delivery date is required")

    if not session["delivery"]["address"]:
        raise ValueError("Delivery address is required")

    # Generate order ID
    order_id = f"CAKE-{datetime.now().strftime('%Y%m%d')}-{uuid.uuid4().hex[:4].upper()}"

    # Step 1: Generate cake images
    image_paths = []
    for i, cake in enumerate(session["cakes"], 1):
        try:
            image_path = generate_cake_image(
                flavor=cake["flavor"],
                nuts=cake["nuts"],
                people_count=cake["people_count"],
                concept=cake["concept"],
                order_id=order_id,
                cake_number=i
            )
            image_paths.append(image_path)
        except Exception as e:
            raise RuntimeError(f"Failed to generate image for cake {i}: {e}")

    # Step 2: Collect ingredients to deduct
    ingredients = []
    for cake in session["cakes"]:
        ingredients.append(cake["flavor"])
        if cake["nuts"] != "none":
            ingredients.append(cake["nuts"])

    # Step 3: Deduct inventory via A2A
    try:
        deduct_inventory(ingredients)
    except Exception as e:
        raise RuntimeError(f"Failed to deduct inventory: {e}")

    # Step 4: Create order via A2A
    try:
        order_result = create_order_remote(
            customer_name=session["customer_name"],
            cakes=session["cakes"],
            address=session["delivery"]["address"],
            postcode=session["delivery"]["postcode"],
            delivery_date=session["delivery"]["selected_date"],
            image_paths=image_paths
        )
    except Exception as e:
        raise RuntimeError(f"Failed to create order: {e}")

    # Update session
    session["status"] = "completed"
    session["order_id"] = order_id
    session["image_paths"] = image_paths

    return {
        "success": True,
        "order_id": order_id,
        "customer_name": session["customer_name"],
        "cakes": session["cakes"],
        "total": session["pricing"]["total"],
        "delivery_date": session["delivery"]["selected_date"],
        "delivery_address": session["delivery"]["address"],
        "image_paths": image_paths,
        "message": f"Order confirmed! Delivery on {session['delivery']['selected_date']}"
    }
