---
id: RA-2026-09-05-admin-users-acl-access
title: Criar administração de utilizadores e expor o acesso à ACL
status: ready
created: 2026-09-05
priority: must
repo: caetano
stack: go+gin
reversibility: two-way
---

# Criar administração de utilizadores e expor o acesso à ACL

## 1. Resumo

O `caetano` tem hoje a mecânica de ACL (`core/acl`) e o ecrã de concessão
`/admin/acl`, mas não tem nenhum ecrã para criar ou gerir os utilizadores em
`core_users` - só existe inserção manual por SQL. Falta também tornar o
`/admin/acl` alcançável a partir da navegação, e falta o botão de logout,
presente no layout mas nunca ligado a nada.

## 2. Problema e evidência

- `core/users/store.go` só tem `GetUserByEmail`, `GetUserByID` e `List` - não
  existe `Create` nem `Update`. O único utilizador em `core_users` (o admin de
  dev) foi inserido à mão por SQL nesta sessão.
- `home/templates/pages/admin_acl.html` e a rota `/admin/acl` já existem e
  funcionam (confirmado em sessão anterior), mas nenhuma página tem um link
  para lá - só é alcançável escrevendo o URL de cor.
- `core/web/templates/layouts/authenticated.html:18` já tem
  `{{if .LogoutURL}}<a ...>Sair</a>{{end}}`, mas nenhum handler define
  `LogoutURL` em `gin.H` - o botão nunca aparece.

## 3. Resultado esperado e métrica de sucesso

**Resultado**

Um admin consegue, sem tocar em SQL, criar um utilizador (email + role),
atribuir-lhe uma área de ACL, e o próprio utilizador consegue depois entrar e
ver só o que lhe foi concedido. Qualquer utilizador autenticado em produção
consegue terminar sessão a partir do topbar.

**Métrica**

Sem métrica quantitativa (funcionalidade administrativa interna, não há
volume de utilização a medir); o critério de sucesso é os critérios de
aceitação da secção 10 passarem, verificados manualmente após o deploy.

## 4. Âmbito

### 4.1 Dentro de âmbito

- `core/users/store.go`: `Create(ctx, email, name, role) (auth.User, error)` e
  `Update(ctx, id, name, role, enabled) error`.
- Novo ecrã `/admin/users` (`RequireAdmin`, mesmo padrão de
  `home/handler/admin_acl.go`): listar, criar, editar (nome/role/ativo).
  Sem password nem 2FA - a identidade continua a vir só do Cloudflare Access;
  este ecrã só regista quem tem o quê.
- `core/web/templates/layouts/authenticated.html`: mostrar "👥 Utilizadores" e
  "🔐 ACL" no topbar quando `.CurrentUser.IsAdmin` é verdadeiro - sem tocar em
  `Nav` de cada módulo, resolvido uma vez no layout partilhado.
- Botão de logout ligado: em produção aponta para `/cdn-cgi/access/logout`
  (endpoint do Cloudflare Access); em dev fica escondido (sem sessão real do
  Cloudflare para terminar).
- Documentar (README ou nota de deploy) o passo manual de inserir o primeiro
  admin em produção por SQL, com o email `fruigodinho@gmail.com`.

### 4.2 Fora de âmbito

- Seed automático de concessões (grants) por omissão para
  manager/leader/user - decisão explícita desta sessão: fica tudo manual via
  `/admin/acl`, RF-5/RF-6 do requisito anterior (negado por omissão) mantêm-se
  à letra.
- Criação automática do primeiro admin no arranque (via flag de configuração)
  - decisão explícita: inserção manual por SQL.
  - Eliminação (`DELETE`) de utilizadores - usa-se o campo `enabled` já
    existente (desativar, nunca apagar), consistente com o resto do sistema.
- Grupos de utilizadores como conceito novo: `admin`/`manager`/`leader`/`user`
  já são os 4 valores fixos do `CHECK` de `core_users.role` - não se cria
  tabela de "grupos" nem se muda o esquema.

### 4.3 Adiado, com condição de reentrada

- Seed de concessões por omissão: reconsiderar se, depois de alguns deploys,
  se confirmar que o passo manual em `/admin/acl` é repetitivo e sempre igual
  entre ambientes.

## 5. Utilizadores, cenários e volume

Só administradores usam `/admin/users` e `/admin/acl` - volume desprezável
(dezenas de operações, não pedidos por segundo). Sem requisito de desempenho.

## 6. Requisitos funcionais

- **RF-1** O sistema deve permitir a um admin criar um utilizador indicando
  email e role (`admin`, `manager`, `leader`, `user`), sem password.
- **RF-2** O sistema deve rejeitar a criação de um utilizador com um email já
  existente em `core_users`, com uma mensagem clara (não um erro genérico de
  BD).
- **RF-3** O sistema deve permitir a um admin editar o nome, o role e o estado
  (ativo/inativo) de um utilizador existente.
- **RF-4** O sistema não deve permitir a um admin desativar a sua própria
  conta (evita ficar sem nenhum admin ativo por engano).
- **RF-5** O sistema deve mostrar, no topbar partilhado, os links
  "Utilizadores" e "ACL" apenas quando o utilizador autenticado é admin,
  independentemente de em que módulo/página está.
