"""Google Cloud Storage tools for cake images."""
import logging
import datetime
from typing import List, Optional
from google.cloud import storage

try:
    from agents.shared.config import config
except ImportError:
    from ...shared.config import config

logger = logging.getLogger(__name__)

# Constants
CAKES_PREFIX = "cakes/orders/"

def _get_client() -> storage.Client:
    """Get GCS client."""
    return storage.Client(project=config.GCP_PROJECT_ID)

def upload_cake_image(order_id: str, cake_number: int, image_bytes: bytes) -> str:
    """
    Upload a cake image to GCS.
    
    Args:
        order_id: The order ID.
        cake_number: The index of the cake in the order (1-based).
        image_bytes: The raw image bytes.
        
    Returns:
        str: The public URL or authenticated URL of the uploaded image.
    """
    client = _get_client()
    bucket = client.bucket(config.GCS_BUCKET)
    
    blob_name = f"{CAKES_PREFIX}{order_id}/cake_{cake_number}.png"
    blob = bucket.blob(blob_name)
    
    try:
        blob.upload_from_string(image_bytes, content_type="image/png")
        logger.info(f"Uploaded cake image to gs://{config.GCS_BUCKET}/{blob_name}")
        # Return a signed URL or public URL depending on requirements.
        # For simplicity in this lab, let's assume signed URLs for temporary access
        # or public if bucket allows. The plan mentions "get_cake_image_urls" returns signed URLs.
        # Here we return the GCS path for internal reference.
        return f"gs://{config.GCS_BUCKET}/{blob_name}"
    except Exception as e:
        logger.error(f"Error uploading image for order {order_id}: {e}")
        raise

def get_cake_image_urls(order_id: str, expiration_minutes: int = 60) -> List[str]:
    """
    Get signed URLs for all images associated with an order.
    
    Args:
        order_id: The order ID.
        expiration_minutes: URL expiration time.
        
    Returns:
        List[str]: List of signed URLs.
    """
    client = _get_client()
    bucket = client.bucket(config.GCS_BUCKET)
    prefix = f"{CAKES_PREFIX}{order_id}/"
    
    urls = []
    blobs = bucket.list_blobs(prefix=prefix)
    
    for blob in blobs:
        if blob.name.endswith(".png"):
            url = blob.generate_signed_url(
                version="v4",
                expiration=datetime.timedelta(minutes=expiration_minutes),
                method="GET"
            )
            urls.append(url)
            
    return urls

def delete_cake_images(order_id: str) -> bool:
    """
    Delete all images for an order.
    
    Args:
        order_id: The order ID.
        
    Returns:
        bool: True if successful.
    """
    client = _get_client()
    bucket = client.bucket(config.GCS_BUCKET)
    prefix = f"{CAKES_PREFIX}{order_id}/"
    
    blobs = list(bucket.list_blobs(prefix=prefix))
    if not blobs:
        return True # Nothing to delete
        
    try:
        bucket.delete_blobs(blobs)
        logger.info(f"Deleted {len(blobs)} images for order {order_id}")
        return True
    except Exception as e:
        logger.error(f"Error deleting images for order {order_id}: {e}")
        return False

def list_orphan_images(known_order_ids: List[str]) -> List[str]:
    """
    List images that do not belong to any known order.
    
    Args:
        known_order_ids: List of valid order IDs from Redis.
        
    Returns:
        List[str]: List of GCS paths (gs://...) for orphan images.
    """
    client = _get_client()
    bucket = client.bucket(config.GCS_BUCKET)
    
    orphans = []
    # List all blobs under cakes/orders/
    blobs = bucket.list_blobs(prefix=CAKES_PREFIX)
    
    for blob in blobs:
        # Expected format: cakes/orders/{order_id}/cake_{N}.png
        parts = blob.name.replace(CAKES_PREFIX, "").split("/")
        if len(parts) < 2:
            continue # Unexpected structure
            
        order_id = parts[0]
        if order_id not in known_order_ids:
            orphans.append(f"gs://{config.GCS_BUCKET}/{blob.name}")
            
    return orphans
