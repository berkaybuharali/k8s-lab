.PHONY: deploy down

deploy:
	@echo "Deploying cluster..."
	@./scripts/setup.sh

down:
	@echo "Destroying cluster..."
	@./scripts/destroy.sh
