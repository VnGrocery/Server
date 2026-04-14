SHELL := /bin/bash

COMPOSE ?= docker compose
BASE_COMPOSE := -f docker-compose.yml
VAULT_PERSISTENT_COMPOSE := -f docker-compose.vault-persistent.yml
BESU_COMPOSE := -f docker-compose.besu-qbft.yml
STAGING_COMPOSE := -f docker-compose.staging.yml
PROD_COMPOSE := -f docker-compose.prod.yml
DEPLOY_COMPOSE := -f docker-compose.deploy.yml
VAULT_HTTP_ADDR ?= http://127.0.0.1:8200
service ?= api

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make api-up              # start api + vault(dev) + ipfs + redis"
	@echo "  make vault-up            # start persistent vault only"
	@echo "  make ipfs-up             # start ipfs only"
	@echo "  make redis-up            # start redis only"
	@echo "  make besu-up             # start 4-node Besu QBFT only"
	@echo "  make stack-up            # start base + persistent vault + besu + staging proxy"
	@echo "  make deploy-up           # start single-file deploy baseline"
	@echo "  make build-api           # build api image in deploy compose"
	@echo "  make run-all             # build api image, then run deploy baseline with vault readiness checks"
	@echo "  make vault-init          # print/init persistent vault manually"
	@echo "  make vault-status        # show persistent vault status"
	@echo "  make deploy-config       # validate deploy compose"
	@echo "  make logs service=api    # tail logs for a service in deploy compose"
	@echo "  make ps                  # show deploy stack containers"
	@echo "  make down                # stop deploy stack"
	@echo "  make clean               # stop deploy stack and remove local temp artifacts"
	@echo "  make clean-local         # stop deploy stack and remove orphan containers"
	@echo "  make clean-all           # stop deploy stack and remove local volumes"

.PHONY: api-up
api-up:
	$(COMPOSE) $(BASE_COMPOSE) up -d --build

.PHONY: vault-up
vault-up:
	$(COMPOSE) $(DEPLOY_COMPOSE) up -d vault

.PHONY: ipfs-up
ipfs-up:
	$(COMPOSE) $(DEPLOY_COMPOSE) up -d ipfs

.PHONY: redis-up
redis-up:
	$(COMPOSE) $(DEPLOY_COMPOSE) up -d redis

.PHONY: besu-up
besu-up:
	$(COMPOSE) $(BESU_COMPOSE) up -d

.PHONY: stack-up
stack-up:
	$(COMPOSE) $(BASE_COMPOSE) $(VAULT_PERSISTENT_COMPOSE) $(BESU_COMPOSE) $(STAGING_COMPOSE) up -d --build

.PHONY: deploy-up
deploy-up:
	./scripts/up-deploy.sh

.PHONY: build-api
build-api:
	$(COMPOSE) $(DEPLOY_COMPOSE) build api

.PHONY: run-all
run-all: build-api
	./scripts/run-all.sh

.PHONY: vault-init
vault-init:
	./scripts/init-vault.sh

.PHONY: vault-status
vault-status:
	$(COMPOSE) $(DEPLOY_COMPOSE) exec vault vault status -address=$(VAULT_HTTP_ADDR)

.PHONY: deploy-config
deploy-config:
	$(COMPOSE) $(DEPLOY_COMPOSE) config

.PHONY: logs
logs:
	$(COMPOSE) $(DEPLOY_COMPOSE) logs -f $(service)

.PHONY: ps
ps:
	$(COMPOSE) $(DEPLOY_COMPOSE) ps

.PHONY: down
down:
	$(COMPOSE) $(DEPLOY_COMPOSE) down

.PHONY: clean
clean:
	$(COMPOSE) $(DEPLOY_COMPOSE) down --remove-orphans
	rm -f /tmp/vngrocery-deploy-compose*.txt /tmp/vngrocery-vault-status*.txt /tmp/vngrocery-vault-runall*.txt

.PHONY: clean-local
clean-local:
	$(COMPOSE) $(DEPLOY_COMPOSE) down --remove-orphans

.PHONY: clean-all
clean-all:
	$(COMPOSE) $(DEPLOY_COMPOSE) down --remove-orphans --volumes
	rm -f /tmp/vngrocery-deploy-compose*.txt /tmp/vngrocery-vault-status*.txt /tmp/vngrocery-vault-runall*.txt
