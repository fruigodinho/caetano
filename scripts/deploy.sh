#!/bin/bash
#
# deploy.sh — Sincroniza o binário do caetano para o servidor de produção.
#
# O serviço NÃO é parado nem reiniciado automaticamente. Após o sync, reiniciar
# com: make deploy-restart
#
# Porquê sync sem stop?
#   - systemctl stop via SSH pode causar instabilidade na ligação
#   - O binário é instalado com mv atómico (.new → destino), seguro com o
#     processo em execução (mantém o inode antigo até ser reiniciado)
#
# Diferença face ao padrão gonotesweb: templates e estáticos vão embutidos no
# binário (go:embed em core/web, home, xmldri, saldos-esperados) — não há uma
# pasta web/ a sincronizar à parte. Só se sobrepõe o binário e, se existir, o
# .example de configuração (nunca o caetano.prod.yaml real, que fica intocado
# no servidor).
#
# Variáveis (sobreponíveis via env ou Makefile):
#   DEPLOY_HOST    — host remoto (obrigatório)
#   DEPLOY_PATH    — diretório remoto (ex.: /opt/caetano)
#   DEPLOY_SERVICE — nome do serviço systemd (ex.: caetano)
#   BINARY_NAME    — nome do binário (padrão: caetano)
#   DEPLOY_OWNER   — utilizador/grupo dono dos ficheiros no servidor

set -euo pipefail

DEPLOY_HOST="${DEPLOY_HOST:?Erro: define DEPLOY_HOST}"
DEPLOY_PATH="${DEPLOY_PATH:-/opt/caetano}"
DEPLOY_SERVICE="${DEPLOY_SERVICE:-caetano}"
BINARY_NAME="${BINARY_NAME:-caetano}"
BINARY_PATH="bin/${BINARY_NAME}"
DEPLOY_OWNER="${DEPLOY_OWNER:-caetano-app}"

SSH_TARGET="${DEPLOY_HOST}"
SSH_CTRL="/tmp/caetano-deploy-$$-ctrl"
SSH_OPTS="-o StrictHostKeyChecking=no -o ConnectTimeout=10 -o ControlMaster=auto -o ControlPath=${SSH_CTRL} -o ControlPersist=120"
RSYNC_SSH="ssh -o StrictHostKeyChecking=no -o ControlMaster=auto -o ControlPath=${SSH_CTRL} -o ControlPersist=120"

trap 'ssh -O exit -o ControlPath=${SSH_CTRL} ${SSH_TARGET} 2>/dev/null || true' EXIT

GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

log_info()    { echo -e "  ${GREEN}▸${NC} $1"; }
log_error()   { echo -e "  ${RED}✗${NC}  $1" >&2; }
log_section() { echo -e "\n${BLUE}${BOLD}── $1${NC}"; }
log_ok()      { echo -e "  ${GREEN}✓${NC}  $1"; }

log_section "Verificações pre-deploy"

if [ ! -f "${BINARY_PATH}" ]; then
    log_error "Binário não encontrado: ${BINARY_PATH}"
    log_error "Executa 'make build-prod' antes de fazer deploy."
    exit 1
fi
log_ok "Binário encontrado: ${BINARY_PATH} ($(du -sh "${BINARY_PATH}" | cut -f1))"

if ! command -v rsync >/dev/null 2>&1; then
    log_error "rsync não encontrado."
    exit 1
fi

log_info "A verificar ligação SSH para ${SSH_TARGET}..."
if ! ssh ${SSH_OPTS} "${SSH_TARGET}" "echo ok" >/dev/null 2>&1; then
    log_error "Falha na ligação SSH para ${SSH_TARGET}"
    exit 1
fi
log_ok "Ligação SSH OK"

REMOTE_TMP="caetano-deploy-$$"

log_section "Upload  →  ${SSH_TARGET}:~/${REMOTE_TMP}/"
ssh ${SSH_OPTS} "${SSH_TARGET}" "mkdir -p ~/${REMOTE_TMP}"
rsync -az --progress -e "${RSYNC_SSH}" "${BINARY_PATH}" "${SSH_TARGET}:~/${REMOTE_TMP}/"
log_ok "Binário enviado"

log_section "Instalar  →  ${DEPLOY_PATH}/"
log_info "A instalar binário (mv atómico)..."
if ! ssh -tt ${SSH_OPTS} "${SSH_TARGET}" \
    "sudo cp ${DEPLOY_PATH}/${BINARY_NAME} ${DEPLOY_PATH}/${BINARY_NAME}.prev 2>/dev/null || true; \
     sudo cp ~/${REMOTE_TMP}/${BINARY_NAME} ${DEPLOY_PATH}/${BINARY_NAME}.new && \
     sudo chmod +x ${DEPLOY_PATH}/${BINARY_NAME}.new && \
     sudo mv -f ${DEPLOY_PATH}/${BINARY_NAME}.new ${DEPLOY_PATH}/${BINARY_NAME}"; then
    log_error "Falha ao instalar binário em ${DEPLOY_PATH}/"
    ssh ${SSH_OPTS} "${SSH_TARGET}" "rm -rf ~/${REMOTE_TMP}" || true
    exit 1
fi
log_ok "Binário instalado"

log_info "A definir owner ${DEPLOY_OWNER}:${DEPLOY_OWNER}..."
if ! ssh -tt ${SSH_OPTS} "${SSH_TARGET}" \
    "sudo chown ${DEPLOY_OWNER}:${DEPLOY_OWNER} ${DEPLOY_PATH}/${BINARY_NAME}"; then
    log_error "Falha ao definir owner"
    ssh ${SSH_OPTS} "${SSH_TARGET}" "rm -rf ~/${REMOTE_TMP}" || true
    exit 1
fi
log_ok "Owner: ${DEPLOY_OWNER}:${DEPLOY_OWNER}"

log_info "A limpar temporários..."
ssh ${SSH_OPTS} "${SSH_TARGET}" "rm -rf ~/${REMOTE_TMP}" || true

echo ""
echo -e "${GREEN}${BOLD}  Sync concluído!${NC}"
echo -e "  Host:    ${SSH_TARGET}:${DEPLOY_PATH}"
echo -e "  Serviço: ${DEPLOY_SERVICE} ${YELLOW}(ainda em execução com a versão anterior)${NC}"
echo ""
echo -e "  ${YELLOW}→ Para aplicar as alterações:${NC}"
echo -e "    ${BOLD}make deploy-restart${NC}   — reinicia o serviço"
echo -e "    ${BOLD}make deploy-status${NC}    — ver logs e estado"
echo ""
