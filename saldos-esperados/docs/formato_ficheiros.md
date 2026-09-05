# Formato de Ficheiros

## Visão Geral

O sistema aceita dois tipos de ficheiros:
1. **Excel (.xlsx)**: Para importar regras de validação
2. **CSV (.csv)**: Para validar balancetes contabilísticos

## Ficheiros Excel (Regras de Validação)

### Estrutura Obrigatória

O ficheiro Excel deve conter **duas folhas**:

#### 1. Folha "LEGENDA"
Define os tipos de saldo suportados pelo sistema.

**Colunas**:
| Coluna | Nome        | Tipo   | Obrigatório | Descrição                    |
|--------|-------------|--------|-------------|------------------------------|
| A      | Code        | Texto  | Sim         | Código do tipo (ex: D, C, S2C) |
| B      | Description | Texto  | Não         | Descrição do tipo            |

**Exemplo**:
```
Code    Description
D       Devedor
C       Credor
Da      Devedor ou Ausente
Dc      Devedor ou Credor
Ca      Credor ou Ausente
Cc      Credor Confirmado
S2C     Saldo a 2 Credor
(R2) S2C Saldo a 2 Credor (Reversível)
S1C     Saldo a 1 Credor
Sa1C    Saldo Ausente a 1 Credor
Sc      Saldo Credor
```

**Regras**:
- Primeira linha é cabeçalho (ignorada)
- Códigos devem ser únicos
- Códigos são case-sensitive: `D` ≠ `d`
- Linhas vazias são ignoradas
- Máximo 50 tipos de saldo

#### 2. Folha "Saldo Esperados"
Define as regras de validação por conta.

**Colunas**:
| Coluna | Nome                  | Tipo   | Obrigatório | Descrição                    |
|--------|-----------------------|--------|-------------|------------------------------|
| A      | Account Number        | Texto  | Sim         | Número da conta              |
| B      | Expected Balance Type | Texto  | Sim         | Código do tipo (da LEGENDA)  |

**Exemplo**:
```
Account Number    Expected Balance Type
2111              C
2111*             C
21111             D
2112              Da
2113*             Dc
```

**Regras**:
- Primeira linha é cabeçalho (ignorada se contém "Account" ou "Conta")
- Account Number pode conter wildcards: `*` (zero ou mais dígitos)
- Expected Balance Type deve existir na LEGENDA
- Máximo 100.000 regras
- Duplicados são ignorados (última ocorrência prevalece)

### Wildcards em Contas

**Sintaxe suportada**:
- `2111*`: Contas que começam com 2111 (2111, 21110, 211101, etc.)
- `*5`: Contas que terminam com 5 (não recomendado, ambíguo)
- `21*1`: Contas com padrão (2101, 21001, 211231, etc.)

**Exemplos**:
```
Account    Rule       Matches
2111       C          2111 (exact match)
2111*      C          2111, 21110, 211101, 21111234
21*        D          21, 211, 2111, 21999, 210000
*          S2C        Todas as contas (fallback global)
```

**Prioridade**:
1. Match exato: `2111` → regra `2111`
2. Wildcard mais específico: `211101` → regra `2111*`
3. Wildcard menos específico: `211101` → regra `21*`
4. Fallback hierárquico (se ativado): `211101` → procura `21110`, `2111`, `211`, `21`, `2`

### Validação de Integridade

Ao importar regras, o sistema verifica:
- ✓ Folha LEGENDA existe
- ✓ Folha "Saldo Esperados" existe
- ✓ Todos os Expected Balance Types existem na LEGENDA
- ✓ Não existem códigos duplicados na LEGENDA
- ✗ Account Numbers duplicados (último prevalece, warning)

**Comando de importação**:
```bash
./bin/cli import-rules --file regras.xlsx
```

**Output com validação**:
```
Importing balance types from LEGENDA sheet...
✓ Successfully imported 11 balance types.

Importing expected balance rules...
⚠ Warning: Duplicate account 2111 (linha 45 sobrescreve linha 12)
✓ Successfully imported 324 rules.
```

## Ficheiros CSV (Balancetes)

### Estrutura Obrigatória

**Formato Geral**:
```csv
[Label Opcional - Primeira Linha]
Account,Balance
211101,1500.50
211102,-300.75
...
```

### Label (Primeira Linha)

