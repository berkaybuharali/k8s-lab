.PHONY: deploy apply destroy connect seed-redis help

# Extract cloud provider from command line (e.g., "make deploy gcp" -> "gcp")
CLOUD := $(filter-out deploy apply destroy connect seed-redis help,$(MAKECMDGOALS))

deploy:
	@./scripts/deploy.sh $(CLOUD)

apply:
	@./scripts/apply.sh $(CLOUD)

destroy:
	@./scripts/destroy.sh $(CLOUD)

connect:
	@./scripts/connect.sh $(CLOUD)

seed-redis:
	@./scripts/seed-redis.sh $(CLOUD)

help:
	@echo "Kubernetes Lab"
	@echo ""
	@echo "Usage: make <command> <cloud>"
	@echo ""
	@echo "Commands:"
	@echo "  deploy <cloud>      Create infrastructure and bootstrap Kubernetes"
	@echo "  apply <cloud>       Deploy applications (NGINX, Redis)"
	@echo "  seed-redis <cloud>  Seed Redis with test data (for Velero testing)"
	@echo "  connect <cloud>     Show tunnel command for local access"
	@echo "  destroy <cloud>     Destroy all resources"
	@echo ""
	@echo "Supported: gcp"
	@echo ""
	@echo "Example workflow:"
	@echo "  make deploy gcp      # Create cluster"
	@echo "  make apply gcp       # Deploy apps"
	@echo "  make seed-redis gcp  # Add test data"
	@echo "  make connect gcp     # Get tunnel command"
	@echo "  make destroy gcp     # Tear down"

%:
	@:
