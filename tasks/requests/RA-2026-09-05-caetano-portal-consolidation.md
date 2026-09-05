---
id: RA-2026-09-05-caetano-portal-consolidation
title: Consolidar xmldri e saldos-esperados num portal único com SSO Cloudflare e ACL centralizada
status: ready
created: 2026-09-05
priority: must
repo: caetano
stack: go+gin
reversibility: one-way
---

# Consolidar xmldri e saldos-esperados num portal único com SSO Cloudflare e ACL centralizada

## 1. Resumo

Criar o mono-repo Go multi-módulo `caetano`, com um módulo `core` do qual dependem os
módulos `home`, `xmldri` e `saldos-esperados`. `home` corre em `https://se.rswebportal.com`
como ponto de entrada único, autentica todos os pedidos via Cloudflare Access e serve
`xmldri` e `saldos-esperados` como sub-aplicações em `/xmldri` e `/saldos-esperados`,
cada uma com UI reescrita de raiz e ACL centralizada por aplicação e por área.

## 2. Problema e evidência

Hoje `xmldri` e `saldos-esperados` são aplicações Go independentes, cada uma com o seu
próprio ciclo de deploy, sem ponto de entrada comum. `saldos-esperados` tem login e 2FA
próprios (bcrypt + TOTP); `xmldri` não tem autenticação nenhuma (confirmado em
`xmldri/app/routes.go` e `xmldri/app/controller.go` - nenhuma rota está protegida). Não
existe controlo de acesso transversal às duas aplicações nem por área dentro de cada
uma; cada utilizador tem de ser gerido em cada sistema separadamente.

Hipótese não validada: a manutenção de dois sistemas de login e UIs divergentes já
representa esforço duplicado de suporte e uma superfície de segurança maior (uma app sem
autenticação nenhuma).

## 3. Resultado esperado e métrica de sucesso

**Resultado**

Um utilizador autenticado pelo Cloudflare Access consegue aceder a `home`, `xmldri` e
`saldos-esperados` a partir de `https://se.rswebportal.com` com um único início de
sessão, vendo apenas as aplicações e áreas para que tem permissão pela ACL centralizada.
Em desenvolvimento, o mesmo fluxo funciona sem Cloudflare, com autologin do utilizador
master definido em configuração.

**Métrica**

Não existe métrica quantitativa de negócio para esta migração (não há utilizadores
finais externos nem SLA definido); o critério de sucesso é binário e verifica-se pela
secção 11: as três aplicações respondem sob o domínio único, o login próprio e o 2FA de
`saldos-esperados` deixam de existir, e a matriz de ACL (aplicação x área x role) produz
o resultado esperado para cada papel testado.

## 4. Âmbito

### 4.1 Dentro de âmbito

- Módulo `core`: autenticação Cloudflare Access (validação JWT, reaproveitando o padrão
  de `gonotesweb/internal/service/cfaccess.go` e `internal/middleware/auth.go`),
  autologin de dev, gestão de utilizadores/roles/ACL, configuração por ambiente (YAML),
  acesso a Postgres.
- Módulo `home`: processo principal (`main.go`), router Gin raiz, monta `xmldri` e
  `saldos-esperados` como sub-routers, página de entrada que lista as aplicações
  visíveis ao utilizador autenticado.
- Migração de `xmldri` e `saldos-esperados` para dependerem de `core` em vez de terem
  autenticação própria; passam a responder em `/xmldri` e `/saldos-esperados`.
- Remoção do login por password e do 2FA de `saldos-esperados`; remoção de qualquer
  verificação de autenticação própria em `xmldri`.
- ACL de duas camadas: por aplicação (quem pode entrar em `xmldri` / `saldos-esperados`)
  e por área dentro da aplicação (ex.: uploads, tipos, utilizadores, auditoria em
  saldos-esperados). Áreas geridas em BD por um admin via UI.
- Quatro roles: `admin` (acesso total, sem passar pela ACL), `manager`, `leader`,
  `user` - os três últimos controlados pela ACL.
- Reescrita de raiz da UI/UX de `xmldri` e `saldos-esperados`: novo conjunto de CSS/JS
  partilhado, com equivalência funcional 1:1 face ao que existe hoje.
- Script de migração pontual (one-off, só leitura) que liga por túnel direto às BDs de
  produção atuais para importar os dados existentes de `saldos-esperados` (utilizadores
  antigos ficam de fora, dado o novo modelo de autenticação).
