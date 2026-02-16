FROM python:3.11-slim

WORKDIR /app

# Copy entire agents directory to preserve package structure
COPY agents /app/agents

# Install packages
RUN pip install --no-cache-dir /app/agents/shared /app/agents/supply_chain

# Add /app to PYTHONPATH so "from agents.shared" imports work
ENV PYTHONPATH=/app

EXPOSE 8002

# Run from /app/agents so "supply_chain" package is found
WORKDIR /app/agents
CMD ["python", "-m", "supply_chain.main"]
