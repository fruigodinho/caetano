# Correção: Compatibilidade POSIX no Makefile

**Data**: 2025-11-26  
**Tipo**: Bug Fix  
**Componente**: Makefile (comandos de migração)

## Problema Identificado

### Sintoma
Os comandos `make migrate-down` e `make db-reset` falhavam com os seguintes erros:

```bash
$ make migrate-down
⚠️  WARNING: This will DROP ALL TABLES!
/bin/sh: 1: read: Illegal option -n
/bin/sh: 3: [[: not found
Cancelled.
```

### Causa Raiz
O Makefile por defeito usa `/bin/sh` como interpretador, que no Pop!_OS (e em muitas distribuições Linux modernas) está vinculado ao **dash** — um shell POSIX minimalista.

O código de confirmação interativa usava sintaxe específica do **bash**:
- `read -n 1 -r` → `-n` não existe no POSIX
- `[[ $REPLY =~ ^[Yy]$ ]]` → `[[` é extensão bash, POSIX usa `[`

```makefile
# Código problemático
migrate-down: tunnel-start
	@echo "⚠️  WARNING: This will DROP ALL TABLES!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \        # ❌ -n não POSIX
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \               # ❌ [[ não POSIX
		echo "Dropping all tables..."; \
		go run cmd/migrate/main.go -dir=down; \
	else \
		echo "Cancelled."; \
	fi
```

## Solução Implementada

### Alteração no Makefile

Adicionada diretiva para forçar o uso do **bash** como shell do Makefile:

```makefile
.PHONY: help build run-cli run-web test clean lint fmt sqlc migrate migrate-up migrate-down db-reset tunnel-start tunnel-stop tunnel-status

# Force bash as shell (needed for read -n and [[]])
SHELL := /bin/bash                                    # ✅ NOVO

# Variables
BINARY_NAME=saldos-esperados
BINARY_PATH=bin/$(BINARY_NAME)
MAIN_PATH=cmd/cli/main.go
```

### Por que Bash?

1. **Consistência**: O código de confirmação já usava features bash
2. **Simplicidade**: Uma linha resolve o problema sem reescrever lógica
3. **Compatibilidade**: Bash está presente em ~100% das distros Linux
4. **Manutenibilidade**: Mantém código legível (`[[ ]]` é mais intuitivo que `[ ]`)

### Alternativa POSIX (não implementada)

Se quiséssemos manter `/bin/sh`, teríamos que reescrever:

```makefile
# Versão POSIX (mais verbosa)
migrate-down: tunnel-start
	@echo "⚠️  WARNING: This will DROP ALL TABLES!"
	@printf "Are you sure? [y/N] "; \
	read REPLY; \                                      # Sem -n
	echo ""; \
	case "$$REPLY" in \                                # case em vez de [[
		[Yy]|[Yy][Ee][Ss]) \
			echo "Dropping all tables..."; \
			go run cmd/migrate/main.go -dir=down ;; \
		*) \
			echo "Cancelled." ;; \
	esac
```

**Desvantagens**:
- Mais verboso e difícil de ler
- Não há confirmação "imediata" (`-n 1` permite responder sem Enter)
- Perde features úteis do bash

## Verificação

### Teste 1: migrate-down com cancelamento
```bash
$ make migrate-down
Starting SSH tunnel for postgres...
postgres tunnel already running (PID: 11607)
⚠️  WARNING: This will DROP ALL TABLES!
Are you sure? [y/N] n
Cancelled.
```
✅ **Resultado**: Funciona corretamente

### Teste 2: db-reset com cancelamento automático
```bash
$ echo "n" | make db-reset
Starting SSH tunnel for postgres...
postgres tunnel already running (PID: 11607)
⚠️  WARNING: This will DROP and RECREATE all tables!

Cancelled.
```
✅ **Resultado**: Funciona corretamente

### Teste 3: Compilação
```bash
$ go build ./cmd/server ./cmd/cli ./cmd/migrate
$ echo $?
0
```
✅ **Resultado**: Sem erros

## Impacto

### Alterações
- ✅ 1 linha adicionada no Makefile
- ✅ Nenhuma alteração de lógica
- ✅ Nenhuma alteração no código Go
- ✅ Comandos funcionam como esperado

### Comandos Afetados
- `make migrate-down` — ✅ Corrigido
- `make db-reset` — ✅ Corrigido
- Todos os outros comandos — ✅ Sem impacto

### Compatibilidade
| Sistema Operacional | Shell Padrão | Compatibilidade |
|---------------------|--------------|-----------------|
| Ubuntu/Pop!_OS      | dash → bash  | ✅ Corrigido     |
| Debian              | dash → bash  | ✅ Corrigido     |
| RHEL/CentOS         | bash         | ✅ Já funcionava |
| Arch Linux          | bash         | ✅ Já funcionava |
| macOS               | zsh/bash     | ✅ Compatível    |

## Lições Aprendidas

1. **Makefiles usam /bin/sh por defeito**: Nunca assumir bash features sem `SHELL := /bin/bash`
2. **Dash vs Bash**: Muitas distros modernas usam dash para `/bin/sh` por performance
3. **Testes em múltiplos ambientes**: Código que funciona no bash pode falhar no dash
4. **Documentação clara**: `read -n` e `[[` são extensões bash, não POSIX

## Referências

- [GNU Make: Choosing the Shell](https://www.gnu.org/software/make/manual/html_node/Choosing-the-Shell.html)
- [Bash vs Dash: Scripting Differences](https://wiki.ubuntu.com/DashAsBinSh)
- [POSIX Shell Command Language](https://pubs.opengroup.org/onlinepubs/9699919799/utilities/V3_chap02.html)

## Ficheiros Modificados

| Ficheiro   | Linhas Alteradas | Tipo         |
|------------|------------------|--------------|
| `Makefile` | +2 (linhas 3-4)  | Adição       |

## Commit Sugerido

```
fix: Forçar bash no Makefile para compatibilidade com confirmações interativas

Os comandos migrate-down e db-reset usam features bash (read -n, [[]])
que não existem em sh POSIX (dash). Forçar SHELL := /bin/bash resolve
o erro "/bin/sh: 1: read: Illegal option -n" em sistemas Pop!_OS/Ubuntu.
```

## Checklist de Validação

- [x] Erro reproduzido e compreendido
- [x] Solução implementada (SHELL := /bin/bash)
- [x] `make migrate-down` testado com cancelamento
- [x] `make db-reset` testado com cancelamento
- [x] Compilação Go verificada
- [x] Documentação criada
- [x] Nenhum impacto em outros comandos

---

**Status**: ✅ Resolvido  
**Impacto**: Baixo (correção de bug de compatibilidade)  
**Risco**: Nenhum (bash disponível em todos os ambientes alvo)
