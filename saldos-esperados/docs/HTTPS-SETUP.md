# Configuração HTTPS/TLS

Este documento descreve como a aplicação está configurada para usar certificados SSL/TLS em diferentes ambientes.

## Visão Geral

A aplicação suporta dois ambientes com certificados diferentes:

| Ambiente | Domínio | Porta | Certificado |
|----------|---------|-------|-------------|
| Desenvolvimento | local.rswebportal.com | 8080 | `/etc/letsencrypt/live/local.rswebportal.com/` |
| Produção | se.rswebportal.com | 443 | `/etc/letsencrypt/live/se.rswebportal.com/` |

## Desenvolvimento Local

### Pré-requisitos

1. Certificado Let's Encrypt já configurado para `local.rswebportal.com`
2. Entrada no `/etc/hosts`:
   ```
   127.0.0.1 local.rswebportal.com
   ```

### Executar o Servidor

```bash
make run-web
```

Este comando:
- Inicia o túnel SSH para a base de dados
- Executa o servidor com `sudo` (necessário para aceder aos certificados em `/etc/letsencrypt/`)
- Servidor disponível em: `https://local.rswebportal.com:8080`

### Desativar HTTPS (Desenvolvimento sem TLS)

Edita o ficheiro `.env` e altera:
```bash
SERVER_ENABLE_TLS=false
```

Depois executa sem `sudo`:
```bash
go run cmd/server/main.go
```

## Produção

### Configuração do Certificado

No servidor de produção, obtém o certificado para `se.rswebportal.com`:

```bash
sudo certbot certonly \
  --dns-cloudflare \
  --dns-cloudflare-credentials /root/.secrets/cloudflare.ini \
  -d se.rswebportal.com \
  -d '*.se.rswebportal.com'
```

### Build e Deploy

1. Compilar o binário:
   ```bash
   make build-server
   ```

2. Copiar o binário para o servidor de produção:
   ```bash
   scp bin/server user@servidor:/opt/saldos-esperados/
   ```

3. Copiar o ficheiro de configuração:
   ```bash
   scp .env.production user@servidor:/opt/saldos-esperados/.env
   ```

4. No servidor, executar:
   ```bash
   sudo ./server
   ```

### Script de Mudança de Ambiente

Para facilitar a mudança entre ambientes localmente:

```bash
# Mudar para produção (simula ambiente de produção)
./scripts/switch-env.sh prod

# Voltar ao desenvolvimento
./scripts/switch-env.sh dev
```

## Estrutura de Configuração

### Variáveis de Ambiente

```bash
# Ativar/desativar TLS
SERVER_ENABLE_TLS=true

# Porta do servidor
SERVER_PORT=8080  # ou 443 para produção

# Caminhos dos certificados
SERVER_TLS_CERT=/etc/letsencrypt/live/DOMINIO/fullchain.pem
SERVER_TLS_KEY=/etc/letsencrypt/live/DOMINIO/privkey.pem
```

### Código Go

A lógica de TLS está implementada em:
- `cmd/server/main.go` - Lê configurações e decide HTTP ou HTTPS
- `internal/adapter/web/server.go` - Método `RunTLS()` para servidor HTTPS

## Renovação de Certificados

### Desenvolvimento (local.rswebportal.com)

O certificado é renovado automaticamente pelo certbot (systemd timer), mas como é um PC que pode estar desligado:

```bash
# Verificar validade
sudo certbot certificates

# Renovar manualmente (se necessário)
sudo certbot renew
```

### Produção (se.rswebportal.com)

O servidor deve estar sempre ligado, portanto a renovação automática funcionará normalmente.

Para forçar renovação:
```bash
sudo certbot renew --force-renewal
```

## Troubleshooting

### Erro: "Permission denied" ao aceder certificados

**Causa**: Os certificados em `/etc/letsencrypt/` requerem permissões de root.

**Solução**: Executar com `sudo`:
```bash
sudo -E go run cmd/server/main.go
# ou
sudo ./bin/server
```

### Erro: "Certificate not found"

**Causa**: Certificado ainda não foi gerado.

**Solução**: Seguir instruções de obtenção do certificado no início deste documento.

### Browser mostra "Certificado inválido"

**Causa**: DNS não está a resolver corretamente.

**Solução**: 
1. Verificar `/etc/hosts` (desenvolvimento)
2. Verificar DNS público (produção)
3. Limpar cache DNS: `sudo systemd-resolve --flush-caches`

## Comandos Úteis

```bash
# Verificar porta em uso
sudo netstat -tulpn | grep :443

# Testar certificado
openssl s_client -connect local.rswebportal.com:8080 -servername local.rswebportal.com

# Verificar configuração TLS atual
curl -v https://local.rswebportal.com:8080
```