**Opcional mas recomendado**. Se a primeira linha:
- Não tem formato `Account,Balance`
- Não começa com número
- Contém texto descritivo

Então é interpretada como **label do upload**.

**Exemplos**:
```csv
Balancete Janeiro 2025
Account,Balance
...
```

Label gravada: `"Balancete Janeiro 2025"`

```csv
Validação Trimestral Q4 2024 - Empresa XYZ
Account,Balance
...
```

Label gravada: `"Validação Trimestral Q4 2024 - Empresa XYZ"`

**Se não existir label**:
```csv
Account,Balance
211101,1500.50
```

Label gravada: `""` (vazio)

### Colunas Obrigatórias

| Coluna | Nome    | Tipo    | Formato          | Descrição                    |
|--------|---------|---------|------------------|------------------------------|
| 1      | Account | Texto   | Alfanumérico     | Número da conta              |
| 2      | Balance | Decimal | 1234.56 ou -500  | Saldo atual da conta         |

**Regras**:
- **Account**:
  - Pode conter letras, números, hífens (ex: `2111-A`)
  - Máximo 50 caracteres
  - Case-insensitive para matching
  
- **Balance**:
  - Formato: `1234.56` (ponto como separador decimal)
  - Aceita negativos: `-300.75`
  - Máximo 15 dígitos antes do decimal, 2 depois

### Separador

**Suportado**: Vírgula (`,`)

**NÃO suportado**:
- Ponto-e-vírgula (`;`)
- Tab (`\t`)
- Pipe (`|`)

Para converter, usar:
```bash
# Converter ; para ,
sed 's/;/,/g' input.csv > output.csv
```

### Encoding

**Suportados**:
- **UTF-8** (recomendado)
- **ISO-8859-1** (detectado automaticamente)
- **Windows-1252** (detectado como ISO-8859-1)

**Detecção automática**:
O parser tenta UTF-8 primeiro. Se falhar, tenta ISO-8859-1.

**Conversão manual** (se necessário):
```bash
# ISO-8859-1 → UTF-8
iconv -f ISO-8859-1 -t UTF-8 input.csv > output_utf8.csv

# Windows-1252 → UTF-8
iconv -f WINDOWS-1252 -t UTF-8 input.csv > output_utf8.csv
```

### Validação de Formato

**Erros comuns**:

#### 1. Cabeçalho inválido
```csv
Conta,Saldo  ❌ (nomes incorretos)
Account,Balance  ✓
```

#### 2. Separador errado
```csv
211101;1500.50  ❌ (ponto-e-vírgula)
211101,1500.50  ✓
```

#### 3. Decimal inválido
```csv
211101,1500,50  ❌ (vírgula como decimal)
211101,1500.50  ✓
```

#### 4. Linhas vazias
```csv
Account,Balance
211101,1500.50

211102,-300  ✓ (linhas vazias são ignoradas)
```

### Exemplo Completo Anotado

```csv
Balancete Janeiro 2025 - Validação Mensal
Account,Balance
# Contas de Ativo
211101,15000.50     # Match exato: regra 211101
211102,-300.00      # Match wildcard: regra 2111*
21111234,500        # Match hierárquico: 2111* → 211* → 21*
# Contas de Passivo  
2211,50000.00       # Match exato: regra 2211
22110,-1000         # Match wildcard: regra 2211*
# Contas sem regra
999999,100.00       # Erro: Sem regra aplicável
```

**Notas**:
- Linhas começadas com `#` são comentários (ignoradas)
- Espaços antes/depois são removidos automaticamente
- Case-insensitive: `Account` = `account` = `ACCOUNT`

## Limites e Restrições

### Excel
- **Tamanho máximo**: 10 MB
- **Folhas**: Máximo 10 (apenas LEGENDA e Saldo Esperados usadas)
- **Linhas por folha**: Máximo 100.000
- **Tipos de saldo (LEGENDA)**: Máximo 50
- **Regras (Saldo Esperados)**: Máximo 100.000

### CSV
- **Tamanho máximo**: 50 MB (configurável)
- **Linhas**: Ilimitado (processado em batches de 100)
- **Colunas**: Exatamente 2 (Account, Balance)
- **Label**: Máximo 255 caracteres

## Validação de Ficheiros

### Pré-validação (antes de processar)

