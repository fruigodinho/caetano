# HTTPS/TLS - Guia Rápido

## Desenvolvimento

Executar com HTTPS:
```bash
make run-web
```

Aceder em: `https://local.rswebportal.com:8080`

## Produção

1. Compilar:
```bash
make build-server
```

2. Copiar para servidor:
```bash
scp bin/server .env.production user@servidor:/opt/saldos-esperados/
```

3. No servidor:
```bash
cd /opt/saldos-esperados
mv .env.production .env
sudo ./server
```

Aceder em: `https://se.rswebportal.com`

## Certificados

- **Desenvolvimento**: `/etc/letsencrypt/live/local.rswebportal.com/`
- **Produção**: `/etc/letsencrypt/live/se.rswebportal.com/`

## Documentação Completa

Ver `docs/HTTPS-SETUP.md` para instruções detalhadas.
