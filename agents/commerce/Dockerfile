FROM python:3.11-slim

WORKDIR /app

# Layer 1: Install dependencies (rarely changes - cached!)
# Copy only dependency files first
COPY agents/shared/pyproject.toml /app/agents/shared/
COPY agents/commerce/pyproject.toml /app/agents/commerce/

# Create minimal __init__.py for pip install to work
RUN mkdir -p /app/agents/shared /app/agents/commerce && \
    touch /app/agents/shared/__init__.py /app/agents/commerce/__init__.py

# Install dependencies - this layer is cached unless pyproject.toml changes
RUN pip install --no-cache-dir /app/agents/shared /app/agents/commerce

# Layer 2: Copy application code (changes frequently)
COPY agents/shared /app/agents/shared
COPY agents/commerce /app/agents/commerce

# Add /app to PYTHONPATH so "from agents.shared" imports work
ENV PYTHONPATH=/app

EXPOSE 8001

# Run from /app/agents so "commerce" package is found
WORKDIR /app/agents
CMD ["uvicorn", "commerce.main:app", "--host", "0.0.0.0", "--port", "8001"]
