# Tipos de Saldo - Referência Completa

Este documento descreve todos os tipos de saldo suportados pelo sistema **Saldos Esperados**.

## Índice

- [Visão Geral](#visão-geral)
- [Tipos Credores (C)](#tipos-credores-c)
- [Tipos Devedores (D)](#tipos-devedores-d)
- [Tipos Mistos (Saldo C/D)](#tipos-mistos-saldo-cd)
- [Tabela Resumo](#tabela-resumo)
- [Exemplos Práticos](#exemplos-práticos)

## Visão Geral

O sistema suporta **11 tipos de saldo** definidos na taxonomia SVAT (Sistema de Validação Automática de Taxonomias). Cada tipo define:

1. **Código**: Identificador único (ex: "D", "C", "S2C")
2. **Descrição**: Nome legível do tipo
3. **Indicador de Coluna**: 
   - `C` = Credor
   - `D` = Devedor  
   - `2` = Ambos (Credor ou Devedor)
4. **Regra de Validação**: Lógica que determina se um saldo é válido

### Regras de Validação

| Regra | Descrição | Lógica |
|-------|-----------|--------|
| `credit_only` | Apenas saldos credores | balance ≤ 0 |
| `debit_only` | Apenas saldos devedores | balance ≥ 0 |
| `both_separate` | Qualquer saldo, 2 campos | Sempre válido |
| `both_net` | Qualquer saldo, 1 campo | Sempre válido |
| `both_before_transfer` | Qualquer saldo, temporal | Sempre válido |

## Tipos Credores (C)

### C - Credor

**Código**: `C`  
**Indicador**: `C`  
**Regra**: `credit_only`

**Descrição Completa**:
> Esperado saldo Credor nas contas com saldo, APÓS apuramento de resultados

**Validação**: O saldo deve ser **negativo ou zero** (balance ≤ 0)

**Exemplos de Contas**:
- 221 - Fornecedores
- 25 - Financiamentos obtidos
- 26 - Acionistas/sócios
- 27 - Outras contas a pagar
- 51 - Capital

**Exemplo de Validação**:
```go
Conta: 221 (Fornecedores)
Tipo: C (Credor)

Saldo: -1000,00  →  ✅ VÁLIDO (credor)
Saldo:     0,00  →  ✅ VÁLIDO (saldado)
Saldo:  +500,00  →  ❌ INVÁLIDO (deveria ser credor)
```

---

### Ca - Credor Antes de Apuramento

**Código**: `Ca`  
**Indicador**: `C`  
**Regra**: `credit_only`

**Descrição Completa**:
> Esperado saldo Credor nas contas com saldo, ANTES de apuramento de resultados. Após apuramento de resultados, as contas apresentam-se saldadas.

**Validação**: balance ≤ 0

**Contexto Temporal**: Antes do apuramento, estas contas têm saldo credor. Após o apuramento, saldo = 0.

**Exemplos de Contas**:
- 61 - Custo das mercadorias vendidas e das matérias consumidas
- 62 - Fornecimentos e serviços externos
- 71 - Vendas

---

### Cc - Credor Antes de Transferência

**Código**: `Cc`  
**Indicador**: `C`  
**Regra**: `credit_only`

**Descrição Completa**:
> Esperado saldo Credor nas contas com saldo, ANTES de transferência para inventários. ANTES de apuramento de resultados, as contas apresentam-se saldadas. Contas não são representadas diretamente em balanço ou demonstração de resultados.

**Validação**: balance ≤ 0

**Contexto**: Contas transitórias que existem apenas durante processo de transferência.

---

## Tipos Devedores (D)

### D - Devedor

**Código**: `D`  
**Indicador**: `D`  
**Regra**: `debit_only`

**Descrição Completa**:
> Esperado saldo Devedor nas contas com saldo, APÓS apuramento de resultados

**Validação**: O saldo deve ser **positivo ou zero** (balance ≥ 0)

**Exemplos de Contas**:
- 11 - Caixa
- 12 - Depósitos bancários
- 21 - Clientes
- 32 - Mercadorias
- 43 - Ativos fixos tangíveis

**Exemplo de Validação**:
```go
Conta: 11 (Caixa)
Tipo: D (Devedor)

Saldo: +1000,00  →  ✅ VÁLIDO (devedor)
Saldo:     0,00  →  ✅ VÁLIDO (saldado)
Saldo:  -500,00  →  ❌ INVÁLIDO (deveria ser devedor)
```

---

### Da - Devedor Antes de Apuramento

**Código**: `Da`  
**Indicador**: `D`  
**Regra**: `debit_only`

**Descrição Completa**:
> Esperado saldo Devedor nas contas com saldo, ANTES de apuramento de resultados. Após apuramento de resultados, as contas apresentam-se saldadas.

**Validação**: balance ≥ 0

**Contexto Temporal**: Antes do apuramento, saldo devedor. Após apuramento, saldo = 0.

**Exemplos de Contas**:
- Contas de gastos antes de transferência para resultados

---

### Dc - Devedor Antes de Transferência

**Código**: `Dc`  
**Indicador**: `D`  
**Regra**: `debit_only`

**Descrição Completa**:
> Esperado saldo Devedor nas contas com saldo, ANTES de transferência para inventários. ANTES de apuramento de resultados, as contas apresentam-se saldadas. Contas não são representadas diretamente em balanço ou demonstração de resultados.

**Validação**: balance ≥ 0

**Contexto**: Contas transitórias no processo de inventário.

---

## Tipos Mistos (Saldo C/D)

Estes tipos permitem **qualquer saldo** (positivo ou negativo), diferindo apenas na forma de representação.

### S2C - Saldo Devedor ou Credor em 2 Campos

**Código**: `S2C`  
**Indicador**: `2`  
**Regra**: `both_separate`

**Descrição Completa**:
> Esperado saldo Credor ou Devedor nas contas com saldo, APÓS apuramento de resultados.
> Contas com saldo devedor somadas para campo DÉBITO.
> Contas com saldo credor somadas para campo CRÉDITO.

**Validação**: **Sempre válido** (qualquer valor permitido)

**Representação**: 
- Saldos devedores (+) → Coluna DÉBITO
- Saldos credores (-) → Coluna CRÉDITO

**Exemplos de Contas**:
- 268 - Outros devedores e credores
- 278 - Outros devedores e credores (sócios)

**Exemplo**:
```
Conta 268:
Empresa A: Saldo +100  →  Débito: 100, Crédito: 0
Empresa B: Saldo -50   →  Débito: 0,   Crédito: 50
Total:                    Débito: 100, Crédito: 50
```

---

### (R2) S2C - S2C Versão R2

**Código**: `(R2) S2C`  
**Indicador**: `2`  
**Regra**: `both_separate`

**Descrição Completa**:
> Nestas taxonomias o saldo esperado foi ajustado da 1ª para a 2ª versão do ficheiro SVAT.
> Esperado saldo Credor ou Devedor nas contas com saldo, APÓS apuramento de resultados.
> Contas com saldo devedor somadas para campo DÉBITO.
> Contas com saldo credor somadas para campo CRÉDITO.

**Validação**: **Sempre válido**

**Diferença**: Ajuste técnico da versão 1 para versão 2 da taxonomia SVAT. Comportamento idêntico a S2C.

---

### S1C - Saldo Devedor ou Credor em 1 Campo

**Código**: `S1C`  
**Indicador**: `2`  
**Regra**: `both_net`

**Descrição Completa**:
> Esperado saldo Credor ou Devedor nas contas com saldo, APÓS apuramento de resultados.
> Se representadas em campo CRÉDITO, somam-se contas credoras e subtraem-se as devedoras.
> Se representadas em campo DÉBITO, somam-se contas devedoras e subtraem-se as credoras.

**Validação**: **Sempre válido**

**Representação**: 
- Valor líquido (soma algébrica) num único campo
- Se representado em CRÉDITO: credoras (+) e devedoras (-)
- Se representado em DÉBITO: devedoras (+) e credoras (-)

**Exemplo**:
```
Contas S1C:
Conta A: +100 (devedor)
Conta B: -50  (credor)

Representação em CRÉDITO: -50 + 100 = +50 (líquido devedor)
Representação em DÉBITO:  100 - 50  = +50 (líquido devedor)
```

---

### Sa1C - Saldo C/D Antes, em 1 Campo

**Código**: `Sa1C`  
**Indicador**: `2`  
**Regra**: `both_net`

**Descrição Completa**:
> Esperado saldo Credor ou Devedor nas contas com saldo, ANTES de apuramento de resultados.
> Se representadas em campo CRÉDITO, somam-se contas credoras e subtraem-se as devedoras.
> Se representadas em campo DÉBITO, somam-se contas devedoras e subtraem-se as credoras.

**Validação**: **Sempre válido**

**Contexto Temporal**: Antes do apuramento de resultados.

**Representação**: Igual a S1C, mas no contexto temporal antes do apuramento.

---

### Sc - Saldo C/D Antes de Transferências

**Código**: `Sc`  
**Indicador**: `2`  
**Regra**: `both_before_transfer`

**Descrição Completa**:
> Esperado saldo Devedor ou Credor nas contas com saldo, ANTES de transferência para inventários|rendimentos|gastos. ANTES de apuramento de resultados, as contas apresentam-se saldadas. Contas não são representadas diretamente em balanço ou demonstração de resultados.

**Validação**: **Sempre válido**

**Contexto**: Contas transitórias no processo de transferência. Saldadas antes do apuramento final.

---

## Tabela Resumo

| Código | Nome | Indicador | Regra | Validação | Temporal |
|--------|------|-----------|-------|-----------|----------|
| **C** | Credor | C | `credit_only` | ≤ 0 | Após apuramento |
| **Ca** | Credor antes | C | `credit_only` | ≤ 0 | Antes apuramento |
| **Cc** | Credor antes transf. | C | `credit_only` | ≤ 0 | Antes transferência |
| **D** | Devedor | D | `debit_only` | ≥ 0 | Após apuramento |
| **Da** | Devedor antes | D | `debit_only` | ≥ 0 | Antes apuramento |
| **Dc** | Devedor antes transf. | D | `debit_only` | ≥ 0 | Antes transferência |
| **S2C** | Saldo C/D 2 campos | 2 | `both_separate` | Qualquer | Após apuramento |
| **(R2) S2C** | S2C versão R2 | 2 | `both_separate` | Qualquer | Após apuramento |
| **S1C** | Saldo C/D 1 campo | 2 | `both_net` | Qualquer | Após apuramento |
| **Sa1C** | Saldo C/D antes, 1 campo | 2 | `both_net` | Qualquer | Antes apuramento |
| **Sc** | Saldo C/D antes transf. | 2 | `both_before_transfer` | Qualquer | Antes transferência |

## Exemplos Práticos

### Exemplo 1: Validação de Caixa

```
Conta: 11 - Caixa
Regra: D (Devedor)

Balancete recebido:
11;Caixa;1234,56

Validação:
- Saldo: +1234,56
- Tipo esperado: D (≥ 0)
- Resultado: ✅ VÁLIDO
```

### Exemplo 2: Erro em Fornecedores

```
Conta: 221 - Fornecedores
Regra: C (Credor)

Balancete recebido:
221;Fornecedores;500,00

Validação:
- Saldo: +500,00
- Tipo esperado: C (≤ 0)
- Resultado: ❌ INVÁLIDO
- Erro: Fornecedores deve ter saldo credor (negativo)
```

### Exemplo 3: Conta com Saldo Misto

```
Conta: 268 - Outros devedores e credores
Regra: S2C (qualquer saldo, 2 campos)

Balancete recebido:
268;Outros devedores e credores;-150,00

Validação:
- Saldo: -150,00
- Tipo esperado: S2C (qualquer)
- Resultado: ✅ VÁLIDO
- Nota: Saldo credor é aceitável para este tipo
```

### Exemplo 4: Validação Hierárquica

```
Conta no Balancete: 211101 (não tem regra específica)

Validação hierárquica:
1. Procura regra para 211101 → Não encontrada
2. Procura regra para 21110  → Não encontrada
3. Procura regra para 2111   → Não encontrada
4. Procura regra para 211    → Não encontrada
5. Procura regra para 21     → ✅ Encontrada!
   Regra: 21 - Clientes - Tipo D (Devedor)

Aplica regra do pai (21) à conta filha (211101)
```

## Perguntas Frequentes

### 1. Por que existem variantes "antes de apuramento"?

As variantes `Ca`, `Da`, `Sa1C` existem porque algumas contas têm saldos temporários que são transferidos/apurados no final do período. Exemplo:
- Conta 71 (Vendas) tem saldo credor (`Ca`) durante o exercício
- No apuramento, é transferida para resultados e fica saldada (saldo = 0)

### 2. Qual a diferença entre S2C e S1C?

- **S2C**: Saldos devedores e credores são reportados **separadamente** (uma coluna para cada)
- **S1C**: Saldos são **somados algebricamente** e reportados num único campo líquido

### 3. O que é a "versão R2" em (R2) S2C?

É um ajuste técnico feito entre a 1ª e 2ª versão da taxonomia SVAT. Algumas contas mudaram de classificação. Para o sistema, comporta-se identicamente a S2C.

### 4. Como adicionar um novo tipo?

1. Editar ficheiro Excel → folha LEGENDA
2. Adicionar linha com novo tipo
3. Executar `make db-seed`
4. Pronto! Sem alteração de código

## Ver Também

- [Regras de Validação](regras_validacao_saldos.md) - Lógica detalhada de validação
- [Formato dos Ficheiros](formato_ficheiros.md) - Estrutura do Excel e CSV
- [README Principal](../README.md) - Visão geral do sistema

---

**Última Atualização**: 2025-11-26  
**Fonte**: Taxonomia SVAT + Análise da folha LEGENDA
