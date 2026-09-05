#!/bin/bash
# Script para alternar entre ambientes de desenvolvimento e produção

set -e

ENV=$1

if [ -z "$ENV" ]; then
    echo "Uso: $0 [dev|prod]"
    echo ""
    echo "Ambientes disponíveis:"
    echo "  dev  - Desenvolvimento (local.rswebportal.com:8080)"
    echo "  prod - Produção (se.rswebportal.com:443)"
    exit 1
fi

case "$ENV" in
    dev)
        if [ -f .env.backup ]; then
            mv .env.backup .env
            echo "✓ Ambiente de desenvolvimento restaurado"
        else
            echo "⚠ Ficheiro .env já está em modo desenvolvimento"
        fi
        echo ""
        echo "Configuração:"
        echo "  - Certificado: local.rswebportal.com"
        echo "  - Porta: 8080"
        echo "  - Comando: make run-web"
        ;;
    prod)
        if [ ! -f .env.production ]; then
            echo "✗ Erro: .env.production não encontrado"
            exit 1
        fi
        cp .env .env.backup
        cp .env.production .env
        echo "✓ Ambiente de produção configurado"
        echo ""
        echo "Configuração:"
        echo "  - Certificado: se.rswebportal.com"
        echo "  - Porta: 443"
        echo "  - Comando: sudo ./bin/server"
        echo ""
        echo "⚠ Não esquecer de compilar: make build-server"
        ;;
    *)
        echo "✗ Ambiente inválido: $ENV"
        echo "Use: dev ou prod"
        exit 1
        ;;
esac
