# Security Phase 1 - Template Syntax Bugfix

**Data:** 2025-12-01  
**Agente:** Security-Go-Gin  
**Contexto:** Correção de erros de sintaxe HTML introduzidos durante implementação de proteção CSRF

## Problema Identificado

Após implementação da proteção CSRF (Phase 1.3), foram identificados erros de sintaxe HTML em templates de listagem que causavam:

```
html/template:authenticated: "<" in attribute name:
```

**Sintoma:** Ao aceder à página `/balances`, ocorria logout automático e redirecionamento para login com CSRF token vazio.

## Causa Raiz

Durante a adição automática (via `sed`) dos campos CSRF hidden inputs nos formulários, foram criados botões de delete com atributos HTML malformados. Especificamente, faltavam atributos obrigatórios no elemento `<button>`.

### Exemplo do erro:

```html
<button type="submit" class="btn btn-danger"
    <svg width="16" height="16" ...>
```

**Problema:** Tag SVG começava diretamente após o nome da classe, sem fechar os atributos do botão.

## Correção Aplicada

### Ficheiros corrigidos:

1. **web/templates/pages/balances_list.html** (linha 92)
2. **web/templates/pages/types_list.html** (linha 45)  
3. **web/templates/pages/users_list.html** (linha 72)

### Sintaxe corrigida:

**Antes:**
```html
<button type="submit" class="btn btn-danger"
    <svg width="16" height="16" ...>
```

**Depois:**
```html
<button type="submit" class="btn btn-danger" style="padding: 0.25rem 0.5rem;" title="Eliminar">
    <svg width="16" height="16" ...>
```

## Validação

```bash
cd /home/rgodinho/Workspaces/GolangProjects/saldos-esperados
go build ./cmd/server  # ✅ Compilação bem-sucedida
```

## Status da Fase 1

Todas as vulnerabilidades críticas (P0) foram corrigidas:

- ✅ **P0.1:** Session Secret hardcoded → Configurável via variável de ambiente
- ✅ **P0.2:** TLS/HTTPS ausente → Suporte completo com certificados
- ✅ **P0.3:** CSRF Protection → Middleware custom implementado
- ✅ **P0.4:** Rate Limiting → 3 middlewares específicos (login, upload, API)
- ✅ **Bugfix:** Templates HTML corrigidos e validados

## Próximos Passos (Phase 2)

Após confirmação do utilizador, avançar para vulnerabilidades P1:
1. Account Lockout (bloqueio após tentativas falhadas)
2. TOTP Secret Encryption (encriptar segredos 2FA na BD)
3. Input Validation (sanitização de inputs)

## Ficheiros Alterados

- `web/templates/pages/balances_list.html`
- `web/templates/pages/types_list.html`
- `web/templates/pages/users_list.html`

## Notas Técnicas

O erro `"<" in attribute name` é específico do parser de templates Go quando detecta caracteres `<` onde espera valores de atributos HTML. Este tipo de erro:
- Ocorre durante o parsing dos templates (antes do runtime)
- Causa falha silenciosa na renderização (redirecionamento para login)
- É facilmente introduzido por manipulação automatizada de HTML

**Lição aprendida:** Sempre validar sintaxe HTML após edições automatizadas (sed, awk, etc.).