- Ficheiro YAML de configuração/segredos por ambiente em `core`, fora do controlo de
  versões, com um exemplo (`*.example.yaml`) versionado.
- Backup diário integral da BD `caetano` para o Dropbox, seguindo o mesmo padrão já
  implementado em `gonotesweb` (`pg_dump` -> gzip -> encriptação AES-256-GCM -> upload
  -> rotação), reaproveitando `DropboxService` e `crypto.go` já existentes em
  `saldos-esperados/internal/service`.

### 4.2 Fora de âmbito

- Revisão dos fluxos funcionais existentes de `xmldri` e `saldos-esperados` (decisão
  registada em D-4): a reescrita é só da camada de UI/UX e da estrutura de templates,
  mantendo o comportamento atual.
- Migração de utilizadores, passwords ou configuração de 2FA de `saldos-esperados` -
  deixam de existir enquanto conceito; não há dados de conta a transportar, só dados de
  negócio (balancetes, regras, histórico).
- Qualquer módulo além dos quatro definidos (`core`, `home`, `xmldri`,
  `saldos-esperados`).
- Funcionalidade permanente de reimportação de BD de produção (decisão D-3): fica só o
  procedimento pontual de migração inicial.
- Gestão de áreas de ACL fora de uma UI simples de administração (sem workflow de
  aprovação, sem versionamento de permissões).

## 5. Utilizadores, cenários e volume

Utilizadores internos da organização, autenticados via Cloudflare Access com o domínio
corporativo. Volume não determinado com precisão pelo utilizador; assume-se escala
pequena (dezenas de contas, não milhares), compatível com o volume atual de
`saldos-esperados` (aplicação interna de validação de balancetes). Sem requisito de
disponibilidade formal (SLA) além de "aplicação interna, uso em horário de expediente".

Frequência de uso: `saldos-esperados` é usada em fecho de balancetes (uso periódico,
não contínuo); `xmldri` é uma ferramenta de conversão pontual. `home` é o ponto de
entrada de cada sessão de trabalho.

## 6. Requisitos funcionais

- **RF-1** O sistema deve autenticar todos os pedidos a `home`, `xmldri` e
  `saldos-esperados` através da validação do JWT do Cloudflare Access (header
  `Cf-Access-Jwt-Assertion` ou cookie `CF_Authorization`), rejeitando com `401` qualquer
  pedido sem token válido.
- **RF-2** Em ambiente de desenvolvimento (e nunca em produção), o sistema deve
  autenticar automaticamente o utilizador cujo email está definido no YAML de
  configuração de `core`, desde que esse utilizador já exista e esteja ativo na BD.
- **RF-3** O sistema deve impedir o arranque em modo autologin de dev quando a
  configuração de ambiente indica produção, mesmo que o parâmetro de autologin esteja
  presente no ficheiro.
- **RF-4** O sistema deve resolver, para cada utilizador autenticado, a lista de
  aplicações (`xmldri`, `saldos-esperados`) a que tem acesso, e mostrar em `home` apenas
  essas.
- **RF-5** Cada grupo de rotas protegido de `xmldri` e `saldos-esperados` deve declarar
  a área de ACL a que corresponde (ex.: grupo de rotas de uploads declara a área
  "uploads" de `saldos-esperados`); o sistema deve negar por omissão (`403`) o acesso a
  um grupo de rotas cuja área não esteja concedida ao utilizador autenticado.
- **RF-6** O utilizador com role `admin` deve ter acesso a todas as aplicações e a todas
  as áreas/rotas, sem consulta à tabela de ACL; qualquer outro role (`manager`,
  `leader`, `user`) começa sem nenhum acesso, exceto o que lhe for concedido
  explicitamente na ACL.
- **RF-7** O sistema deve permitir a um `admin` gerir, via UI, o catálogo de áreas de
  ACL (uma entrada por grupo de rotas declarado por cada módulo) e as permissões de cada
  utilizador ou role por aplicação e por área.
- **RF-8** O sistema deve servir `xmldri` sob o caminho `/xmldri` e
  `saldos-esperados` sob o caminho `/saldos-esperados`, a partir do mesmo processo e do
  mesmo domínio que serve `home`.
