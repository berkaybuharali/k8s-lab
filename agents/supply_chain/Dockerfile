FROM python:3.11-slim

WORKDIR /app

# Copy shared package
COPY agents/shared /app/shared

# Copy supply_chain package
COPY agents/supply_chain /app/supply_chain

# Install both packages
RUN pip install --no-cache-dir /app/shared /app/supply_chain

EXPOSE 8002

CMD ["python", "-m", "supply_chain.main"]
