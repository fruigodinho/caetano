# Scripts Auxiliares - Saldos Esperados

## generate-secrets.sh

Script para gerar secrets seguros para as variáveis de ambiente de segurança.

### Uso

**Da raiz do projeto:**
```bash
bash scripts/generate-secrets.sh
```

**Do diretório scripts:**
```bash
cd scripts
./generate-secrets.sh
```

### O que faz

1. **Gera SESSION_SECRET** (44 caracteres base64)
   - Usado para encriptar sessões HTTP
   - Deve ter pelo menos 32 caracteres

2. **Gera CSRF_AUTH_KEY** (32 caracteres hex)
   - Usado para proteção CSRF
   - Deve ter exatamente 32 caracteres

3. **Opções de atualização**
   - Pode atualizar `.env` automaticamente
   - Cria backup do `.env` existente (`.env.backup`)
   - Pode criar `.env` a partir do `.env.example` se não existir

### Exemplo de Output

```
🔐 Gerador de Secrets Seguros - Saldos Esperados
==================================================

✅ SESSION_SECRET gerado (44 caracteres):
   X4hHml2IoOMt8LmNQZDxYxi+i1m4Z4hMrFFOzJr+3Ug=

✅ CSRF_AUTH_KEY gerado (32 caracteres):
   ce79ffae68f11265d33340a394358d66

==================================================
📝 Copie estes valores para o ficheiro .env:
==================================================

SESSION_SECRET=X4hHml2IoOMt8LmNQZDxYxi+i1m4Z4hMrFFOzJr+3Ug=
CSRF_AUTH_KEY=ce79ffae68f11265d33340a394358d66

Atualizar ficheiro .env automaticamente? (s/n)
```

### Requisitos

- **openssl** - Para geração de valores aleatórios criptograficamente seguros
  ```bash
  # Ubuntu/Debian
  sudo apt-get install openssl
  
  # Fedora/RHEL
  sudo dnf install openssl
  
  # macOS
  brew install openssl
  ```

### Compatibilidade

- ✅ Linux (testado em Pop!_OS, Ubuntu)
- ✅ macOS (usa `sed -i ''` automaticamente)
- ✅ WSL (Windows Subsystem for Linux)

### Segurança

⚠️ **IMPORTANTE:**
- **NÃO** partilhe os secrets gerados
- **NÃO** faça commit dos valores no Git
- Use valores **diferentes** em desenvolvimento e produção
- Em produção, guarde os secrets num **Secret Manager** (AWS Secrets Manager, HashiCorp Vault, etc.)
- O `.env` deve estar no `.gitignore`

### Regenerar Secrets

Pode executar o script múltiplas vezes para gerar novos secrets. O script criará sempre um backup do `.env` anterior.

```bash
bash scripts/generate-secrets.sh
# Escolher 's' para atualizar automaticamente
# Backup criado em .env.backup
```

### Manual

Se preferir adicionar manualmente:

1. Gerar secrets:
   ```bash
   # SESSION_SECRET
   openssl rand -base64 32
   
   # CSRF_AUTH_KEY  
   openssl rand -hex 16
   ```

2. Adicionar ao `.env`:
   ```bash
   SESSION_SECRET=<valor_gerado>
   CSRF_AUTH_KEY=<valor_gerado>
   ```

### Troubleshooting

**Erro: "openssl: command not found"**
```bash
# Instalar openssl
sudo apt-get update && sudo apt-get install openssl
```

**Erro: "Permission denied"**
```bash
# Tornar script executável
chmod +x scripts/generate-secrets.sh
```

**Script não encontra .env**
```bash
# Verificar que está na raiz do projeto
pwd
# Deve mostrar: /path/to/saldos-esperados

# Criar .env se não existir
cp .env.example .env
```

---

## Outros Scripts

(Adicionar aqui quando houver mais scripts)
