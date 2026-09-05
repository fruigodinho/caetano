# Guia de Uso

## Visão Geral

O **Saldos Esperados** oferece duas interfaces:
- **CLI (Command Line Interface)**: Para operações batch e automação
- **Web Interface**: Para gestão visual e consultas interativas

## Interface de Linha de Comandos (CLI)

### 1. Importar Regras de Validação

Importa tipos de saldo e regras de validação a partir de um ficheiro Excel.

```bash
./bin/cli import-rules --file regras.xlsx
```

**Formato do Excel**:
- **Folha "LEGENDA"**: Tipos de saldo
  - Coluna A: `Code` (ex: D, C, S2C)
  - Coluna B: `Description` (ex: "Devedor")
- **Folha "Saldo Esperados"**: Regras
  - Coluna A: `Account Number` (ex: 2111, 2111*)
  - Coluna B: `Expected Balance Type` (ex: C, Da)

**Output esperado**:
```
Importing balance types from LEGENDA sheet...
Successfully imported 11 balance types.
Importing expected balance rules...
Successfully imported 324 rules.
```

**Notas**:
- Importação é idempotente (re-executar não cria duplicados)
- Suporta wildcards em contas: `2111*` → valida 2111, 21111, 211101, etc.
- Folha LEGENDA é processada primeiro para garantir integridade referencial

### 2. Processar Ficheiro de Balancete

Valida um ficheiro CSV contra as regras importadas.

```bash
./bin/cli process-file --file balancete_2025-01.csv
```

**Com Label Personalizada** (recomendado):
```bash
# Label extraída automaticamente da 1ª linha do CSV
# Se CSV começa com "Balancete Janeiro 2025", usa esse label
./bin/cli process-file --file balancete_2025-01.csv
```

**Output esperado**:
```
Processing file with label: Balancete Janeiro 2025
Batch 1/10 processed (100 records)
Batch 2/10 processed (200 records)
...
File processed successfully.

Summary:
- Total records: 1.234
- Valid: 1.150 (93%)
- Invalid: 84 (7%)
```

**Comportamento**:
- **Batch processing**: Processa em lotes de 100 registos
- **Validação hierárquica**: Se conta 211101 não tem regra, procura 21110, 2111, 211, 21, 2
- **Encoding automático**: Detecta ISO-8859-1 ou UTF-8
- **Label**: Primeira linha do CSV (ex: "Balancete Janeiro 2025")

### 3. Criar Utilizador

Cria um novo utilizador com 2FA.

```bash
./bin/cli create-user <username> <email> <password> <role>
```

**Exemplo**:
```bash
# Admin
./bin/cli create-user admin admin@example.com StrongPass123! admin

# Manager
./bin/cli create-user gestor1 gestor@example.com Pass456! manager

# Operator
./bin/cli create-user op1 op@example.com Pass789! operator
```

**Roles disponíveis**:
- `admin`: Acesso total (gestão users, audit logs, configurações)
- `manager`: Upload, consultas, relatórios
- `operator`: Upload e consultas básicas

**Output esperado**:
```
User admin created successfully.
Configure 2FA:
TOTP Secret: JBSWY3DPEHPK3PXP
QR Code:
████████████████████████████
...
```

**IMPORTANTE**: Guardar TOTP secret em local seguro (necessário para 2FA).

## Interface Web

Aceder em: `http://localhost:8080` (ou porta configurada em `.env`)

### Login e 2FA

1. **Página de Login**
   - Username
   - Password
   - 2FA Code (6 dígitos do Google Authenticator)

2. **Primeiro Login**
   - Escanear QR code gerado no `create-user`
   - Guardar TOTP secret como backup
   - Introduzir código de 6 dígitos

### Dashboard

Após login, o dashboard mostra:
- **Estatísticas Globais**:
  - Total de uploads
  - Registos processados (últimos 30 dias)
  - Taxa de validação média
  - Uploads recentes (lista)