**Excel**:
```bash
# Verificar estrutura do Excel
file regras.xlsx
# Deve retornar: Microsoft Excel 2007+

# Listar folhas (com Python)
python3 -c "import openpyxl; wb=openpyxl.load_workbook('regras.xlsx'); print(wb.sheetnames)"
# Deve incluir: ['LEGENDA', 'Saldo Esperados']
```

**CSV**:
```bash
# Verificar encoding
file -i balancete.csv
# UTF-8: charset=utf-8
# ISO-8859-1: charset=iso-8859-1

# Contar linhas
wc -l balancete.csv

# Verificar cabeçalho
head -2 balancete.csv
```

### Durante Processamento

O sistema valida:
- ✓ Encoding suportado
- ✓ Separador é vírgula
- ✓ Cabeçalho correto (Account, Balance)
- ✓ Balance é numérico
- ✓ Account não vazio
- ✗ Se conta tem regra (gera warning, não erro)

**Logs de validação**:
```
Processing balancete.csv...
Line 5: Warning - Account '999999' has no matching rule (using fallback)
Line 12: Error - Balance 'abc' is not a valid number (skipped)
Line 20: Warning - Account '' is empty (skipped)
```

## Boas Práticas

### Excel
1. **Nomenclatura**: Usar nomes descritivos: `regras_2025_v2.xlsx`
2. **Backup**: Manter versões anteriores antes de atualizar
3. **Validação**: Testar com `import-rules` antes de produção
4. **Comentários**: Usar colunas extras (C, D) para notas (ignoradas)

### CSV
1. **Label**: Sempre incluir label descritiva na 1ª linha
2. **Formato**: Exportar de Excel como "CSV UTF-8 (Comma delimited)"
3. **Validação**: Processar ficheiro pequeno de teste primeiro
4. **Compressão**: CSVs >10MB → comprimir antes de upload (`.csv.gz`)

## Ferramentas de Conversão

### Excel → CSV
```bash
# Com LibreOffice (CLI)
libreoffice --headless --convert-to csv:"Text - txt - csv (StarCalc)":44,34,76 regras.xlsx

# Com Python
python3 -c "
import pandas as pd
df = pd.read_excel('regras.xlsx', sheet_name='Saldo Esperados')
df.to_csv('regras.csv', index=False)
"
```

### CSV → UTF-8
```bash
# Detectar encoding atual
file -i input.csv

# Converter para UTF-8
iconv -f ISO-8859-1 -t UTF-8 input.csv > output.csv
```

### Limpar CSV
```bash
# Remover linhas vazias
sed '/^$/d' input.csv > output.csv

# Remover espaços
sed 's/ //g' input.csv > output.csv

# Remover BOM UTF-8 (se existir)
sed '1s/^\xEF\xBB\xBF//' input.csv > output.csv
```

## Resolução de Problemas

### "Failed to parse Excel"
**Causa**: Ficheiro corrompido ou formato inválido

**Solução**:
1. Abrir Excel e "Save As" → novo ficheiro
2. Verificar extensão é `.xlsx` (não `.xls`)
3. Verificar folhas LEGENDA e "Saldo Esperados" existem

### "Invalid CSV encoding"
**Causa**: Encoding não suportado

**Solução**:
```bash
# Detectar encoding
file -i ficheiro.csv

# Converter para UTF-8
iconv -f WINDOWS-1252 -t UTF-8 ficheiro.csv > ficheiro_utf8.csv
```

### "Missing expected balance type"
**Causa**: Regra referencia tipo não existente na LEGENDA

**Solução**:
1. Abrir Excel, folha "Saldo Esperados"
2. Verificar coluna B contém apenas códigos da LEGENDA
3. Corrigir códigos errados ou adicionar à LEGENDA

### "Duplicate account number"
**Causa**: Mesma conta aparece várias vezes

**Solução**:
- Decidir qual regra manter (última prevalece)
- Remover duplicados manualmente
- Aceitar warning (não é erro crítico)

## Referências

- [CSV RFC 4180](https://datatracker.ietf.org/doc/html/rfc4180)
- [Excel File Format](https://learn.microsoft.com/en-us/office/open-xml/spreadsheet)
- [ISO-8859-1 Encoding](https://en.wikipedia.org/wiki/ISO/IEC_8859-1)
