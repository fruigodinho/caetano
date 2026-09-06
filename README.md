# Caetano

Mono-repo Go com workspace multi-módulo (`core`, `home`, `xmldri`, `saldos-esperados`).

## Requisitos do host de produção

O host de deploy (CT 121, Alpine/OpenRC) precisa dos seguintes pacotes instalados manualmente (não vêm por omissão numa imagem Alpine mínima):

- `rsync` — usado por `make deploy` / `scripts/deploy.sh` para o upload do binário.
  ```sh
  apk add --no-cache rsync
  ```
- `postgresql-client` (fornece `pg_dump`) — necessário para o backup diário integral da base de dados. Sem isto, o backup falha silenciosamente todos os dias (`pg_dump não encontrado no PATH`), sem afetar a aplicação em si.
  ```sh
  apk add --no-cache postgresql-client
  ```

O binário é sempre compilado com `CGO_ENABLED=0` (ver `Makefile`, targets `build`/`build-prod`) para produzir um executável estático compatível com musl (Alpine) — não linkar dinamicamente contra glibc.

## Deploy

```sh
make deploy          # build + upload (sem restart)
make deploy-restart   # reinicia o serviço e verifica health check
make deploy-status    # logs e estado do serviço
```

Ver `scripts/deploy.sh` para detalhes de configuração (`DEPLOY_HOST`, `DEPLOY_PATH`, `DEPLOY_SERVICE`).