- **Ações Rápidas**:
  - `Novo Upload`: Submeter ficheiro CSV
  - `Ver Histórico`: Listar todos os uploads
  - `Procurar Contas`: Pesquisa avançada

### Upload de Ficheiros

**Passos**:
1. Click em "Novo Upload" ou menu "Upload"
2. Selecionar ficheiro CSV
3. (Opcional) Introduzir label descritiva
4. Click "Submeter"

**Interface mostra**:
- Progress bar de upload
- Validação em tempo real
- Resumo ao concluir:
  - Registos válidos vs inválidos
  - Lista de erros (se existirem)
  - Link para download de relatório

**Formato CSV esperado**:
```csv
Balancete Janeiro 2025
Account,Balance
211101,1500.50
211102,-300.00
...
```

### Consulta de Resultados

**Listar Uploads**:
1. Menu "Histórico"
2. Filtrar por:
   - Data (intervalo)
   - Utilizador
   - Label
3. Click em upload para ver detalhes

**Detalhes de Upload**:
- Ficheiro: nome original
- Label: identificação
- Data/Hora: quando foi processado
- Utilizador: quem fez upload
- Resumo: válidos/inválidos
- **Tabela de Registos**:
  - Account Number
  - Current Balance
  - Expected Type
  - Status (✓ Valid / ✗ Invalid)
  - Error Message (se inválido)

**Exportar Resultados**:
- Botão "Exportar CSV"
- Botão "Exportar Excel"
- Filtros aplicados são mantidos

### Pesquisa Avançada de Contas

**Aceder**: Menu "Procurar Contas"

**Funcionalidades**:
- **Pesquisa simples**: Introduzir número de conta
- **Wildcards**:
  - `2111*`: Todas as contas que começam com 2111
  - `*5`: Todas as contas que terminam com 5
  - `21*1`: Contas com 21...1

**Resultados mostram**:
- Conta encontrada
- Tipo de saldo esperado
- Regra aplicada (directa ou hierárquica)
- Histórico de uploads (se existir)

### Gestão de Utilizadores (Admin)

**Aceder**: Menu "Utilizadores" (visível apenas para admins)

**Listar Utilizadores**:
- Username
- Email
- Role
- Status (Ativo/Inativo)
- Data de criação

**Criar Utilizador**:
1. Click "Novo Utilizador"
2. Preencher formulário:
   - Username
   - Email
   - Password (mínimo 8 caracteres)
   - Role (dropdown)
3. Submeter
4. QR code 2FA gerado automaticamente

**Editar Utilizador**:
- Alterar email
- Alterar role
- Reset password
- Desativar/Ativar conta

**Desativar Utilizador**:
- Não apaga da BD
- Bloqueia login
- Mantém histórico de uploads

### Audit Logs (Admin)

**Aceder**: Menu "Audit Logs"

**Visualizar Logs**:
- **Filtros**:
  - Utilizador (dropdown)
  - Ação (login, logout, upload, create_user, etc.)
  - Intervalo de datas
- **Colunas**:
  - Timestamp
  - Utilizador
  - Ação
  - Recurso (ex: ficheiro.csv, username)
  - Detalhes (JSON com metadados)

**Ações Auditadas**:
- `login`: Autenticação com 2FA
- `logout`: Fim de sessão
- `upload`: Submissão de ficheiro
- `create_user`: Criação de utilizador
- `edit_user`: Alteração de utilizador
- `deactivate_user`: Desativação de conta
- `import_rules`: Importação de regras

**Exportar Audit Logs**:
- Botão "Exportar CSV"
- Filtros aplicados são mantidos
- Útil para compliance e auditorias

### Configurações (Admin)

**Aceder**: Menu "Configurações"

**Opções**:
- **Validação**:
  - Ativar/Desativar fallback hierárquico
  - Batch size (default: 100)
- **Sessões**:
  - Timeout de sessão (minutos)
- **Uploads**:
  - Tamanho máximo de ficheiro (MB)
  - Formatos permitidos (CSV, Excel)

## Casos de Uso Práticos

