"""Image generation tool using Gemini 2.5 Flash Image via google-genai SDK."""
import os

from google import genai
from google.genai import types
from google.cloud import storage

# Module-level initialisation — env vars are read once at import time.
# The GenAI client picks up GOOGLE_API_KEY automatically from the environment.
_project_id: str = os.getenv("GCP_PROJECT_ID", "")
_bucket_name: str = os.getenv("GCS_BUCKET", "")
_genai_client: genai.Client = genai.Client()


def generate_cake_image(
    flavor: str,
    nuts: str,
    people_count: int,
    concept: str,
    order_id: str,
    cake_number: int = 1,
) -> str:
    """Generate cake image using Gemini 2.5 Flash Image and upload to GCS.

    Args:
        flavor: Cake flavor (chocolate, ananas, banana)
        nuts: Nut topping (almond, walnut, or "none")
        people_count: Number of people (6-50)
        concept: Theme/concept (birthday, Star Wars, etc.)
        order_id: Order ID (e.g., CAKE-20260217-A3F2)
        cake_number: Cake number in order (1, 2, etc.)

    Returns:
        GCS path: gs://{bucket}/cakes/orders/{order_id}/cake_{N}.png
    """
    valid_flavors = ["chocolate", "ananas", "banana"]
    valid_nuts = ["almond", "walnut", "none"]

    if flavor not in valid_flavors:
        raise ValueError(f"Invalid flavor: {flavor}. Must be one of {valid_flavors}")
    if nuts not in valid_nuts:
        raise ValueError(f"Invalid nuts: {nuts}. Must be one of {valid_nuts}")
    if not (6 <= people_count <= 50):
        raise ValueError(f"Invalid people_count: {people_count}. Must be 6-50")

    if not _project_id or not _bucket_name:
        raise RuntimeError("GCP_PROJECT_ID and GCS_BUCKET must be set in the environment")

    nuts_text = f"with {nuts} decoration" if nuts != "none" else "without nuts"
    prompt = (
        f"Generate an image of a beautiful {flavor} cake for {people_count} people "
        f"{nuts_text}, {concept} theme. Professional bakery photo style, "
        f"high quality, appetizing, realistic, elegant presentation."
    )

    try:
        response = _genai_client.models.generate_content(
            model="gemini-2.5-flash-image",
            contents=prompt,
            config=types.GenerateContentConfig(
                response_modalities=["IMAGE"],
            ),
        )

        # Extract image bytes via the typed SDK path
        image_bytes = None
        for part in response.candidates[0].content.parts:
            if part.inline_data is not None:
                image_bytes = part.inline_data.data
                break

        if image_bytes is None:
            raise RuntimeError("Gemini returned no image in response")

        # Upload to GCS
        gcs_path = f"cakes/orders/{order_id}/cake_{cake_number}.png"
        full_gcs_path = f"gs://{_bucket_name}/{gcs_path}"

        storage_client = storage.Client(project=_project_id)
        bucket = storage_client.bucket(_bucket_name)
        blob = bucket.blob(gcs_path)
        blob.upload_from_string(image_bytes, content_type="image/png")

        return full_gcs_path

    except Exception as e:
        raise RuntimeError(f"Failed to generate or upload image: {e}")
