FROM python:3.11-slim

WORKDIR /app

# Layer 1: Install dependencies (rarely changes - cached!)
# Copy only dependency files first
COPY agents/shared/pyproject.toml /app/agents/shared/
COPY agents/supply_chain/pyproject.toml /app/agents/supply_chain/

# Create minimal __init__.py for pip install to work
RUN mkdir -p /app/agents/shared /app/agents/supply_chain && \
    touch /app/agents/shared/__init__.py /app/agents/supply_chain/__init__.py

# Install dependencies - this layer is cached unless pyproject.toml changes
RUN pip install --no-cache-dir /app/agents/shared /app/agents/supply_chain

# Layer 2: Copy application code (changes frequently)
COPY agents/shared /app/agents/shared
COPY agents/supply_chain /app/agents/supply_chain

# Add /app to PYTHONPATH so "from agents.shared" imports work
ENV PYTHONPATH=/app

EXPOSE 8002

# Run from /app/agents so "supply_chain" package is found
WORKDIR /app/agents
CMD ["uvicorn", "supply_chain.main:app", "--host", "0.0.0.0", "--port", "8002"]
