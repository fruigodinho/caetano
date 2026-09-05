# Requisitos do Projeto: Saldos Esperados

## 1. Visão Geral
O objetivo é desenvolver uma aplicação (CLI e Web) para processamento de balancetes contabilísticos em formato CSV, comparando os saldos das contas com um padrão de "saldos esperados" definido em ficheiro Excel. A aplicação deve identificar discrepâncias e armazenar histórico.

## 2. Arquitetura e Tecnologia

### 2.1. Stack Tecnológico
- **Linguagem**: Go (Golang) 1.23+
- **Base de Dados**: SQLite (com suporte a migração para PostgreSQL se necessário).
- **ORM/Query Builder**: `sqlc` para geração de código SQL type-safe.
- **Migrações**: `golang-migrate` ou similar.
- **Web Framework**: Gin (conforme regras do utilizador).
- **Frontend**: HTML5, CSS3 (Vanilla/Moderno), Templates Go (`html/template`).
- **CLI**: `cobra` ou `flag` standard.
- **Excel**: `excelize`.
- **CSV**: `encoding/csv` (com suporte a ISO-8859-1 se necessário).

### 2.2. Arquitetura (Standard Go Layout)
A estrutura do projeto seguirá o padrão standard do Go:
```
/
├── cmd/
│   ├── cli/            # Entrypoint para a ferramenta de linha de comandos
│   └── server/         # Entrypoint para o servidor web
├── internal/
│   ├── core/           # Entidades de domínio e interfaces (Portas)
│   │   ├── domain/
│   │   └── ports/
│   ├── service/        # Lógica de negócio (Casos de Uso)
│   └── adapter/        # Implementações (Adaptadores)
│       ├── storage/    # Repositório DB (SQLite/Postgres)
│       ├── parser/     # Parsers CSV e Excel
│       └── web/        # Handlers HTTP
├── pkg/                # Código reutilizável (ex: utils)
├── web/
│   ├── templates/      # Templates HTML
│   └── assets/         # CSS/JS
├── data/               # Ficheiros de dados para teste
├── migrations/         # Scripts SQL de migração
└── Makefile            # Automação de tarefas
```

## 3. Funcionalidades

### 3.1. Core (Lógica de Negócio)
- **Importação de Saldos Esperados (Excel)**:
    - Ler ficheiro `saldos_esperados.xlsx`.
    - Mapear contas (incluindo contas "mãe") para o tipo de saldo esperado (Devedor/Credor ou Positivo/Negativo).
- **Processamento de Balancete (CSV)**:
    - Ler ficheiro CSV em lotes (batch size: 100 linhas).
    - Ignorar cabeçalho (dados começam na linha 4).
    - Filtrar apenas contas de 8 dígitos (contas de movimento).
    - Normalizar valores (tratar formatação europeia `.` milhar, `,` decimal).
- **Validação de Saldos**:
    - Para cada conta de 8 dígitos, verificar a regra na conta correspondente ou na sua conta mãe mais próxima definida nos "Saldos Esperados".
    - Determinar se o saldo está "Correto" ou "Incorreto".
- **Persistência**:
    - Guardar registo do upload (Data, Nome Ficheiro).
    - Guardar linhas processadas e resultado da validação.

### 3.2. Interface de Linha de Comandos (CLI)
- Comando para importar configurações (Saldos Esperados).
- Comando para processar um ficheiro CSV e apresentar relatório no terminal.
- Gerido via `Makefile` (ex: `make run-cli-import`, `make run-cli-process`).

### 3.3. Interface Web
- **Autenticação**:
    - Login (Microsoft OAuth ou Local com 2FA).
    - Middleware de proteção de rotas.
- **Dashboard**:
    - Visão geral dos últimos processamentos.
- **Upload**:
    - Formulário para envio do ficheiro CSV.
    - Feedback de progresso/sucesso.
- **Resultados**:
    - Tabela com contas processadas.
    - Filtros (Apenas Incorretos, Por Conta, etc.).
    - Exportação para CSV.
- **Histórico**:
    - Lista de uploads anteriores e consulta de detalhes.

## 4. Plano de Atividades (Fases)

### Fase 1: Fundações e CLI (Foco Atual)
1.  **Setup**: Inicializar projeto, `go.mod`, estrutura de pastas.
2.  **Base de Dados**: Criar schema SQL, configurar `sqlc` e migrações.
3.  **Domínio**: Definir structs (`Account`, `BalanceRule`, `ProcessingResult`).
4.  **Parsers**:
    - Implementar leitura de Excel (`saldos_esperados.xlsx`).
    - Implementar leitura de CSV (`Balancete.csv`) com tratamento de encoding e formatação numérica.
5.  **Lógica**: Implementar algoritmo de verificação (match conta 8 dígitos -> regra conta mãe).
6.  **CLI**: Criar comandos para executar a importação e processamento.
7.  **Testes**: Validar com os ficheiros na pasta `data`.

### Fase 2: Interface Web
1.  **Servidor**: Configurar Gin, Router e Middleware.
2.  **Templates**: Criar layout base e páginas (Upload, Lista).
3.  **Handlers**: Conectar HTTP à lógica de negócio.
4.  **Autenticação**: Implementar Login.
5.  **UI/UX**: Estilizar com CSS moderno.

### Fase 3: Refinamento e Entrega
1.  **Exportação**: Funcionalidade de exportar resultados.
2.  **Otimização**: Rever performance e logs.
3.  **Documentação**: Atualizar README e instruções de uso.