- **RF-9** O sistema não deve expor nenhum formulário de login ou de configuração de 2FA
  em `xmldri` ou `saldos-esperados`; a autenticação é sempre resolvida por `core`.
- **RF-10** O sistema deve carregar as credenciais e tokens de configuração (BD,
  Cloudflare, integrações) a partir de um ficheiro YAML por ambiente gerido em `core`,
  nunca de valores fixos no código.
- **RF-11** O sistema deve disponibilizar um comando de migração que, ligado por túnel
  SSH direto à BD de produção existente de `saldos-esperados` (credenciais em
  `saldos-esperados/systemd.env`, túnel aberto com `make tunnel-start`, conforme
  `scripts/db-tunnel.sh`), importa os dados de negócio (regras, balancetes, histórico)
  para o schema `caetano` do novo sistema.
- **RF-12** As páginas de `xmldri` e `saldos-esperados` devem usar uma base de CSS/JS
  comum e nova, sem reaproveitar os ficheiros de estilo/scripts atuais de nenhum dos
  dois.
- **RF-13** O sistema deve executar diariamente um backup integral da BD `caetano`
  (`pg_dump` comprimido e encriptado com AES-256-GCM) e fazer upload para o Dropbox,
  mantendo uma retenção de backups configurável (default: 3), eliminando os mais antigos
  após cada backup bem-sucedido.

## 7. Requisitos não-funcionais

- Segurança: nenhum segredo (credenciais de BD, chaves, tokens) fica versionado no
  git; só o ficheiro de exemplo por ambiente é versionado.
- Segurança: o modo de autologin de dev (RF-2/RF-3) tem de ser estruturalmente
  impossível de ativar quando `APP_ENV` (ou equivalente) indica produção - validado no
  arranque, não só documentado.
- Persistência: schema único `caetano` em Postgres, tabelas prefixadas por módulo
  (`core_*`, `xmldri_*`, `saldos_esperados_*`), conforme decisão do utilizador em D-2.
- Idioma: mensagens ao utilizador e logs em português europeu, consistente com o padrão
  já usado em `saldos-esperados/AGENTS.md` e `gonotesweb`.
- Retenção: backups da BD mantidos no Dropbox por 3 cópias mais recentes (rotação
  automática), alinhado com o default já usado em `gonotesweb`.
- Segurança: o dump de backup nunca sai para o Dropbox sem encriptação AES-256-GCM.

## 8. Decisões e desvios

| # | Decisão | Recomendação | Escolha | Justificação | Consequência aceite | Rev. |
|---|---------|--------------|---------|---------------|---------------------|------|
| D-2 | Fronteira de dados entre módulos | Uma instância Postgres, um schema por módulo | Schema único `caetano`, tabelas prefixadas por módulo | Poucas tabelas e pouco volume de dados | Perda do isolamento lógico de schema; convenção de prefixo tem de ser respeitada manualmente em todas as migrações futuras | one-way |
| D-1 | Composição dos processos | Binário único (home importa xmldri e saldos-esperados como bibliotecas Go) | Binário único | - | - | one-way |
| D-3 | Alcance da importação de BD de produção | Ligação direta por túnel, procedimento pontual | Túnel direto, pontual | - | - | two-way |
| D-4 | Alcance da reescrita de UI | Equivalência funcional 1:1, só camada visual | Equivalência funcional 1:1 | - | - | two-way |
| D-5 | Sessão pós-Cloudflare | Stateless, revalida o JWT a cada pedido | Stateless | - | - | two-way |
| D-6 | Existência do utilizador master em dev | Autologin falha se o utilizador não existir na BD | Tem de existir | - | - | two-way |
| D-7 | Granularidade das áreas de ACL | Áreas fixas no código de cada módulo | Áreas configuráveis em BD, geridas por admin, com a enforcement ao nível do grupo de rotas | Utilizador quer gestão das áreas sem alterar código; o ponto de imposição fica no grupo de rotas de cada módulo (RF-5), o que reduz o risco de desalinhamento | Cada grupo de rotas protegido tem de declarar explicitamente a área a que corresponde; o middleware de autorização tem de negar por omissão (RF-5) qualquer rota sem área declarada, para não ficar acessível por esquecimento | one-way |
| D-8 | Backup da BD | Backup diário automático, comprimido e encriptado, com rotação | Reaproveitar 1:1 o `BackupService` do gonotesweb (pg_dump -> gzip -> AES-256-GCM -> Dropbox -> retenção 3) | - | - | two-way |

