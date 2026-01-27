.PHONY: deploy apply destroy connect seed-redis backup restore list-backups delete-backup delete-all-backups help

# Extract cloud provider from command line (e.g., "make deploy gcp" -> "gcp")
CLOUD := $(filter-out deploy apply destroy connect seed-redis backup restore list-backups delete-backup delete-all-backups help,$(MAKECMDGOALS))

deploy:
	@./scripts/cluster/deploy.sh $(CLOUD)

apply:
	@./scripts/apps/apply.sh $(CLOUD)

destroy:
	@./scripts/cluster/destroy.sh $(CLOUD)

connect:
	@./scripts/cluster/connect.sh $(CLOUD)

seed-redis:
	@./scripts/apps/seed-redis.sh $(CLOUD)

backup:
	@./scripts/velero/backup.sh $(CLOUD)

restore:
	@./scripts/velero/restore.sh $(CLOUD)

list-backups:
	@./scripts/velero/list-backups.sh $(CLOUD)

delete-backup:
	@./scripts/velero/delete-backup.sh $(CLOUD) $(NAME)

delete-all-backups:
	@./scripts/velero/delete-all-backups.sh $(CLOUD)

help:
	@echo "Kubernetes Lab"
	@echo ""
	@echo "Usage: make <command> <cloud>"
	@echo ""
	@echo "Commands:"
	@echo "  deploy <cloud>              Create infrastructure and bootstrap Kubernetes"
	@echo "  apply <cloud>               Deploy applications (NGINX, Redis, Velero)"
	@echo "  seed-redis <cloud>          Seed Redis with test data (for Velero testing)"
	@echo "  backup <cloud>              Backup applications to cloud storage (Velero)"
	@echo "  list-backups <cloud>        List all Velero backups"
	@echo "  delete-backup <cloud> NAME= Delete a Velero backup by name"
	@echo "  delete-all-backups <cloud>  Delete all Velero backups"
	@echo "  restore <cloud>             Restore applications from backup (Velero)"
	@echo "  connect <cloud>             Show tunnel command for local access"
	@echo "  destroy <cloud>             Destroy all resources"
	@echo ""
	@echo "Supported: gcp"
	@echo ""
	@echo "Day 1 workflow:"
	@echo "  make deploy gcp      # Create cluster"
	@echo "  make apply gcp       # Deploy apps + Velero"
	@echo "  make seed-redis gcp  # Add test data"
	@echo "  make backup gcp      # Backup to GCS"
	@echo "  make destroy gcp     # Tear down"
	@echo ""
	@echo "Day 2+ workflow (restore):"
	@echo "  make deploy gcp      # Create cluster"
	@echo "  make restore gcp     # Restore apps from backup"
	@echo "  make destroy gcp     # Tear down"

%:
	@:
