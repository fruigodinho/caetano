# caetano — Makefile

SHELL := /bin/bash

.PHONY: help build build-prod run-dev run test test-coverage clean fmt vet \
        db-migrate-up db-migrate-down db-ping db-dump db-restore backup-decrypt \
        tunnel-start tunnel-stop tunnel-status tunnel-restart \
        tunnel-start-prod tunnel-stop-prod tunnel-status-prod \
        import-legacy \
        deploy deploy-restart deploy-status deploy-logs

BINARY_NAME=caetano
BINARY_PATH=bin/$(BINARY_NAME)

BUILD_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || date -u +%s)
LDFLAGS_PROD = -w -s -X main.BuildVersion=$(BUILD_HASH)

# Variáveis de deploy (sobrepor via env ou linha de comando: make deploy DEPLOY_HOST=...)
DEPLOY_HOST    ?=
DEPLOY_PATH    ?= /opt/caetano
DEPLOY_SERVICE ?= caetano
HEALTH_URL     ?= https://se.rswebportal.com/ping
DEPLOY_OWNER   ?= caetano-app

# Túnel SSH para a BD — aliases definidos no ~/.ssh/config do operador.
# Ver caetano.dev.yaml.example / caetano.prod.yaml.example para o mapeamento.
DB_TUNNEL_HOST      ?= postgres
DB_TUNNEL_HOST_PROD ?= postgres-pgvector
DB_TUNNEL_PORT_PROD ?= 5433

GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
NC=\033[0m

help: ## Mostra esta mensagem de ajuda
	@echo "Comandos disponíveis:"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-24s$(NC) %s\n", $$1, $$2}'

# ============================================================================
# Build / execução
# ============================================================================

build: ## Compila o binário único de produção (home)
	@mkdir -p bin
	@go build -o $(BINARY_PATH) ./home

build-prod: ## Compila o binário otimizado para produção
	@echo -e "$(GREEN)A compilar [$(BUILD_HASH)]...$(NC)"
	@mkdir -p bin
	@go build -ldflags="$(LDFLAGS_PROD)" -trimpath -o $(BINARY_PATH) ./home

run-dev: ## Corre em modo dev (GIN_MODE=debug, go run)
	@GIN_MODE=debug go run ./home

run: build-prod ## Compila e corre o binário de produção localmente
	@./$(BINARY_PATH)

clean: ## Remove artefactos de build
	@rm -rf bin/
	@go clean ./...

# ============================================================================
# Testes / qualidade
# ============================================================================

test: ## Corre todos os testes do workspace
	@go test ./core/... ./home/... ./xmldri/... ./saldos-esperados/...

test-coverage: ## Corre testes com cobertura
	@go test -coverprofile=coverage.out ./core/... ./home/... ./xmldri/... ./saldos-esperados/...
	@go tool cover -func=coverage.out

vet: ## go vet em todo o workspace
	@go vet ./core/... ./home/... ./xmldri/... ./saldos-esperados/...

fmt: ## Formata todo o código
	@go fmt ./core/... ./home/... ./xmldri/... ./saldos-esperados/...

# ============================================================================
# Base de dados
# ============================================================================

db-migrate-up: ## Aplica as migrações do schema caetano
	@go run ./core/db/cmd/migrate -dir=up

db-migrate-down: ## ⚠️ Reverte a última migração do schema caetano
	@go run ./core/db/cmd/migrate -dir=down

db-dump: SHELL := /bin/bash
db-dump: ## Dump integral da BD para pasta (ex: make db-dump BACKUP_DIR=/tmp/backups)
	@if [ -z "$(BACKUP_DIR)" ]; then echo -e "$(RED)Erro: especifica BACKUP_DIR=<pasta>$(NC)"; exit 1; fi
	@set -o pipefail; \
	URL=$$(grep -A2 '^db:' caetano.dev.yaml 2>/dev/null | grep 'dsn:' | sed -E 's/.*dsn:\s*"?([^"]*)"?/\1/'); \
	if [ -z "$$URL" ]; then echo -e "$(RED)Erro: não encontrei db.dsn em caetano.dev.yaml$(NC)"; exit 1; fi; \
	mkdir -p "$(BACKUP_DIR)"; \
	FILE="$(BACKUP_DIR)/caetano_$$(date +%Y%m%d_%H%M%S).sql.gz"; \
	echo -e "$(GREEN)A criar dump integral em $$FILE...$(NC)"; \
	pg_dump "$$URL" --no-owner --no-privileges --clean --if-exists | gzip > "$$FILE" \
		&& echo -e "$(GREEN)✓ Dump concluído: $$FILE$(NC)" \
		|| { rm -f "$$FILE"; echo -e "$(RED)Erro no dump$(NC)"; exit 1; }

backup-decrypt: ## Desencripta um backup .enc (ex: make backup-decrypt BACKUP_FILE=x.sql.gz.enc [OUT=x.sql.gz])
	@if [ -z "$(BACKUP_FILE)" ]; then echo "$(RED)Erro: especifica BACKUP_FILE=<ficheiro.enc>$(NC)"; exit 1; fi
	@OUT_FILE="$(OUT)"; [ -n "$$OUT_FILE" ] || OUT_FILE="$(basename $(BACKUP_FILE) .enc)"; \
	go run ./core/cmd/decrypt -file "$(BACKUP_FILE)" -outfile "$$OUT_FILE"

import-legacy: ## Importa dados de negócio da BD legada de saldos-esperados (RF-11, pontual, só leitura na origem)
	@echo -e "$(YELLOW)Confirma que o túnel para a origem está ativo: make tunnel-start-prod$(NC)"
	@go run ./core/cmd/import-legacy

# ============================================================================
# Túnel SSH (ver caetano.{dev,prod}.yaml.example para os aliases/hosts reais)
# ============================================================================

tunnel-start: ## Inicia túnel SSH para a BD de dev (porta local 5432)
	@./scripts/db-tunnel.sh start postgres $(DB_TUNNEL_HOST) 5432

tunnel-stop: ## Para o túnel SSH da BD de dev
	@./scripts/db-tunnel.sh stop postgres $(DB_TUNNEL_HOST) 5432

tunnel-status: ## Mostra o estado do túnel SSH da BD de dev
	@./scripts/db-tunnel.sh status postgres $(DB_TUNNEL_HOST) 5432

tunnel-restart: tunnel-stop tunnel-start ## Reinicia o túnel SSH da BD de dev

tunnel-start-prod: ## Inicia túnel SSH para a BD de produção (porta local $(DB_TUNNEL_PORT_PROD), só leitura prevista em RF-11)
	@./scripts/db-tunnel.sh start postgres $(DB_TUNNEL_HOST_PROD) $(DB_TUNNEL_PORT_PROD)

tunnel-stop-prod: ## Para o túnel SSH da BD de produção
	@./scripts/db-tunnel.sh stop postgres $(DB_TUNNEL_HOST_PROD) $(DB_TUNNEL_PORT_PROD)

tunnel-status-prod: ## Mostra o estado do túnel SSH da BD de produção
	@./scripts/db-tunnel.sh status postgres $(DB_TUNNEL_HOST_PROD) $(DB_TUNNEL_PORT_PROD)

# ============================================================================
# Deploy
# ============================================================================

export DEPLOY_HOST DEPLOY_PATH DEPLOY_SERVICE BINARY_NAME DEPLOY_OWNER

deploy: build-prod ## Sync para produção: build + upload + instalação atómica do binário (sem parar o serviço)
	@if [ -z "$(DEPLOY_HOST)" ]; then echo -e "$(RED)Erro: define DEPLOY_HOST=<host> (make deploy DEPLOY_HOST=...)$(NC)"; exit 1; fi
	@./scripts/deploy.sh

deploy-restart: ## Reinicia o serviço remoto e verifica disponibilidade via health check
	@if [ -z "$(DEPLOY_HOST)" ]; then echo -e "$(RED)Erro: define DEPLOY_HOST=<host>$(NC)"; exit 1; fi
	@echo -e "$(YELLOW)A reiniciar $(DEPLOY_SERVICE) em $(DEPLOY_HOST)...$(NC)"
	@ssh -t -o StrictHostKeyChecking=no $(DEPLOY_HOST) "sudo systemctl restart $(DEPLOY_SERVICE)"
	@echo -e "$(GREEN)Serviço reiniciado. A verificar disponibilidade...$(NC)"
	@for i in $$(seq 1 15); do \
		STATUS=$$(curl -s -o /dev/null -w '%{http_code}' --max-time 4 $(HEALTH_URL) 2>/dev/null || echo 000); \
		if [ "$$STATUS" = "200" ]; then \
			echo -e "$(GREEN)Aplicação disponível após $${i}s — HTTP $$STATUS$(NC)"; \
			break; \
		fi; \
		sleep 1; \
		if [ "$$i" = "15" ]; then \
			echo -e "$(YELLOW)Health check falhou após 15s — HTTP $$STATUS. Verifica: make deploy-status$(NC)"; \
		fi; \
	done

deploy-status: ## Estado e últimos logs do serviço remoto
	@if [ -z "$(DEPLOY_HOST)" ]; then echo -e "$(RED)Erro: define DEPLOY_HOST=<host>$(NC)"; exit 1; fi
	@ssh -t -o StrictHostKeyChecking=no $(DEPLOY_HOST) \
		"sudo systemctl status $(DEPLOY_SERVICE) --no-pager -l; echo; sudo journalctl -u $(DEPLOY_SERVICE) -n 20 --no-pager"

deploy-logs: ## Segue os logs em tempo real do serviço remoto
	@if [ -z "$(DEPLOY_HOST)" ]; then echo -e "$(RED)Erro: define DEPLOY_HOST=<host>$(NC)"; exit 1; fi
	@ssh -t -o StrictHostKeyChecking=no $(DEPLOY_HOST) "sudo journalctl -u $(DEPLOY_SERVICE) -f"
