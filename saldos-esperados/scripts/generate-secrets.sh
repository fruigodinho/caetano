#!/bin/bash
# Script para gerar secrets seguros para as variáveis de ambiente
#
# Uso:
#   ./scripts/generate-secrets.sh
#   ou
#   bash scripts/generate-secrets.sh (da raiz do projeto)
#
# Este script gera valores seguros para:
# - SESSION_SECRET (32+ caracteres)
# - CSRF_AUTH_KEY (exatamente 32 caracteres)
# - ENCRYPTION_KEY (exatamente 32 caracteres)

set -e

# Detectar diretório raiz do projeto
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Mudar para o diretório raiz
cd "$PROJECT_ROOT"

echo "🔐 Gerador de Secrets Seguros - Saldos Esperados"
echo "=================================================="
echo ""

# Verificar se openssl está disponível
if ! command -v openssl &> /dev/null; then
    echo "❌ ERRO: openssl não está instalado."
    echo "   Instale com: sudo apt-get install openssl"
    exit 1
fi

echo "Gerando secrets aleatórios..."
echo ""

# Gerar SESSION_SECRET (44 chars base64 = ~32 bytes)
SESSION_SECRET=$(openssl rand -base64 32)
echo "✅ SESSION_SECRET gerado (44 caracteres):"
echo "   $SESSION_SECRET"
echo ""

# Gerar CSRF_AUTH_KEY (exatamente 32 bytes = 32 caracteres hex)
CSRF_AUTH_KEY=$(openssl rand -hex 16)
echo "✅ CSRF_AUTH_KEY gerado (32 caracteres):"
echo "   $CSRF_AUTH_KEY"
echo ""

# Gerar ENCRYPTION_KEY (exatamente 32 bytes = 32 caracteres hex)
ENCRYPTION_KEY=$(openssl rand -hex 16)
echo "✅ ENCRYPTION_KEY gerado (32 caracteres):"
echo "   $ENCRYPTION_KEY"
echo ""

echo "=================================================="
echo "📝 Copie estes valores para o ficheiro .env:"
echo "=================================================="
echo ""
echo "SESSION_SECRET=$SESSION_SECRET"
echo "CSRF_AUTH_KEY=$CSRF_AUTH_KEY"
echo "ENCRYPTION_KEY=$ENCRYPTION_KEY"
echo ""
echo "⚠️  IMPORTANTE:"
echo "   - NÃO partilhe estes valores"
echo "   - NÃO faça commit destes valores no Git"
echo "   - Use valores diferentes em desenvolvimento/produção"
echo "   - Guarde os valores de produção num Secret Manager"
echo ""

# Opção para atualizar .env automaticamente
read -p "Atualizar ficheiro .env automaticamente? (s/n) " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Ss]$ ]]; then
    ENV_FILE="$PROJECT_ROOT/.env"
    ENV_EXAMPLE="$PROJECT_ROOT/.env.example"
    
    if [ -f "$ENV_FILE" ]; then
        # Backup do .env atual
        cp "$ENV_FILE" "$ENV_FILE.backup"
        echo "✅ Backup criado: .env.backup"
        
        # Atualizar ou adicionar valores
        # Função para atualizar ou adicionar linha no .env
        update_or_add_env_var() {
            local key="$1"
            local value="$2"
            local file="$3"
            
            if grep -q "^${key}=" "$file"; then
                # Linha existe, substituir
                if [[ "$OSTYPE" == "darwin"* ]]; then
                    sed -i '' "s|^${key}=.*|${key}=${value}|" "$file"
                else
                    sed -i "s|^${key}=.*|${key}=${value}|" "$file"
                fi
            else
                # Linha não existe, adicionar
                echo "${key}=${value}" >> "$file"
            fi
        }
        
        update_or_add_env_var "SESSION_SECRET" "$SESSION_SECRET" "$ENV_FILE"
        update_or_add_env_var "CSRF_AUTH_KEY" "$CSRF_AUTH_KEY" "$ENV_FILE"
        update_or_add_env_var "ENCRYPTION_KEY" "$ENCRYPTION_KEY" "$ENV_FILE"
        
        echo "✅ Ficheiro .env atualizado!"
        echo "   Backup anterior guardado em .env.backup"
        echo "   Localização: $ENV_FILE"
    else
        echo "⚠️  Ficheiro .env não encontrado em: $ENV_FILE"
        
        if [ -f "$ENV_EXAMPLE" ]; then
            echo ""
            read -p "Criar .env a partir do .env.example? (s/n) " -n 1 -r
            echo ""
            if [[ $REPLY =~ ^[Ss]$ ]]; then
                cp "$ENV_EXAMPLE" "$ENV_FILE"
                echo "✅ Ficheiro .env criado!"
                
                # Atualizar secrets no novo .env
                update_or_add_env_var() {
                    local key="$1"
                    local value="$2"
                    local file="$3"
                    
                    if grep -q "^${key}=" "$file"; then
                        if [[ "$OSTYPE" == "darwin"* ]]; then
                            sed -i '' "s|^${key}=.*|${key}=${value}|" "$file"
                        else
                            sed -i "s|^${key}=.*|${key}=${value}|" "$file"
                        fi
                    else
                        echo "${key}=${value}" >> "$file"
                    fi
                }
                
                update_or_add_env_var "SESSION_SECRET" "$SESSION_SECRET" "$ENV_FILE"
                update_or_add_env_var "CSRF_AUTH_KEY" "$CSRF_AUTH_KEY" "$ENV_FILE"
                update_or_add_env_var "ENCRYPTION_KEY" "$ENCRYPTION_KEY" "$ENV_FILE"
                
                echo "✅ Secrets adicionados ao novo .env!"
                echo "   Localização: $ENV_FILE"
            else
                echo "ℹ️  Copie manualmente:"
                echo "   cp .env.example .env"
                echo "   Depois adicione os secrets acima"
            fi
        else
            echo "   Ficheiro .env.example também não encontrado."
            echo "   Crie manualmente o ficheiro .env na raiz do projeto."
        fi
    fi
else
    echo "ℹ️  Copie manualmente os valores acima para o ficheiro .env na raiz do projeto:"
    echo "   Localização: $PROJECT_ROOT/.env"
fi

echo ""
echo "✅ Concluído!"
