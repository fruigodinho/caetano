# Deploy

## Ambientes

### Desenvolvimento
Ver [Instalação](instalacao.md).

### Staging
Réplica de produção para testes finais antes de deploy.

### Produção
Ambiente live com dados reais.

## Build para Produção

### Compilar
```bash
# Build otimizado (remove símbolos de debug)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o bin/server \
    ./cmd/server

# CLI
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o bin/cli \
    ./cmd/cli
```

### Verificar Binário
```bash
ls -lh bin/
file bin/server
# ELF 64-bit LSB executable, statically linked
```

## Deploy com Systemd

### 1. Copiar Binário
```bash
scp bin/server user@servidor:/opt/saldos-esperados/
scp -r web user@servidor:/opt/saldos-esperados/
```

### 2. Criar Service Unit
```ini
# /etc/systemd/system/saldos-esperados.service
[Unit]
Description=Saldos Esperados Web Server
After=network.target postgresql.service

[Service]
Type=simple
User=saldos
WorkingDirectory=/opt/saldos-esperados
ExecStart=/opt/saldos-esperados/bin/server
Restart=always
RestartSec=5
Environment="GIN_MODE=release"
EnvironmentFile=/opt/saldos-esperados/.env

[Install]
WantedBy=multi-user.target
```

### 3. Ativar Serviço
```bash
sudo systemctl daemon-reload
sudo systemctl enable saldos-esperados
sudo systemctl start saldos-esperados
sudo systemctl status saldos-esperados
```

### 4. Ver Logs
```bash
journalctl -u saldos-esperados -f
```

## Nginx Reverse Proxy

### Configuração
```nginx
# /etc/nginx/sites-available/saldos-esperados
server {
    listen 80;
    server_name saldos.empresa.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name saldos.empresa.com;

    # SSL
    ssl_certificate /etc/letsencrypt/live/saldos.empresa.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/saldos.empresa.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Security Headers
    add_header Strict-Transport-Security "max-age=31536000" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "DENY" always;

    # Proxy
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Logs
    access_log /var/log/nginx/saldos-acesso.log;
    error_log /var/log/nginx/saldos-erro.log;
}
```

### Ativar
```bash
sudo ln -s /etc/nginx/sites-available/saldos-esperados /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

## SSL com Let's Encrypt

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d saldos.empresa.com
# Seguir wizard, escolher redirect HTTPS
```

Renovação automática:
```bash
sudo certbot renew --dry-run
# Adiciona CRON automaticamente
```

## Docker (Opcional)

### Dockerfile
```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/bin/server .
COPY --from=builder /app/web ./web
EXPOSE 8080
CMD ["./server"]
```

### docker-compose.yml
```yaml
version: '3.8'
services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      DB_HOST: db
      DB_PORT: 5432
    env_file:
      - .env
    depends_on:
      - db

  db:
    image: postgres:14-alpine
    volumes:
      - postgres_data:/var/lib/postgresql/data
    environment:
      POSTGRES_DB: saldos_esperados
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}

volumes:
  postgres_data:
```

### Executar
```bash
docker-compose up -d
```

## Backups

### Base de Dados
```bash
#!/bin/bash
# backup-db.sh
DATE=$(date +%Y%m%d_%H%M%S)
pg_dump saldos_esperados | gzip > backup_${DATE}.sql.gz
aws s3 cp backup_${DATE}.sql.gz s3://backups-saldos/
find . -name "backup_*.sql.gz" -mtime +30 -delete
```

CRON diário:
```cron
0 2 * * * /opt/scripts/backup-db.sh
```

## Monitorização

### Health Check
Adicionar endpoint:
```go
router.GET("/health", func(c *gin.Context) {
    c.JSON(200, gin.H{"status": "ok"})
})
```

Monitorizar:
```bash
curl http://localhost:8080/health
```

### Prometheus (Opcional)
```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

router.GET("/metrics", gin.WrapH(promhttp.Handler()))
```

## Rollback

```bash
# Parar serviço
sudo systemctl stop saldos-esperados

# Restaurar binário anterior
cp bin/server.backup bin/server

# Restaurar BD (se necessário)
psql saldos_esperados < backup_20250130.sql

# Reiniciar
sudo systemctl start saldos-esperados
```

## Checklist de Deploy

- [ ] Testes passam em staging
- [ ] Backup da BD criado
- [ ] .env configurado em produção
- [ ] HTTPS ativado
- [ ] Systemd service configurado
- [ ] Nginx reverse proxy ativo
- [ ] Logs monit orizados
- [ ] Health check responde
- [ ] Primeiro utilizador admin criado
