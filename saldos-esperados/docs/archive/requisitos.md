## Requisitos

Implementar aplicação Web que receba arquivos CSV com as seguintes colunas:
- Número Conta (coluna 1)
- Designação Conta (coluna 2)
- Saldo (coluna 3)

Os dados começam na quarta linha do arquivo.
O arquivo deve ser processado em lotes de 100 linhas.

### Saldos Esperados

A pasta data contém exemplo de arquivo csv que serão submetidos para processamento, via upload.

O ficheiro data/saldos_esperados.xlsx contém os saldos esperados para cada conta.
Indica se determinada conta deve ter saldo positivo ou negativo.

### Processamento

Deve ser gerado ficheiro csv com a indicação se determinada conta tem saldo esperado ou não.
Para o efeito de processamento apenas interessam as contas a 8 digitos.
As contas a 8 digitos para saberem se tem saldo esperado ou não devem ser comparadas com o ficheiro data/saldos_esperados.xlsx. contas "mãe", ou seja contas com menos digitos.

## Base de dados

O ficheiro data/saldos_esperados.xlsx deve ser convertido para base de dados.

Os ficheiros csv uploaded devem ser mantidos em base de dados para histórico, assim como os resultados do processamento.

### Interface

A interface deve moderna e responsive mas simultaneamnete deve ser simples e intuitiva.
Deve ter um menu com as seguintes opções:
- Upload de ficheiro csv
- Visualização de resultados
- Visualização de histórico
- Visualização de saldos esperados

Deve ser possível exportar os resultados para ficheiro csv.

A aplicação deve ser implementada em Go.

Deverá ser implementado mecanismo de autenticação e autorização.

Avaliar possibilidade de efetuar login via Microsoft ou como alternativa a implementação de um mecanismo de autenticação 2FA.

