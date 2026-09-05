# Testes

## Executar Testes

```bash
# Todos os testes
go test ./...

# Com verbose
go test -v ./...

# Específico
go test ./internal/service/

# Cobertura
go test -cover ./...
```

## Estrutura Table-Driven

```go
func TestValidateBalance(t *testing.T) {
    tests := []struct {
        name          string
        account       string
        balance       float64
        expectedType  string
        wantValid     bool
    }{
        {"Devedor válido", "2111", 100.0, "D", true},
        {"Credor válido", "2111", -50.0, "C", true},
        {"Devedor inválido", "2111", -50.0, "D", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            valid := ValidateBalance(tt.account, tt.balance, tt.expectedType)
            if valid != tt.wantValid {
                t.Errorf("got %v, want %v", valid, tt.wantValid)
            }
        })
    }
}
```

## Cobertura de Testes

Meta: **80%+**

```bash
# Gerar relatório
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Abrir browser
xdg-open coverage.html
```

## Convenções

- Ficheiro de teste: `*_test.go`
- Função de teste: `Test<NomeFuncao>`
- Comentários em **português de Portugal**
- Testes table-driven quando possível

## CI/CD

GitHub Actions example:
```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      - run: go test -cover ./...
```
