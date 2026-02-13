FROM python:3.11-slim

WORKDIR /app

# Copy shared package
COPY agents/shared /app/shared

# Copy commerce package
COPY agents/commerce /app/commerce

# Install both packages
RUN pip install --no-cache-dir /app/shared /app/commerce

EXPOSE 8001

CMD ["python", "-m", "commerce.main"]
