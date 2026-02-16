"""Image generation tool using Gemini 2.5 Flash Image (Nano Banana)."""
import os
import base64
import vertexai
from vertexai.generative_models import GenerativeModel
from google.cloud import storage
from io import BytesIO


def generate_cake_image(
    flavor: str,
    nuts: str,
    people_count: int,
    concept: str,
    order_id: str,
    cake_number: int = 1
) -> str:
    """Generate cake image using Gemini 2.5 Flash Image and upload to GCS.

    Uses GCP service account credentials (no separate API key needed).
    Model: gemini-2.5-flash-image (Nano Banana) - fast, high-quality image generation
    Alternative: gemini-3-pro-image-preview (commented out) - higher quality, slower

    Args:
        flavor: Cake flavor (chocolate, ananas, banana)
        nuts: Nut topping (almond, walnut, or "none")
        people_count: Number of people (6-50)
        concept: Theme/concept (birthday, Star Wars, etc.)
        order_id: Order ID (e.g., CAKE-20260217-A3F2)
        cake_number: Cake number in order (1, 2, etc.)

    Returns:
        GCS path: gs://{bucket}/cakes/orders/{order_id}/cake_{N}.png

    Raises:
        ValueError: If parameters are invalid
        RuntimeError: If image generation or upload fails
    """
    # Validate inputs
    valid_flavors = ["chocolate", "ananas", "banana"]
    valid_nuts = ["almond", "walnut", "none"]

    if flavor not in valid_flavors:
        raise ValueError(f"Invalid flavor: {flavor}. Must be one of {valid_flavors}")

    if nuts not in valid_nuts:
        raise ValueError(f"Invalid nuts: {nuts}. Must be one of {valid_nuts}")

    if not (6 <= people_count <= 50):
        raise ValueError(f"Invalid people_count: {people_count}. Must be 6-50")

    # Get configuration from environment
    project_id = os.getenv("GCP_PROJECT_ID")
    region = os.getenv("GCP_REGION", "europe-west4")
    bucket_name = os.getenv("GCS_BUCKET")

    if not project_id or not bucket_name:
        raise RuntimeError("GCP_PROJECT_ID and GCS_BUCKET environment variables required")

    # Construct prompt for Gemini
    nuts_text = f"with {nuts} decoration" if nuts != "none" else "without nuts"
    prompt = (
        f"Generate an image of a beautiful {flavor} cake for {people_count} people "
        f"{nuts_text}, {concept} theme. Professional bakery photo style, "
        f"high quality, appetizing, realistic, elegant presentation."
    )

    try:
        # Initialize Vertex AI
        vertexai.init(project=project_id, location=region)

        # Load Gemini image generation model
        # Primary: Gemini 2.5 Flash Image (Nano Banana) - fast and efficient
        model = GenerativeModel("gemini-2.5-flash-image")

        # Alternative (higher quality, slower):
        # model = GenerativeModel("gemini-3-pro-image-preview")

        # Generate image
        response = model.generate_content(prompt)

        # Extract image from response
        # Gemini returns base64-encoded image in response
        if not response.candidates or not response.candidates[0].content.parts:
            raise RuntimeError("Gemini returned no image")

        # Get the image data (base64 encoded)
        image_part = response.candidates[0].content.parts[0]
        if not hasattr(image_part, 'inline_data'):
            raise RuntimeError("No image data in response")

        image_bytes = image_part.inline_data.data

        # Upload to GCS
        gcs_path = f"cakes/orders/{order_id}/cake_{cake_number}.png"
        full_gcs_path = f"gs://{bucket_name}/{gcs_path}"

        storage_client = storage.Client(project=project_id)
        bucket = storage_client.bucket(bucket_name)
        blob = bucket.blob(gcs_path)

        blob.upload_from_string(image_bytes, content_type="image/png")

        return full_gcs_path

    except Exception as e:
        raise RuntimeError(f"Failed to generate or upload image: {e}")


def check_ingredient_available(item: str) -> bool:
    """Check if ingredient is available (stub for Phase 3, A2A in Phase 4).

    Args:
        item: Ingredient name (chocolate, ananas, banana, walnut, almond)

    Returns:
        True if available (always True in Phase 3 stub)
    """
    # Phase 3 stub: Always return True
    # Phase 4: This will be replaced with A2A call to Supply Chain Inventory agent
    valid_items = ["chocolate", "ananas", "banana", "walnut", "almond"]

    if item not in valid_items:
        raise ValueError(f"Invalid item: {item}. Must be one of {valid_items}")

    # Stub: return True for all items
    # Real implementation in Phase 4 will call supply_chain via A2A
    return True
