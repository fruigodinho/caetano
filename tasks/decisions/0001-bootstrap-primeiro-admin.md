# 0001 - Bootstrap do primeiro admin

**Contexto:** não há UI de criação de utilizadores em produção antes de
existir pelo menos um admin (a UI de `/admin/users` está atrás de
`RequireAdmin`), e não há seed automático (decisão D-1 de
`tasks/requests/RA-2026-09-05-admin-users-acl-access.md`). Sem um admin já
em `core_users`, ninguém consegue autenticar-se com privilégio suficiente
para criar o segundo.

**Decisão:** o primeiro admin de cada ambiente é inserido manualmente por
SQL, antes do primeiro deploy/arranque contra essa base de dados.

**Produção:** email `fruigodinho@gmail.com` (autenticado via Cloudflare
Access com esse email).

```sql
INSERT INTO core_users (email, name, role, enabled)
VALUES ('fruigodinho@gmail.com', 'Rui Godinho', 'admin', true);
```

Correr contra a base de dados de produção do `caetano` assim que for
provisionada, antes do primeiro `systemctl start caetano` (ver
`scripts/caetano.service.example`).

**Ambiente de dev:** já feito nesta sessão, com o email do
`dev.auto_login_email` de `caetano.dev.yaml` (`admin@example.com`).
