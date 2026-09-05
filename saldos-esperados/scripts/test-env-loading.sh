#!/bin/bash
# Script para testar carregamento de variáveis de ambiente
# Simula comportamento do systemd

set -e

echo "🧪 Testando carregamento de variáveis de ambiente..."
echo ""

# Carregar variáveis de systemd.env (simulando systemd)
if [ ! -f systemd.env ]; then
    echo "❌ systemd.env não encontrado. Execute: make generate-systemd-env"
    exit 1
fi

echo "📋 Carregando variáveis de systemd.env..."
# Extrair apenas os valores (remover Environment=")
while IFS= read -r line; do
    if [[ $line =~ ^Environment=\"(.*)\"$ ]]; then
        export "${BASH_REMATCH[1]}"
        echo "  ✓ ${BASH_REMATCH[1]%%=*}"
    fi
done < systemd.env

echo ""
echo "🔍 Verificando variáveis críticas..."
critical_vars=("DB_HOST" "DB_USER" "DB_NAME" "SESSION_SECRET" "ENCRYPTION_KEY" "GIN_MODE")
all_ok=true

for var in "${critical_vars[@]}"; do
    if [ -z "${!var}" ]; then
        echo "  ❌ $var não definida"
        all_ok=false
    else
        # Não mostrar valores sensíveis completos
        if [[ "$var" == *"SECRET"* ]] || [[ "$var" == *"KEY"* ]] || [[ "$var" == *"PASSWORD"* ]]; then
            echo "  ✓ $var=****** (${#!var} chars)"
        else
            echo "  ✓ $var=${!var}"
        fi
    fi
done

echo ""
if [ "$all_ok" = true ]; then
    echo "✅ Todas as variáveis críticas estão definidas"
    echo ""
    echo "🚀 Iniciando servidor com variáveis de ambiente..."
    echo "   (Pressione Ctrl+C para parar)"
    echo ""
    
    # Compilar e executar
    if [ ! -f bin/server ]; then
        echo "📦 Compilando servidor..."
        make build-server >/dev/null 2>&1
    fi
    
    # Executar servidor com variáveis carregadas
    ./bin/server
else
    echo "❌ Algumas variáveis críticas estão em falta"
    exit 1
fi
