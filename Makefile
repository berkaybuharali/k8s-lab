.PHONY: deploy-infra deploy-tools deploy-applications deploy destroy connect seed-redis backup restore list-backups delete-backup delete-all-backups help build-ui

# Extract cloud provider from command line (e.g., "make deploy-infra gcp" -> "gcp")
CLOUD := $(filter-out deploy-infra deploy-tools deploy-applications deploy destroy connect seed-redis backup restore list-backups delete-backup delete-all-backups help,$(MAKECMDGOALS))

deploy-infra:
	@./scripts/infra/deploy.sh $(CLOUD)

deploy-tools:
	@./scripts/platform/deploy.sh $(CLOUD)

deploy-applications:
	@./scripts/workloads/deploy.sh $(CLOUD)

# Convenience target: deploy infra + tools + applications
deploy: deploy-infra deploy-tools deploy-applications

destroy:
	@./scripts/infra/destroy.sh $(CLOUD)

connect:
	@./scripts/infra/connect.sh $(CLOUD)

seed-redis:
	@./scripts/workloads/seed-redis.sh $(CLOUD)

backup:
	@NAME=$(NAME) NAMESPACES=$(NAMESPACES) ./scripts/backup/create.sh $(CLOUD)

restore:
	@./scripts/backup/restore.sh $(CLOUD)

list-backups:
	@./scripts/backup/list.sh $(CLOUD)

delete-backup:
	@./scripts/backup/delete.sh $(CLOUD) $(NAME)

delete-all-backups:
	@./scripts/backup/delete-all.sh $(CLOUD)

build-ui:
	@cd ui/frontend && npm run build
	@rm -rf cli/pkg/ui/dist && mkdir -p cli/pkg/ui/dist
	@cp -R ui/frontend/dist/* cli/pkg/ui/dist/
	@cd cli && go build -o ../bin/k8s-lab .

help:
	@echo "Kubernetes Lab"
	@echo ""
	@echo "Usage: make <command> <cloud>"
	@echo ""
	@echo "Commands:"
	@echo "  deploy-infra <cloud>        Create infrastructure and bootstrap Kubernetes"
	@echo "  deploy-tools <cloud>        Deploy cluster tools (CSI, StorageClass, Velero)"
	@echo "  deploy-applications <cloud> Deploy applications (NGINX, Redis)"
	@echo "  deploy <cloud>              All-in-one: infra + tools + applications"
	@echo "  seed-redis <cloud>          Seed Redis with test data (for backup testing)"
	@echo "  backup <cloud>              Backup applications (adds timestamp to name)"
	@echo "                              Optional: NAME=<base> NAMESPACES=<ns1,ns2>"
	@echo "  list-backups <cloud>        List all Velero backups"
	@echo "  delete-backup <cloud> NAME= Delete a Velero backup by name"
	@echo "  delete-all-backups <cloud>  Delete all Velero backups"
	@echo "  restore <cloud>             Restore from latest backup (includes tools setup)"
	@echo "  connect <cloud>             Show tunnel command for local access"
	@echo "  destroy <cloud>             Destroy all resources"
	@echo "  build-ui                    Build frontend and Go binary"
	@echo ""
	@echo "Supported: gcp"
	@echo ""
	@echo "Day 1 workflow (fresh deployment):"
	@echo "  make deploy-infra gcp        # Create cluster"
	@echo "  make deploy-tools gcp        # CSI + Velero"
	@echo "  make deploy-applications gcp # NGINX + Redis"
	@echo "  make seed-redis gcp          # Add test data"
	@echo "  make backup gcp              # Backup to GCS"
	@echo "  make destroy gcp             # Tear down"
	@echo ""
	@echo "Quick workflow (all-in-one):"
	@echo "  make deploy gcp              # Infra + tools + apps"
	@echo "  make seed-redis gcp"
	@echo "  make backup gcp"
	@echo "  make destroy gcp"
	@echo ""
	@echo "Custom backup examples:"
	@echo "  NAME=prod-backup make backup gcp"
	@echo "  NAMESPACES=app1,app2 make backup gcp"
	@echo "  NAME=multi NAMESPACES=ns1,ns2,ns3 make backup gcp"
	@echo ""
	@echo "Day 2+ workflow (restore from backup):"
	@echo "  make deploy-infra gcp        # Create cluster"
	@echo "  make restore gcp             # Tools + apps from backup"
	@echo "  make destroy gcp             # Tear down"

%:
	@:
