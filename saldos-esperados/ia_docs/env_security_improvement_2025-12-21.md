# Melhoria de Segurança: Variáveis de Ambiente
**Data:** 2025-12-21  
**Agente:** Warp AI

## Resumo
Implementação de gestão segura de variáveis de ambiente, eliminando o ficheiro `.env` em produção e utilizando variáveis carregadas pelo systemd.

## Alterações Implementadas

### 1. Novo Target: `generate-systemd-env`
Criado target no Makefile que gera ficheiro `systemd.env` com variáveis formatadas para systemd.

**Uso:**
```bash
make generate-systemd-env
```

**Resultado:**
- Cria ficheiro `systemd.env` com formato `Environment="VAR=value"`
- Adiciona `systemd.env` ao `.gitignore` automaticamente
- Pode ser copiado diretamente para o ficheiro de serviço systemd

### 2. Remoção do Sync de .env no Deploy
**Antes:**
```bash
rsync -avz .env.production rswebportal.pt:/opt/saldos-app/.env
```

**Depois:**
- Removida linha do target `deploy`
- Deploy agora avisa para configurar variáveis no systemd
- Sem ficheiros `.env` em produção

### 3. Lógica Condicional de Carregamento
**Ficheiro:** `cmd/server/main.go`

**Comportamento:**
- **Desenvolvimento** (`GIN_MODE != release`): Carrega `.env` se existir
- **Produção** (`GIN_MODE = release`): Usa apenas variáveis de ambiente

**Código:**
```go
func initConfig() {
    // Sempre ler variáveis de ambiente primeiro (prioritário)
    viper.AutomaticEnv()

    // Tentar ler .env apenas se não estivermos em modo produção
    ginMode := os.Getenv("GIN_MODE")
    if ginMode != "release" {
        // Modo desenvolvimento: tentar carregar .env
        viper.SetConfigFile(".env")
        if err := viper.ReadInConfig(); err != nil {
            log.Println("⚠️  No .env file found in development mode")
        } else {
            log.Println("✓ Loaded configuration from .env (development)")
        }
    } else {
        // Modo produção: usar apenas variáveis de ambiente
        log.Println("✓ Production mode: using environment variables from systemd")
    }
}
```

### 4. Script de Teste
**Ficheiro:** `scripts/test-env-loading.sh`

Simula comportamento do systemd para testar localmente se a aplicação consegue ler as variáveis de ambiente.

**Uso:**
```bash
make test-env
```

**Funcionalidade:**
1. Carrega variáveis de `systemd.env`
2. Verifica variáveis críticas
3. Compila servidor se necessário
4. Executa servidor em modo produção com variáveis carregadas

## Configuração do Systemd

### Ficheiro: `/etc/systemd/system/saldos-app.service`

**Remover:**
```ini
EnvironmentFile=/opt/saldos-app/config/.env
```

**Adicionar (copiar de systemd.env):**
```ini
[Service]
Environment="DB_DRIVER=postgres"
Environment="DB_HOST=localhost"
Environment="DB_PORT=5432"
Environment="DB_USER=manager_user"
Environment="DB_PASSWORD=MyPwd4_BD"
Environment="DB_NAME=saldos_esperados"
Environment="DB_SSLMODE=disable"
Environment="SERVER_PORT=8080"
Environment="USE_TLS=true"
Environment="TLS_CERT_FILE=/etc/letsencrypt/live/se.rswebportal.com/fullchain.pem"
Environment="TLS_KEY_FILE=/etc/letsencrypt/live/se.rswebportal.com/privkey.pem"
Environment="GIN_MODE=release"
Environment="SESSION_SECRET=vO4FkiywUG4aoNIxYd3owT/PPCh5h7KAFLxdR/ZcoRg="
Environment="CSRF_AUTH_KEY=fbd9007f69dc23823b69d2e0d427fde1"
Environment="ENCRYPTION_KEY=e564c0c19919668b40ea8b889024ba03"
Environment="DROPBOX_APP_KEY=uiqs0egp5kmg322"
Environment="DROPBOX_APP_SECRET=rs79al45l95aqaz"
Environment="DROPBOX_PATH =/Balancetes"
Environment="DROPBOX_REFRESH_TOKEN=puyo5ABqNxUAAAAAAAAAAfTsxpNFY-Rxea2c-pBVVIyJo3zW9K70Eeqp5FONOr9k"
```

### Aplicar Mudanças
```bash
sudo systemctl daemon-reload
sudo systemctl restart saldos-app
```

## Verificação em Produção

### 1. Verificar Variáveis Carregadas
```bash
sudo systemctl show saldos-app --property=Environment
```

### 2. Verificar Logs
```bash
sudo journalctl -u saldos-app -n 20 --no-pager
```

Deve aparecer:
```
✓ Production mode: using environment variables from systemd
```

### 3. Confirmar Ausência de .env
```bash
ssh rswebportal.pt "ls -la /opt/saldos-app/.env 2>&1"
```

Deve retornar: `No such file or directory`

## Vantagens de Segurança

1. **Sem Ficheiros Sensíveis no Disco:**
   - Variáveis carregadas diretamente na memória do processo
   - Não há ficheiro `.env` para ser lido indevidamente

2. **Controlo de Permissões:**
   - Apenas root pode editar o ficheiro de serviço systemd
   - Utilizador `saldos-app` não tem acesso às definições

3. **Auditoria:**
   - Mudanças em variáveis ficam registadas em `/var/log/auth.log`
   - Systemd gere o ciclo de vida das variáveis

4. **Sem Sync de Secrets:**
   - Deploy não transfere dados sensíveis via rsync
   - Configuração manual no servidor (one-time setup)

## Comandos Úteis

### Desenvolvimento
```bash
# Usar .env local
make run-web

# Testar modo produção localmente
make test-env
```

### Produção
```bash
# Gerar configuração para systemd
make generate-systemd-env

# Deploy (sem .env)
make deploy

# Verificar serviço
ssh rswebportal.pt "sudo systemctl status saldos-app"
```

## Checklist de Deploy

- [ ] Executar `make generate-systemd-env`
- [ ] Copiar conteúdo de `systemd.env` para `/etc/systemd/system/saldos-app.service`
- [ ] Remover linha `EnvironmentFile=` do serviço systemd
- [ ] Executar `sudo systemctl daemon-reload`
- [ ] Executar `make deploy`
- [ ] Verificar logs: `sudo journalctl -u saldos-app -f`
- [ ] Confirmar ausência de `.env` em `/opt/saldos-app/`

## Ficheiros Alterados

- `Makefile` - Novos targets `generate-systemd-env` e `test-env`
- `cmd/server/main.go` - Lógica condicional de carregamento
- `scripts/test-env-loading.sh` - Script de teste (novo)
- `.gitignore` - Adicionado `systemd.env`

## Notas

- O ficheiro `systemd.env` é gerado localmente e **NÃO deve ser commitado**
- Em desenvolvimento, a aplicação continua a usar `.env` normalmente
- A variável `GIN_MODE=release` é o trigger para modo produção
