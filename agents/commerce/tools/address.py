"""Address validation tools for Amsterdam delivery."""
import re


def validate_amsterdam_address(postcode: str, house_number: str) -> dict[str, any]:
    """Validate Amsterdam address.

    Amsterdam postcodes range from 1000 to 1109.
    Dutch postcode format: 4 digits + space + 2 uppercase letters (e.g., "1013 AP")

    Args:
        postcode: Dutch postcode (e.g., "1013 AP")
        house_number: House number (e.g., "42", "42A")

    Returns:
        {
            "valid": bool,
            "formatted_address": str or None,
            "error": str or None
        }
    """
    # Normalize postcode (remove extra spaces, convert to uppercase)
    postcode = postcode.strip().upper()

    # Validate format: 4 digits + space + 2 letters
    pattern = r"^(\d{4})\s*([A-Z]{2})$"
    match = re.match(pattern, postcode)

    if not match:
        return {
            "valid": False,
            "formatted_address": None,
            "error": "Invalid postcode format. Expected: NNNN XX (e.g., 1013 AP)"
        }

    # Extract parts
    digits, letters = match.groups()
    postcode_num = int(digits)

    # Check Amsterdam range (1000-1109)
    if not (1000 <= postcode_num <= 1109):
        return {
            "valid": False,
            "formatted_address": None,
            "error": "Sorry, Magic Cake only delivers in Amsterdam (postcodes 1000-1109)"
        }

    # Validate house number (basic check)
    if not house_number or not house_number.strip():
        return {
            "valid": False,
            "formatted_address": None,
            "error": "House number is required"
        }

    # Format address
    formatted_postcode = f"{digits} {letters}"
    formatted_address = f"{house_number.strip()}, {formatted_postcode} Amsterdam"

    return {
        "valid": True,
        "formatted_address": formatted_address,
        "error": None
    }


def get_available_delivery_dates() -> list[str]:
    """Get available delivery dates (next 3 days).

    Returns:
        List of dates in YYYY-MM-DD format (tomorrow, day after, day after that)
    """
    from datetime import datetime, timedelta

    today = datetime.now()
    dates = []

    for i in range(1, 4):  # Next 3 days
        date = today + timedelta(days=i)
        dates.append(date.strftime("%Y-%m-%d"))

    return dates
