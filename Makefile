.PHONY: up down

up:
	@echo "Setting up cluster..."
	@./scripts/setup.sh

down:
	@echo "Destroying cluster..."
	@./scripts/destroy.sh
