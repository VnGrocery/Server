# Thin wrapper over ./scripts/vng, which is the real entry point.
# Run `./scripts/vng help` (or `make help`) for the full command list.
SHELL := /bin/bash
VNG := ./scripts/vng
service ?= api

.PHONY: help up up-prod up-qbft down restart status logs health reset contract vault-init e2e test
help:
	@$(VNG) help

up:              ## dev stack, single-node Besu
	$(VNG) up

up-prod:         ## production overlay
	$(VNG) up --prod

up-qbft:         ## 4-validator cluster (demo)
	$(VNG) up --qbft

down:
	$(VNG) down

restart:
	$(VNG) restart

status:
	$(VNG) status

logs:            ## make logs service=api
	$(VNG) logs $(service)

health:
	$(VNG) health

reset:           ## wipe containers + volumes (asks for confirmation)
	$(VNG) reset

contract:        ## deploy IntegrityRegistry and update .env
	$(VNG) contract-deploy

vault-init:
	$(VNG) --prod vault-init

e2e:
	$(VNG) e2e

test:
	go test ./...
