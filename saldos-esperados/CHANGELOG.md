# Changelog

Todas as alterações relevantes neste projeto serão documentadas neste ficheiro.

O formato é baseado em [Keep a Changelog](https://keepachangelog.com/pt/1.0.0/),
e este projeto adere ao [Semantic Versioning](https://semver.org/lang/pt/).

## [Unreleased]

### Security - Phase 3 (2025-12-02)
- **Security Headers:** Adicionados CSP, X-Frame-Options, X-Content-Type-Options, X-XSS-Protection, Referrer-Policy, Permissions-Policy
- **Password Complexity:** Validação de força de password (mínimo 10 chars, maiúscula, minúscula, número, símbolo)
- **Database Connection Pooling:** Configurado MaxOpenConns=25, MaxIdleConns=5, ConnMaxLifetime=5min, ConnMaxIdleTime=5min
- **User Enumeration Prevention:** Mensagens de erro genéricas "credenciais inválidas" (não revela se user existe)
- Middleware SecurityHeadersMiddleware aplicado globalmente
- Função ValidatePasswordStrength disponível para handlers

### Security - Phase 2 (2025-12-02)
- **Account Lockout:** Conta bloqueada por 15 minutos após 5 tentativas de login falhadas
- **TOTP Encryption:** Secrets 2FA encriptados em BD com AES-256-GCM (usando ENCRYPTION_KEY)
- **Input Validation:** Funções de validação para username, email, account number, filenames (previne XSS, path traversal)
- Migração 005: Adicionadas colunas `failed_login_attempts` e `locked_until` na tabela users
- `generate-secrets.sh` atualizado para gerar ENCRYPTION_KEY (32 caracteres)
- Compatibilidade retroativa: TOTP continua funcional com secrets antigos (plaintext)

### Security - Phase 1 (2025-12-01)
- **Session Secret:** Removido secret hardcoded, agora configurável via `SESSION_SECRET` (mínimo 32 caracteres)
- **TLS/HTTPS:** Suporte completo para TLS com certificados via `USE_TLS`, `TLS_CERT_FILE`, `TLS_KEY_FILE`
- **CSRF Protection:** Middleware custom de proteção CSRF implementado (token em sessão, validação em POST/PUT/DELETE)
- **Rate Limiting:** 3 middlewares específicos implementados:
  - Login: 5 tentativas por 15 minutos
  - Upload: 10 uploads por minuto
  - API: 60 requests por minuto
- **Secrets Management:** Script `generate-secrets.sh` para gerar SESSION_SECRET e CSRF_AUTH_KEY
- Configuração de sessões com HttpOnly, Secure, SameSite=Strict (timeout 30 minutos)
- Documentação de segurança: `ia_docs/security_audit_2025-12-01.md`, `docs/QUICKSTART_SECURITY.md`

### Fixed
- Templates HTML com sintaxe malformada em botões delete (balances_list, types_list, users_list)
- Parser de templates Go corrigido (erro "<" in attribute name)

### Changed - UI/UX
- Melhorado layout dos filtros com ícone de lupa nas caixas de pesquisa
- Unificada altura de inputs, selects e botões (42px) para alinhamento perfeito
- Adicionado espaçamento entre avatar e nome do utilizador na sidebar
- Removidos estilos inline dos templates em favor de classes CSS reutilizáveis
- Ícone de lupa muda de cor (cinza → azul) ao interagir com campo de pesquisa
- Layout flex responsivo dos filtros com wrap automático

## [2.0.0] - 2025-11-27

### Added - Segurança & Gestão
- Autenticação robusta com bcrypt (cost 10)
- 2FA obrigatório com TOTP (Google Authenticator)
- Role-Based Access Control (RBAC): Admin, Manager, Operator
- Sistema completo de gestão de utilizadores
  - Criar, listar, editar, desativar utilizadores
  - Gestão granular de permissões por role
- Audit logs completo com pesquisa e filtros
  - Registo de login/logout, uploads, alterações de config
  - Filtros por utilizador, ação, data
- Comando CLI `create-user` para criar utilizadores com 2FA
- Pesquisa avançada de contas com wildcards (*, ?)
- Upload com progress bar e feedback visual
- Middleware de autenticação (RequireAuth, RequireAdmin)

### Changed
- Autenticação básica substituída por sistema robusto bcrypt+2FA
- Permissões granulares aplicadas em todas as rotas
- Interface web expandida (gestão users, audit logs, configurações)
- Schema BD: nova tabela `users` com `role` e `totp_secret`
- Tabela `audit_logs` para rastreabilidade completa

### Security
- Passwords armazenadas em bcrypt em vez de plaintext
- 2FA obrigatório para todos os utilizadores
- CSRF protection em todas as rotas POST/PUT/DELETE
- Proteção contra SQL injection via SQLC
- Audit trail completo de operações críticas
- Session-based authentication com secret seguro

### Added
- Sistema de tipos de saldo dinâmico via tabela `balance_types` (2025-11-26)
- Parser da folha LEGENDA do Excel para importar tipos automaticamente
- Suporte para 11 tipos de saldo (C, Ca, Cc, D, Da, Dc, S2C, (R2) S2C, S1C, Sa1C, Sc)
- Comando `make db-seed` para importar regras e tipos automaticamente
- Migrações 003 e 004 para gestão de balance_types
- Domain model `BalanceType` com validações
- Queries SQLC para gestão de tipos de saldo
- Foreign keys entre `expected_balances`/`processed_records` e `balance_types`
- Fallback de validação para compatibilidade com tipos legacy

### Changed
- Refatorado `Validator.validateBalance()` para consultar BD em vez de lógica hardcoded
- Comando `import-rules` agora importa tipos da LEGENDA antes de importar regras
- Schema SQL atualizado com tabela `balance_types`
- Makefile com cores corrigidas (SHELL := /bin/bash)
- Documentação expandida com 11 tipos suportados

### Fixed
- Confirmações interativas no Makefile (compatibilidade POSIX → bash)
- Cores não aplicadas nos comandos make (adicionado echo -e)
- Drop de tabela `balance_types` em `migrate-down`

## [0.2.0] - 2025-11-25

### Added
- Sistema de templates com herança (layouts + pages)
- Layouts `base.html` e `authenticated.html`
- Label de identificação em uploads (primeira linha do CSV)
- Coluna `expected_balance_type` em `processed_records` para histórico
- Migração 002 para adicionar label e balance_type
- Documentação completa de comandos de migração

### Changed
- Migrado sistema de templates de partials para herança
- Handlers web agora referenciam layouts em vez de ficheiros
- Redução de ~40% de código duplicado em templates
- Estrutura de templates: `layouts/` e `pages/`

### Fixed
- Ordem de carregamento de templates (layouts primeiro, depois pages)
- Parser Excel ignora linhas de cabeçalho ("Conta", "Account")

## [0.1.0] - 2025-11-24

### Added
- Validação de 4 tipos de saldo: D (Devedor), C (Credor), S2C, (R2) S2C
- Parser CSV com suporte a encodings ISO-8859-1 e UTF-8
- Parser Excel para importação de regras
- CLI com comandos `import-rules` e `process-file`
- Servidor web com dashboard e upload de ficheiros
- Autenticação básica via sessões
- Sistema de batch processing de CSVs
- Validação hierárquica de contas (fallback para contas-pai)
- SSH tunnel automático para base de dados remota
- SQLC para geração de queries type-safe
- Makefile com targets de desenvolvimento

### Database
- Tabelas: `expected_balances`, `uploads`, `processed_records`
- Índices para performance de queries
- Suporte a PostgreSQL

### Documentation
- README.md com instruções de setup
- WARP.md para orientação de agentes IA
- Documentação de regras de validação de saldos
- Comentários em Português de Portugal

---

## Tipos de Alterações

- **Added**: Novas funcionalidades
- **Changed**: Alterações em funcionalidades existentes
- **Deprecated**: Funcionalidades obsoletas (a remover)
- **Removed**: Funcionalidades removidas
- **Fixed**: Correções de bugs
- **Security**: Correções de vulnerabilidades
