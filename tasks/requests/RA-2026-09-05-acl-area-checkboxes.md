---
id: RA-2026-09-05-acl-area-checkboxes
title: Substituir a select de área por checkboxes na concessão de ACL
status: ready
created: 2026-09-05
priority: should
repo: caetano
stack: go+gin
reversibility: two-way
---

# Substituir a select de área por checkboxes na concessão de ACL

## 1. Resumo

Na página de administração de ACL (`/admin/acl`), a escolha da área a conceder passa de
uma select de escolha única para uma lista de checkboxes agrupada por aplicação, nos
dois formulários de concessão (utilizador e role), permitindo marcar e conceder várias
áreas numa só submissão em vez de repetir o formulário área a área.

## 3. Resultado esperado e métrica de sucesso

**Resultado**

Um administrador consegue conceder acesso a várias áreas, da mesma aplicação ou de
aplicações diferentes, a um utilizador ou a um role, numa única submissão de um dos
formulários de `/admin/acl`.

**Métrica**

O número de submissões necessárias para conceder N áreas ao mesmo destinatário passa de
N (uma por área) para 1. Não existe log dedicado a esta ação; a confirmação é direta na
UI (contagem de concessões criadas na tabela "Concessões atuais" após uma única
submissão).

## 4. Âmbito

### 4.1 Dentro de âmbito

- Trocar a select "Aplicação / área" por checkboxes das áreas ativas, agrupadas por
  aplicação (um bloco com o nome da app como cabeçalho, checkboxes das suas áreas por
  baixo), nos dois formulários de `home/templates/pages/admin_acl.html`.
- Alterar `home/handler/admin_acl.go:Grant` para aceitar múltiplas áreas na mesma
  submissão e criar uma concessão por área marcada.
- Validar cada área submetida contra o catálogo de áreas ativas (`core_acl_areas`,
  via `core/acl.Store.ListActiveAreas`) antes de gravar qualquer concessão da
  submissão.

### 4.2 Fora de âmbito

- A seleção de "Utilizador" e de "Role" mantém-se como select de escolha única; não
  entra nesta mudança.
- A tabela "Concessões atuais" e o fluxo de revogação não se alteram.
- Não se introduz atalho de seleção (ex.: "marcar todas as áreas desta app") além dos
  checkboxes individuais.

## 6. Requisitos funcionais

- **RF-1** O sistema deve apresentar as áreas ativas como checkboxes agrupadas por
  aplicação (cabeçalho com o nome da app, checkboxes das áreas dessa app por baixo), nos
  dois formulários de concessão de `/admin/acl`.
- **RF-2** O sistema deve permitir marcar mais do que uma área, da mesma aplicação ou de
  aplicações diferentes, e conceder todas as marcadas numa só submissão do formulário.
- **RF-3** O sistema deve impedir a submissão sem nenhuma área marcada: validação no
  formulário (HTML) e, se o pedido chegar ao servidor sem nenhuma área, resposta `400`
  sem criar concessões.
- **RF-4** O sistema deve validar cada área submetida contra o catálogo de áreas ativas
  antes de gravar qualquer concessão; se alguma área da submissão não existir no
  catálogo, o servidor responde `400` e nenhuma concessão dessa submissão é gravada,
  incluindo as áreas válidas do mesmo pedido.
- **RF-5** O sistema deve manter os dois formulários (utilizador e role) independentes,
  cada um com o seu próprio conjunto de checkboxes e a sua própria submissão.

## 8. Decisões e desvios

