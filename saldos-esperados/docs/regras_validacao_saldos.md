# Regras de Validação de Saldos de Contas

## Visão Geral

Este documento descreve o processo completo de validação de saldos de contas contabilísticas, desde a importação de regras até à determinação se um registo está correto ou incorreto.

## 1. Fonte de Regras: Ficheiro Excel

### 1.1. Estrutura do Ficheiro de Regras (`saldos_esperados.xlsx`)

**Localização**: `data/saldos_esperados.xlsx`

**Estrutura**:
```
| Número Conta | Nome Conta          | Tipo Esperado |
|--------------|---------------------|---------------|
| 1            | Ativo               | Devedor       |
| 11           | Disponibilidades    | Devedor       |
| 12345678     | Conta Específica    | Credor        |
```

**Colunas**:
1. **Número Conta** (string): Código da conta (pode ter qualquer comprimento)
2. **Nome Conta** (string): Designação da conta
3. **Tipo Esperado** (string): Tipo de saldo esperado para a conta

### 1.2. Tipos de Saldo Aceites

O sistema utiliza **códigos específicos** no campo `expected_balance_type`:

| Código     | Descrição                                  | Regra de Validação           |
|-------------|----------------------------------------------|---------------------------------|
| **D**       | Devedor                                      | Saldo deve ser ≥ 0 (positivo) |
| **C**       | Credor                                       | Saldo deve ser ≤ 0 (negativo) |
| **S2C**     | Saldo Devedor ou Credor                      | Qualquer valor é válido       |
| **(R2) S2C**| Saldo Devedor ou Credor em 2 campos (Rubrica 2) | Qualquer valor é válido    |

**Notas Importantes**:
- **S2C** e **(R2) S2C**: Contas que podem ter saldo positivo OU negativo legitimamente (ex: Clientes, Fornecedores com adiantamentos)
- Sistema preserva *case* exato dos códigos ("D", "C", não "d", "c")
- Tipos legados ("Devedor", "Credor", "Positivo", "Negativo") ainda suportados como fallback

### 1.3. Importação de Regras

**Comando CLI**:
```bash
./bin/saldos-esperados import-rules -f data/saldos_esperados.xlsx
```

**Processo**:
1. Abre ficheiro Excel
2. Lê primeira sheet
3. Ignora linha 1 (cabeçalho)
4. Para cada linha válida (com número de conta):
   - Extrai `AccountNumber`, `AccountName`, `Type`
   - Valida que número de conta não está vazio
   - Insere na tabela `expected_balances`
5. **Comportamento**: Limpa todas as regras existentes antes de importar (substitui completamente)

**Base de Dados**: Tabela `expected_balances`
```sql
CREATE TABLE expected_balances (
    id SERIAL PRIMARY KEY,
    account_number VARCHAR(20) NOT NULL UNIQUE,
    account_name VARCHAR(255) NOT NULL,
    expected_balance_type VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

## 2. Fonte de Dados: Ficheiro CSV Balancete

### 2.1. Estrutura do Ficheiro CSV

**Características**:
- **Encoding**: ISO-8859-1 (Latin-1)
- **Separador**: `;` (ponto e vírgula)
- **Formato numérico**: Europeu (ex: `1.234,56`)
- **Metadados**: 3 linhas iniciais (ignoradas)
- **Cabeçalho**: Linha 4

**Estrutura**:
```
Linha 1: Metadado 1
Linha 2: Metadado 2
Linha 3: Metadado 3
Linha 4: NumConta;Conta;Saldo
Linha 5+: Dados das contas
```

**Exemplo Real**:
```csv
NumConta;Conta;Saldo
11;Caixa;1.234.567,89
12;Depósitos à ordem;-567.890,12
21111234;Clientes gerais;12.345,67
```

### 2.2. Processamento do CSV

**Comando CLI**:
```bash
./bin/saldos-esperados process-file -f data/Balancete.csv
```

**Passos de Processamento**:

1. **Abertura do Ficheiro**
   - Aplica decoder ISO-8859-1
   - Configura separador `;`
   - Ativa `LazyQuotes` para parsing flexível

2. **Ignorar Metadados**
   - Salta linhas 1-3 (metadados)
   - Lê linha 4 (cabeçalho) mas não processa

3. **Leitura de Dados** (Linha 5 em diante)
   - Mapeia colunas: `[0]=NumConta, [1]=Conta, [2]=Saldo`
   - Valida que linha tem pelo menos 3 colunas

4. **Normalização de Valores**
   ```go
   // Formato Europeu: "1.234,56" -> 1234.56
   balanceStr = strings.ReplaceAll(balanceStr, ".", "")  // Remove milhares
   balanceStr = strings.ReplaceAll(balanceStr, ",", ".") // Vírgula vira ponto
   balance, _ = strconv.ParseFloat(balanceStr, 64)
   ```

5. **Filtro de Contas de Movimento**
   ```go
   // Apenas contas com exatamente 8 dígitos são processadas
   if len(accNum) == 8 {
       // Processar esta conta
   }
   ```
   **Razão**: Contas de movimento têm sempre 8 dígitos. Contas agregadas (1, 11, 111, etc.) não são validadas diretamente.

6. **Processamento em Lotes (Batching)**
   - Tamanho do lote: **100 registos**
   - Acumula registos em memória
   - Quando atinge 100, processa o lote
   - Processa registos remanescentes no final

## 3. Algoritmo de Validação

### 3.1. Busca de Regra Aplicável

Para cada conta de 8 dígitos do CSV, o sistema procura uma regra aplicável usando **algoritmo de fallback hierárquico**:

**Função**: `findRule(accNum string, rules map[string]ExpectedBalance)`

**Algoritmo**:
```
1. Tentar match EXATO com número da conta
   - Se encontrado: usar essa regra