## 9. Riscos e questões em aberto

**Riscos**

- Desalinhamento entre as áreas de ACL geridas em BD (D-7) e os pontos de código
  realmente protegidos, se um módulo acrescentar uma rota sem registar a área
  correspondente.
- `xmldri` não tem hoje persistência nem conceito de utilizador; a integração com ACL
  por área implica desenhar de raiz essa camada nesse módulo.
- Ligação por túnel direto à BD de produção (D-3) implica acesso de rede a um ambiente
  produtivo a partir da máquina de quem faz a migração; `saldos-esperados/systemd.env`
  contém credenciais em claro e não deve ser copiado para o repo `caetano`, só lido
  localmente no momento da migração.

**Questões em aberto**

- Volume real de utilizadores e frequência de uso (secção 5 assume escala pequena por
  falta de número fornecido) - não bloqueia o desenho, mas falta confirmar que não há
  requisito de desempenho a considerar.
- Formato exato do YAML de configuração por ambiente em `core` (nomes de chaves,
  localização do ficheiro, mecanismo de leitura por ambiente) - fica para a fase de
  plano de implementação, não é uma decisão de requisito.
- Nomes iniciais das áreas de ACL por módulo ficam definidos pelo catálogo de grupos de
  rotas que a implementação vier a declarar (RF-5/RF-7); não é uma decisão a fechar
  agora, é resolvida pelo próprio desenho de rotas.

## 10. Critérios de aceitação

- [ ] (RF-1) Dado um pedido a `/saldos-esperados` sem `Cf-Access-Jwt-Assertion` nem
      cookie `CF_Authorization`, quando chega ao sistema em modo produção, então a
      resposta é `401`.
- [ ] (RF-1) Dado um pedido com um JWT do Cloudflare Access válido para um utilizador
      ativo, quando chega a qualquer uma das três aplicações, então o pedido é aceite e
      o utilizador fica identificado no contexto do pedido.
- [ ] (RF-2) Dado o sistema a correr em modo dev com o email do YAML correspondente a um
      utilizador ativo na BD, quando se acede a qualquer rota, então o pedido é
      autenticado automaticamente com esse utilizador, sem pedir login.
- [ ] (RF-2) Dado o sistema a correr em modo dev com o email do YAML sem utilizador
      correspondente ativo na BD, quando se tenta aceder a qualquer rota, então o acesso
      é recusado (`403`) e não se cria nenhum utilizador automaticamente.
- [ ] (RF-3) Dado o sistema configurado com a flag de ambiente de produção, quando o
      ficheiro de configuração também tem um email de autologin definido, então o
      sistema arranca sem ativar o autologin e sem expor a rota correspondente.
- [ ] (RF-4/RF-5) Dado um utilizador com role `user` e ACL só para a área "uploads" de
      `saldos-esperados`, quando acede a `home`, então só vê `saldos-esperados` listada;
      quando tenta aceder ao grupo de rotas da área "utilizadores" dessa aplicação,
      então recebe `403`.
- [ ] (RF-5) Dado um utilizador com role `manager` recém-criado, sem nenhuma entrada na
      ACL, quando tenta aceder a qualquer grupo de rotas protegido de `xmldri` ou
      `saldos-esperados`, então recebe `403` (negado por omissão).
- [ ] (RF-6) Dado um utilizador com role `admin`, quando acede a qualquer aplicação ou
      grupo de rotas existente, então o acesso é sempre permitido, mesmo sem entrada na
      tabela de ACL para esse caso.
- [ ] (RF-7) Dado um `admin` autenticado, quando cria uma nova área de ACL para
      `xmldri` e atribui permissão a um `manager`, então esse `manager` passa a ver essa
      área imediatamente, sem reiniciar a aplicação.
- [ ] (RF-8) Dado o sistema em execução, quando se pede `https://se.rswebportal.com/xmldri`
      e `https://se.rswebportal.com/saldos-esperados`, então ambos respondem a partir do
      mesmo processo, sem redireção para outro host ou porta.
- [ ] (RF-9) Dado o novo sistema em produção, quando se procura por rotas de login ou de
      configuração de 2FA em `xmldri` ou `saldos-esperados`, então nenhuma existe.