| # | Decisão | Recomendação | Escolha | Justificação | Consequência aceite | Rev. |
|---|---------|--------------|---------|--------------|---------------------|------|
| D-1 | Codificação do valor de cada checkbox | Valor composto `app:area_key`, para não assumir que `area_key` é globalmente único entre aplicações | Composto `app:area_key` | - | - | two-way |
| D-2 | Validação da área submetida | Validar por lista de permitidos (catálogo ativo) antes de gravar, em vez de confiar no par app/área vindo do cliente | Validar contra `ListActiveAreas`; rejeitar a submissão inteira com `400` se alguma área não existir | - | - | two-way |
| D-3 | Concessão repetida na mesma submissão ou em submissões diferentes | Idempotência por natureza (at-least-once) | Mantém-se o `ON CONFLICT (...) DO NOTHING` já existente em `core/acl.Store.Grant`, sem alteração | - | - | two-way |
| D-4 | Agrupamento das checkboxes | A opção mais simples seria uma lista plana, igual à ordem atual das opções da select | Agrupar por aplicação, com cabeçalho por bloco | Torna mais rápido marcar várias áreas da mesma aplicação de uma vez, que é o motivo direto do pedido | Pequeno aumento de complexidade do template (agrupamento por `GroupLabel`/App antes de desenhar os checkboxes) | two-way |

## 10. Critérios de aceitação

- [ ] (RF-1) Dado um administrador em `/admin/acl`, quando a página carrega, então cada
      formulário de concessão mostra as áreas ativas como checkboxes agrupadas por
      aplicação, com o nome da aplicação como cabeçalho de cada bloco.
- [ ] (RF-2) Dado o formulário "Conceder acesso a um utilizador", quando o administrador
      marca duas áreas de aplicações diferentes e submete, então são criadas duas
      concessões nessa submissão, ambas visíveis na tabela "Concessões atuais".
- [ ] (RF-3) Dado o mesmo formulário, quando o administrador tenta submeter sem marcar
      nenhuma área, então o browser impede a submissão; se o pedido for feito
      diretamente ao endpoint sem nenhuma área marcada, o servidor responde `400` sem
      criar concessões.
- [ ] (RF-4) Dado um pedido `POST /admin/acl/grant` com uma área que não existe no
      catálogo ativo, quando é submetido, então o servidor responde `400` e nenhuma
      concessão do pedido é gravada, mesmo que o pedido incluísse outras áreas válidas.
- [ ] (RF-5) Dado o formulário "Conceder acesso a um role", quando o administrador marca
      três áreas da mesma aplicação e submete, então as três concessões ficam associadas
      ao role escolhido, sem qualquer efeito no formulário de utilizador.
- [ ] (RF-2) Dado um destinatário que já tem uma área concedida, quando essa área volta
      a ser marcada junto com uma área nova na mesma submissão, então a submissão não
      falha, a área nova fica concedida e a repetida não duplica na tabela.

## 11. Plano de verificação

1. `go build ./...` e `go vet ./home/...` - confirmar que o handler e o template
   compilam sem erros.
2. `make run-dev`, autenticar como admin (autologin de dev) e abrir
   `http://localhost:8080/admin/acl`.
3. Confirmar visualmente que cada formulário mostra checkboxes agrupadas por aplicação,
   com o nome da app como cabeçalho de cada bloco.
4. No formulário de utilizador, marcar duas áreas de aplicações diferentes e submeter;
   confirmar na tabela "Concessões atuais" que aparecem as duas linhas novas.
5. Submeter o mesmo formulário sem marcar nenhuma área; confirmar que o browser bloqueia
   o envio. Depois, com
   `curl -i -X POST http://localhost:8080/admin/acl/grant -d "subject_type=user" -d "subject_id=1"`,
   confirmar `HTTP 400` e que a tabela não ganhou nenhuma linha.
6. Com
   `curl -i -X POST http://localhost:8080/admin/acl/grant -d "subject_type=user" -d "subject_id=1" -d "area_key=app-inexistente:chave-falsa"`,
   confirmar `HTTP 400` e que nenhuma concessão foi gravada.
7. Marcar uma área já concedida junto com uma área nova na mesma submissão; confirmar
   que não há erro e que só a área nova aparece pela primeira vez na tabela.