2. Se não encontrado, tentar contas "mãe" (parent accounts)
   - Remover último dígito da direita
   - Procurar regra para esse número
   - Repetir até encontrar ou esgotar dígitos

3. Se nenhuma regra encontrada: conta sem regra
```

**Exemplo Prático**:

Para conta `12345678`:
```
Procura 1: 12345678 (match exato)      ❌ Não existe
Procura 2: 1234567  (7 dígitos)        ❌ Não existe
Procura 3: 123456   (6 dígitos)        ❌ Não existe
Procura 4: 12345    (5 dígitos)        ❌ Não existe
Procura 5: 1234     (4 dígitos)        ✅ EXISTE! Usar regra da conta 1234
```

**Código**:
```go
func (v *Validator) findRule(accNum string, rules map[string]ExpectedBalance) (*ExpectedBalance, string) {
    // Tentativa 1: Match exato
    if rule, ok := rules[accNum]; ok {
        return &rule, accNum
    }

    // Tentativa 2: Parent accounts (remover dígitos da direita)
    for i := len(accNum) - 1; i > 0; i-- {
        parentNum := accNum[:i]
        if rule, ok := rules[parentNum]; ok {
            return &rule, parentNum  // Retorna regra e número da conta mãe
        }
    }

    return nil, ""  // Nenhuma regra encontrada
}
```

### 3.2. Determinação: Correto ou Incorreto

**Função**: `validateBalance(balance float64, expectedType string) bool`

**Lógica de Validação**:

```go
switch expectedType {
    case "D":
        // Devedor: saldo deve ser positivo ou zero
        return balance >= 0
    
    case "C":
        // Credor: saldo deve ser negativo ou zero
        return balance <= 0
    
    case "S2C", "(R2) S2C":
        // Saldo Devedor ou Credor: qualquer valor é válido
        // Estas contas podem ter saldo positivo ou negativo sem erro
        return true
    
    default:
        // Fallback para tipos legados ou desconhecidos
        log.Printf("Unknown expected type: %s", expectedType)
        return true  // Assume válido para evitar falsos negativos
}
```

**Casos Especiais**:

1. **Saldo = 0**
   - ✅ Válido para **ambos** "Devedor" e "Credor"
   - Razão: Zero tecnicamente satisfaz `>= 0` e `<= 0`

2. **Conta sem regra**
   - ✅ Marca como `isCorrect = true`
   - `expected_rule_account = NULL`
   - Razão: Não podemos validar o que não tem regra definida

3. **Tipo desconhecido**
   - ✅ Marca como `isCorrect = true`
   - Log de warning emitido
   - Razão: Evitar falsos negativos por erros de dados

### 3.3. Tabela de Decisão Completa

| Tipo Esperado | Saldo Conta | Resultado | Razão                              |
|---------------|-------------|-----------|-------------------------------------|
| **D**         | +1.234,56   | ✅ OK     | Positivo conforme esperado          |
| **D**         | 0,00        | ✅ OK     | Zero é válido para devedor          |
| **D**         | -1.234,56   | ❌ ERRO   | Negativo quando devia ser positivo  |
| **C**         | -1.234,56   | ✅ OK     | Negativo conforme esperado          |
| **C**         | 0,00        | ✅ OK     | Zero é válido para credor           |
| **C**         | +1.234,56   | ❌ ERRO   | Positivo quando devia ser negativo  |
| **S2C**       | +1.234,56   | ✅ OK     | Devedor aceite para conta flexível  |
| **S2C**       | -1.234,56   | ✅ OK     | Credor aceite para conta flexível   |
| **S2C**       | 0,00        | ✅ OK     | Zero sempre aceite                  |
| **(R2) S2C**  | +1.234,56   | ✅ OK     | Devedor aceite (rubrica 2)          |
| **(R2) S2C**  | -1.234,56   | ✅ OK     | Credor aceite (rubrica 2)           |
| **(R2) S2C**  | 0,00        | ✅ OK     | Zero sempre aceite                  |
| (sem regra)   | qualquer    | ✅ OK*    | Não há regra para validar           |
| (desconhecido)| qualquer    | ✅ OK*    | Tipo inválido, assume correto       |

\* *Marcado como OK mas sem validação efetiva*

**Casos de Uso Reais**:
- **Conta 11 (Caixa)**: Tipo "D" → Deve ter saldo ≥ 0 sempre
- **Conta 12 (Depósitos à ordem)**: Tipo "S2C" → Pode ser positivo (dinheiro no banco) ou negativo (descoberto bancário)
- **Conta 2111 (Clientes c/c)**: Tipo "(R2) S2C" → Pode ser positivo (cliente deve) ou negativo (adiantamentos)

## 4. Persistência de Resultados

### 4.1. Tabela `uploads`

Regista cada processamento de ficheiro:

```sql
CREATE TABLE uploads (
    id SERIAL PRIMARY KEY,
    filename VARCHAR(255) NOT NULL,
    upload_date TIMESTAMP NOT NULL DEFAULT NOW(),
    status VARCHAR(50) NOT NULL DEFAULT 'pending'  -- 'pending', 'processing', 'processed', 'failed'
);
```

### 4.2. Tabela `processed_records`

Regista cada linha processada:

```sql
CREATE TABLE processed_records (
    id SERIAL PRIMARY KEY,
    upload_id INTEGER NOT NULL REFERENCES uploads(id) ON DELETE CASCADE,
    account_number VARCHAR(20) NOT NULL,
    account_name VARCHAR(255) NOT NULL,
    balance DECIMAL(15, 2) NOT NULL,
    expected_rule_account VARCHAR(20),  -- Número da conta que forneceu a regra (pode ser conta mãe)
    is_correct BOOLEAN NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

**Campos Importantes**:
- `expected_rule_account`: Guarda qual conta foi usada para validação
  - Se `12345678` foi validado usando regra da conta `1234`, este campo = `"1234"`
  - Se nenhuma regra encontrada, este campo = `NULL`
- `is_correct`: Resultado da validação (`true` ou `false`)

### 4.3. Fluxo de Persistência

```
1. Criar registo em uploads (status='processing')
   ↓
2. Para cada lote de 100 contas:
   a. Carregar todas as regras em memória (cache)
   b. Para cada conta de 8 dígitos:
      - Buscar regra aplicável (hierarchical fallback)
      - Validar saldo contra regra
      - Inserir em processed_records
   ↓
3. Atualizar uploads (status='processed')
```

## 5. Exemplos Práticos

### Exemplo 1: Validação com Match Exato

**Regra Importada**:
```
Conta: 12345678
Tipo: Devedor
```

**Linha do CSV**:
```
12345678;Caixa Principal;5.432,10
```

**Processo**:
1. Parser lê: `AccountNumber="12345678"`, `Balance=5432.10`
2. Filtro: 8 dígitos ✅ → processar
3. Busca regra: Match exato encontrado (`12345678` existe)
4. Validação: Tipo="Devedor" → requer `balance >= 0`
5. Verificação: `5432.10 >= 0` ✅ **CORRETO**

**Resultado na BD**:
```
account_number: "12345678"
balance: 5432.10
expected_rule_account: "12345678"
is_correct: true
```

### Exemplo 2: Validação com Conta Mãe (Fallback)

**Regras Importadas**:
```
Conta: 11
Tipo: Devedor
```

**Linha do CSV**:
```
11234567;Depósitos Ordem Banco A;-234,56
```

**Processo**:
1. Parser lê: `AccountNumber="11234567"`, `Balance=-234.56`
2. Filtro: 8 dígitos ✅ → processar
3. Busca regra:
   - `11234567` não existe
   - `1123456` não existe
   - `112345` não existe
   - `11234` não existe
   - `1123` não existe
   - `112` não existe
   - `11` **existe!** ✅
4. Validação: Tipo="Devedor" → requer `balance >= 0`
5. Verificação: `-234.56 >= 0` ❌ **INCORRETO**

**Resultado na BD**:
```
account_number: "11234567"
balance: -234.56
expected_rule_account: "11"  ← Usou regra da conta mãe
is_correct: false
```

### Exemplo 3: Conta Sem Regra

**Regras Importadas**:
```
Conta: 1
Tipo: Devedor
```

**Linha do CSV**:
```
98765432;Conta Não Mapeada;1.234,56
```

**Processo**:
1. Parser lê: `AccountNumber="98765432"`, `Balance=1234.56`
2. Filtro: 8 dígitos ✅ → processar
3. Busca regra:
   - `98765432` não existe
   - `9876543` não existe
   - ... (continua até `9`)
   - `9` não existe
   - Nenhuma regra encontrada
4. **Comportamento**: Marca como correto (sem validação)

**Resultado na BD**:
```
account_number: "98765432"
balance: 1234.56
expected_rule_account: NULL  ← Sem regra aplicável
is_correct: true  ← Assume correto por ausência de regra
```

### Exemplo 4: Saldo Zero (Edge Case)

**Regra Importada**:
```
Conta: 22
Tipo: C
```

**Linha do CSV**:
```
22123456;Fornecedores Diversos;0,00
```

**Processo**:
1. Parser lê: `AccountNumber="22123456"`, `Balance=0.00`
2. Busca regra: Fallback encontra `22`
3. Validação: Tipo="C" → requer `balance <= 0`
4. Verificação: `0.00 <= 0` ✅ **CORRETO**

**Resultado**: Zero satisfaz ambos D e C!

### Exemplo 5: Conta com Saldo Flexível (S2C) - Positivo

**Regra Importada**:
```
Conta: 12
Nome: Depósitos à ordem
Tipo: S2C
```

**Linha do CSV**:
```
12345678;Depósito Banco A;45.678,90
```

**Processo**:
1. Parser lê: `AccountNumber="12345678"`, `Balance=45678.90`
2. Filtro: 8 dígitos ✅ → processar
3. Busca regra: Fallback encontra conta `12`
4. Validação: Tipo="S2C" → qualquer valor é válido
5. Verificação: Sempre retorna `true` ✅ **CORRETO**

**Resultado na BD**:
```
account_number: "12345678"
balance: 45678.90
expected_rule_account: "12"
is_correct: true
```

**Interpretação**: Saldo positivo = Dinheiro no banco (normal)

### Exemplo 6: Conta com Saldo Flexível (S2C) - Negativo

**Regra Importada**:
```
Conta: 12
Nome: Depósitos à ordem
Tipo: S2C
```

**Linha do CSV**:
```
12345678;Depósito Banco A;-12.345,67
```

**Processo**:
1. Parser lê: `AccountNumber="12345678"`, `Balance=-12345.67`
2. Filtro: 8 dígitos ✅ → processar
3. Busca regra: Fallback encontra conta `12`
4. Validação: Tipo="S2C" → qualquer valor é válido
5. Verificação: Sempre retorna `true` ✅ **CORRETO**

**Resultado na BD**:
```
account_number: "12345678"
balance: -12345.67
expected_rule_account: "12"
is_correct: true
```

**Interpretação**: Saldo negativo = Descoberto bancário (também válido)

### Exemplo 7: Cliente com Adiantamento (R2) S2C

**Regra Importada**:
```
Conta: 2111
Nome: Clientes - Clientes c/c - Clientes gerais
Tipo: (R2) S2C
```

**Linha do CSV**:
```
21111234;Cliente XYZ Lda;-5.432,10
```

**Processo**:
1. Parser lê: `AccountNumber="21111234"`, `Balance=-5432.10`
2. Filtro: 8 dígitos ✅ → processar
3. Busca regra:
   - `21111234` não existe
   - `2111123` não existe
   - ... (fallback continua)
   - `2111` **existe!** ✅
4. Validação: Tipo="(R2) S2C" → qualquer valor é válido
5. Verificação: Sempre retorna `true` ✅ **CORRETO**

**Resultado na BD**:
```
account_number: "21111234"
balance: -5432.10
expected_rule_account: "2111"
is_correct: true
```

**Interpretação**: Saldo negativo em conta de clientes = Adiantamentos recebidos (legítimo)

## 6. Otimizações e Performance

### 6.1. Caching de Regras

```go
// Carrega TODAS as regras uma vez antes de processar lote
rulesDB, _ := v.queries.ListExpectedBalances(ctx)
rulesMap := make(map[string]ExpectedBalance)
for _, r := range rulesDB {
    rulesMap[r.AccountNumber] = r
}
```

**Benefício**: Evita N queries à base de dados (1 por conta)

### 6.2. Processamento em Lotes

- Lote padrão: **100 registos**
- Transação por lote (atomic)
- Commit apenas no final do lote

### 6.3. Filtro Pré-Processamento

Apenas contas de 8 dígitos são enviadas para validação:
```go
if len(accNum) == 8 {
    batch.append(entry)
}
```

**Benefício**: Reduz volume de dados processados em ~70-80%

## 7. Tratamento de Erros

| Situação                  | Comportamento                          |
|---------------------------|----------------------------------------|
| Ficheiro CSV não encontrado| Retorna erro, não processa            |
| Linha CSV inválida        | Skip linha, continua processamento     |
| Balance não numérico      | Skip linha, continua processamento     |
| Erro inserção BD          | Rollback transação, retorna erro       |
| Tipo regra desconhecido   | Log warning, marca como correto        |

## 8. Limitações e Considerações

### 8.1. Comportamentos Especiais

1. **Saldo Zero**: Validado como correto para tipos **D**, **C**, **S2C** e **(R2) S2C**
   - Razão: `0 >= 0` (D) e `0 <= 0` (C) são ambos verdadeiros
   
2. **Contas sem regra**: Sempre marcadas como corretas
   - Campo `expected_rule_account` = NULL
   - Razão: Não podemos validar o que não tem regra
   
3. **Tipo desconhecido**: Assume correto e emite log warning
   - Evita falsos negativos por erros de dados
   
4. **Case sensitivity**: Códigos principais ("D", "C") são case-sensitive
   - Tipos legados suportam case-insensitive como fallback

### 8.2. Diferença entre S2C e (R2) S2C

**Na prática, ambos funcionam identicamente no sistema atual**:
- Ambos permitem saldo positivo OU negativo
- Ambos sempre retornam `is_correct = true`

**Diferença conceitual (não implementada)**:
- **(R2) S2C**: Indica que a conta pode precisar ser reportada em duas rubricas diferentes no mapa oficial (Rubrica 2)
- **S2C**: Saldo flexível sem necessidade de dupla reportação

Futura implementação pode adicionar lógica específica para (R2) S2C se necessário.

## 9. Queries Úteis para Análise

### Contas Incorretas
```sql
SELECT account_number, account_name, balance, expected_rule_account
FROM processed_records
WHERE is_correct = false
ORDER BY account_number;
```

### Contas Sem Regra
```sql
SELECT account_number, account_name, balance
FROM processed_records
WHERE expected_rule_account IS NULL
ORDER BY account_number;
```

### Taxa de Erro por Upload
```sql
SELECT 
    u.filename,
    COUNT(*) as total,
    SUM(CASE WHEN is_correct = false THEN 1 ELSE 0 END) as incorretos,
    ROUND(100.0 * SUM(CASE WHEN is_correct = false THEN 1 ELSE 0 END) / COUNT(*), 2) as taxa_erro
FROM processed_records pr
JOIN uploads u ON pr.upload_id = u.id
GROUP BY u.id, u.filename;
```

## 10. Conclusão

O sistema implementa uma validação **robusta e tolerante a falhas**, priorizando:
- **Hierarquia de contas**: Fallback automático para contas mãe
- **Flexibilidade**: Aceita múltiplos formatos de tipo (Devedor/Positivo/Credor/Negativo)
- **Performance**: Caching e processamento em lotes
- **Rastreabilidade**: Guarda qual regra foi aplicada a cada validação

Esta abordagem minimiza falsos negativos (contas marcadas incorretamente como erradas) ao assumir que ausência de regra = correto, delegando a decisão final ao utilizador durante análise dos resultados.