- **RF-6** O sistema deve mostrar um botão de logout apontando para
  `/cdn-cgi/access/logout` quando `IsProduction()` é verdadeiro, e escondê-lo
  quando é falso.
- **RF-7** O sistema deve continuar a negar por omissão o acesso de
  manager/leader/user a qualquer área sem concessão explícita (sem alteração
  face ao comportamento atual).

## 7. Requisitos não-funcionais

- Segurança: `/admin/users` e `/admin/acl` só acessíveis a `admin`
  (`RequireAdmin`), consistente com o resto da administração.
- Idioma: mensagens e rótulos em português europeu, consistente com o resto
  do `caetano`.

## 8. Decisões e desvios

| # | Decisão | Recomendação | Escolha | Justificação | Consequência aceite | Rev. |
|---|---------|--------------|---------|---------------|---------------------|------|
| D-1 | Bootstrap do 1º admin em produção | Inserção manual por SQL | Inserção manual por SQL, email `fruigodinho@gmail.com` | - | - | two-way |
| D-2 | Seed de concessões por omissão | Sem seed, tudo manual via `/admin/acl` | Sem seed, tudo manual | - | - | two-way |
| D-3 | Remoção vs. desativação de utilizador | Desativar (`enabled=false`), nunca apagar | Desativar | - | - | two-way |
| D-4 | Acesso a `/admin/acl` e ao novo `/admin/users` | Link no topbar, só para admin | Link no topbar, só para admin | - | - | two-way |
| D-5 | Comportamento do logout | Prod: `/cdn-cgi/access/logout`; dev: escondido | Prod: `/cdn-cgi/access/logout`; dev: escondido | - | - | two-way |

## 9. Riscos e questões em aberto

**Riscos**

- O email `fruigodinho@gmail.com` só faz sentido como admin de **produção**
  (autenticado via Cloudflare Access); a base de dados de produção do
  `caetano` ainda não existe (só a de dev foi provisionada nesta sessão) -
  o `INSERT` manual fica pendente até essa base ser criada.

**Questões em aberto**

- Nenhuma bloqueante identificada.

## 10. Critérios de aceitação

- [ ] (RF-1) Dado um admin autenticado em `/admin/users/new`, quando submete
      email e role válidos, então o utilizador aparece na listagem com esse
      role.
- [ ] (RF-2) Dado um email já existente em `core_users`, quando um admin tenta
      criar outro utilizador com o mesmo email, então recebe uma mensagem de
      erro específica ("Já existe um utilizador com este email") e nenhuma
      linha nova é criada.
- [ ] (RF-3) Dado um utilizador existente, quando um admin lhe muda o role de
      `user` para `manager` e grava, então o novo role fica visível na
      listagem e passa a ser usado nas verificações de `RequireArea`.
- [ ] (RF-4) Dado um admin autenticado a editar a sua própria conta, quando
      tenta desativá-la, então o sistema recusa a alteração com uma mensagem
      explícita.
- [ ] (RF-5) Dado um utilizador `manager` autenticado, quando visita qualquer
      página (`/`, `/xmldri`, `/saldos-esperados/dashboard`), então o topbar
      **não** mostra "Utilizadores" nem "ACL"; dado o mesmo pedido feito por
      um `admin`, então ambos aparecem.
- [ ] (RF-6) Dado `GIN_MODE=release` no ambiente, quando qualquer utilizador
      autenticado carrega uma página, então o topbar mostra "Sair" a apontar
      para `/cdn-cgi/access/logout`; dado o modo de dev (sem `GIN_MODE`),
      então o botão não aparece.
- [ ] (RF-7) Dado um utilizador `user` recém-criado sem nenhuma concessão em
      `core_acl_grants`, quando tenta aceder a qualquer área de
      `saldos-esperados` ou `xmldri`, então recebe `403`.

## 11. Plano de verificação

1. `go build ./... && go vet ./...` a partir da raiz do workspace.
2. Criar um utilizador de teste via `/admin/users/new` com role `user`;
   confirmar `403` em todas as áreas até se lhe conceder uma em `/admin/acl`,
   e acesso liberado depois da concessão.
3. Tentar desativar a própria conta de admin em `/admin/users` e confirmar a
   recusa (RF-4).
4. Correr com `GIN_MODE=release` num ambiente de teste e confirmar que o
   botão "Sair" aparece e aponta para `/cdn-cgi/access/logout`; correr em dev
   (sem `GIN_MODE`) e confirmar que não aparece.
5. Navegar para `/`, `/xmldri` e `/saldos-esperados/dashboard` autenticado
   como `admin` e depois como `manager`, confirmando a presença/ausência dos
   links "Utilizadores"/"ACL" no topbar em cada caso.

## 12. Referências

- `home/handler/admin_acl.go`, `home/templates/pages/admin_acl.html` - padrão
  a seguir para o novo ecrã de utilizadores.
- `core/users/store.go` - onde acrescentar `Create`/`Update`.
- `core/web/templates/layouts/authenticated.html` - layout partilhado onde
  entram os links de admin e o logout.
- `core/middleware/identity.go`, `core/acl/acl.go` - `RequireAdmin`,
  `RequireArea`, negação por omissão (RF-7).