- [ ] (RF-10) Dado o repositório clonado de raiz, quando se procura por credenciais ou
      tokens no código-fonte versionado, então nenhum é encontrado; existe apenas um
      ficheiro de exemplo sem valores reais.
- [ ] (RF-11) Dado o script de migração corrido contra a BD de produção atual via
      `make tunnel-start` (túnel de `saldos-esperados/systemd.env`), quando termina,
      então o número de registos de negócio (regras, balancetes, histórico) no schema
      `caetano` corresponde ao número existente na origem.
- [ ] (RF-13) Dado o serviço de backup em execução, quando chega a hora agendada (01:00),
      então aparece no Dropbox, na pasta de backups, um novo ficheiro `.sql.gz.enc` com
      timestamp do dia; quando existem mais de 3 backups na pasta, então os mais antigos
      são eliminados até restarem 3.
- [ ] (RF-13) Dado um backup gerado, quando se descarrega e corre
      `make backup-decrypt` (ou equivalente em `caetano`) com a `ENCRYPTION_KEY`
      correta, então o `.sql.gz` resultante restaura sem erros contra uma BD de teste.
- [ ] (RF-12) Dado o inventário de ficheiros CSS/JS de `xmldri/public` e
      `saldos-esperados/web/assets`, quando se compara com os ficheiros usados pelas
      páginas depois da reescrita, então nenhum dos ficheiros antigos está referenciado.

## 11. Plano de verificação

1. `go build ./...` a partir da raiz do mono-repo, com todos os quatro módulos
   (`core`, `home`, `xmldri`, `saldos-esperados`) a compilar num único binário via
   `home`.
2. `go test ./...` em cada módulo, com testes de autenticação Cloudflare (mock de JWKS,
   como já existe em `gonotesweb/internal/service/cfaccess_test.go`) e de resolução de
   ACL por aplicação/área/role.
3. Arrancar em modo dev sem token Cloudflare e confirmar autologin com o utilizador
   master do YAML; parar o servidor, remover esse utilizador da BD e confirmar que o
   arranque falha com `403` em vez de criar o utilizador.
4. Arrancar com a flag de produção ativa e um email de autologin presente na
   configuração; confirmar nos logs que o autologin não foi ativado.
5. Testar manualmente os quatro roles (`admin`, `manager`, `leader`, `user`) com
   combinações de ACL distintas em `/xmldri` e `/saldos-esperados`, confirmando que cada
   um só vê e acede às áreas concedidas.
6. Correr o script de migração pontual contra um dump ou réplica de teste da BD de
   produção de `saldos-esperados` (nunca diretamente contra produção na primeira
   execução) e comparar contagens de linhas antes/depois por tabela.
7. Inspeção visual das páginas de `xmldri` e `saldos-esperados` após a reescrita,
   confirmando a mesma funcionalidade (upload de CSV, validação de balancetes) com a UI
   nova.
8. Correr o backup manualmente (`RunBackup` equivalente ao de `gonotesweb`) contra uma
   BD de teste e confirmar no Dropbox o ficheiro encriptado; testar a rotação criando
   mais de 3 backups e confirmando que só ficam os 3 mais recentes.

## 12. Referências

- `/home/rgodinho/Workspaces/GolangProjects/gonotesweb` - padrão de referência para
  autenticação Cloudflare Access, autologin de dev, ligação a Postgres e backup diário
  para Dropbox (`internal/service/cfaccess.go`, `internal/middleware/auth.go`,
  `internal/config/config.go`, `internal/service/backup.go`, `scripts/db-tunnel.sh`).
- `saldos-esperados/internal/service/dropbox.go`, `saldos-esperados/internal/service/crypto.go`
  - `DropboxService` e encriptação AES-256-GCM já existentes, a reaproveitar para RF-13.
- `saldos-esperados/systemd.env`, `saldos-esperados/Makefile` (`tunnel-start`),
  `saldos-esperados/scripts/db-tunnel.sh` - credenciais e procedimento do túnel para a
  migração pontual (RF-11). Ficheiro com segredos em claro; não copiar para o repo
  `caetano`.
- `agents/go-project-architecture.md` - arquitetura de referência para novos projetos Go
  multi-módulo (a validar contra este desenho na fase de plano).
- `saldos-esperados/AGENTS.md`, `saldos-esperados/README.md` - inventário funcional
  atual de `saldos-esperados` a preservar (D-4).