### Caso 1: Validação Mensal de Balancete

**Objetivo**: Validar balancete de Janeiro 2025

**Passos**:
1. Exportar balancete do sistema contabilístico como CSV
2. Aceder à interface web → "Novo Upload"
3. Selecionar `balancete_2025-01.csv`
4. Introduzir label: "Balancete Janeiro 2025"
5. Submeter
6. Revisar resultados:
   - Se erros: Consultar "Error Message" para corrigir
   - Exportar relatório para partilhar

**Validação Hierárquica**:
- Conta `211101` valida contra regra `2111*`
- Se `2111*` não existir, procura `211*`, `21*`, `2*`

### Caso 2: Importar Novas Regras

**Objetivo**: Atualizar regras de validação após mudança no plano de contas

**Passos**:
1. Preparar Excel com folhas LEGENDA e "Saldo Esperados"
2. Via CLI:
   ```bash
   ./bin/cli import-rules --file regras_2025_v2.xlsx
   ```
3. Verificar output: "Successfully imported X rules"
4. Testar com ficheiro de validação conhecido

**Notas**:
- Importação substitui regras existentes com mesmo `Account Number`
- Tipos de saldo não são apagados (append apenas)

### Caso 3: Auditoria de Utilizador

**Objetivo**: Verificar atividade de um utilizador específico

**Passos** (Admin):
1. Menu "Audit Logs"
2. Filtrar por utilizador: `gestor1`
3. Intervalo: "Últimos 30 dias"
4. Analisar ações:
   - Quantos logins?
   - Quantos uploads?
   - Alguma ação suspeita?
5. Exportar relatório se necessário

### Caso 4: Pesquisa de Conta com Wildcard

**Objetivo**: Verificar regra aplicada a contas 2111xxx

**Passos**:
1. Menu "Procurar Contas"
2. Introduzir: `2111*`
3. Resultados mostram:
   - `2111` → Tipo C (regra directa)
   - `21110` → Tipo C (herdado de 2111*)
   - `211101` → Tipo C (herdado de 2111*)

### Caso 5: Criar Utilizador para Novo Colaborador

**Objetivo**: Dar acesso a novo gestor

**Passos** (Admin via web):
1. Menu "Utilizadores" → "Novo Utilizador"
2. Preencher:
   - Username: `gestor2`
   - Email: `gestor2@empresa.com`
   - Password: (gerar password forte)
   - Role: `manager`
3. Submeter
4. Partilhar com colaborador:
   - Credenciais (via canal seguro)
   - QR code 2FA (screenshot ou TOTP secret)

**Passos** (Admin via CLI):
```bash
./bin/cli create-user gestor2 gestor2@empresa.com Pass!234Secure manager
# Partilhar TOTP secret gerado
```

## Atalhos de Teclado (Web)

- `Ctrl+U`: Novo Upload (qualquer página)
- `Ctrl+S`: Procurar Contas
- `Ctrl+H`: Ver Histórico
- `Esc`: Fechar modal/diálogo

## Limites e Restrições

### Tamanho de Ficheiros
- **CSV**: Até 50 MB (configurável em Settings)
- **Excel**: Até 10 MB

### Batch Processing
- CSVs grandes são processados em lotes de 100 registos
- Progress bar mostra progresso em tempo real

### Sessões
- Timeout: 30 minutos de inatividade (configurável)
- Após timeout: Redirecionado para login

### Rate Limiting
- Max 10 uploads por minuto por utilizador
- Max 100 requisições/min por IP

## Próximos Passos

Para detalhes técnicos:
- [Formato de Ficheiros](formato_ficheiros.md): Especificação completa de CSV/Excel
- [API Reference](api_reference.md): Endpoints REST (se integração)
- [Desenvolvimento](desenvolvimento.md): Contribuir com código

Para operações:
- [Segurança](seguranca.md): Boas práticas de produção
- [Deploy](deploy.md): Configurar em produção
- [Testes](testes.md): Executar suite de testes
