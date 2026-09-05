# Guia de Configuração Dropbox API

Este guia detalha os passos para criar e configurar uma aplicação na Dropbox para permitir a integração com o SaldosEsperados.

## 1. Criar Aplicação na Dropbox
1. Aceda à [App Console da Dropbox](https://www.dropbox.com/developers/apps).
2. Clique em **"Create app"**.
3. Escolha as seguintes opções:
   - **Scoped access**: Selecionado (Obrigatoriedade moderna).
   - **Choose the type of access**: "App folder" (Recomendado - cria uma pasta segura `/Apps/NomeDaSuaApp`) ou "Full Dropbox" (se precisar de aceder a tudo).
   - **Name your app**: Dê um nome único (ex: `SaldosEsperados-Prod`).
4. Clique em **"Create app"**.

## 2. Configurar Permissões (Scopes)
Para resolver o erro `missing_scope`, é crítico ativar as permissões certas **ANTES** de gerar qualquer token.

1. Na página da sua App, vá à aba **"Permissions"**.
2. Na secção **"Files"**, marque as caixas:
   - `files.metadata.read` (Para listar ficheiros)
   - `files.content.read` (Para descarregar ficheiros)
3. Clique em **"Submit"** no fundo da página para guardar.

## 3. Obter Credenciais
1. Vá à aba **"Settings"**.
2. Encontre a secção **"OAuth 2"**.
3. Copie os valores de:
   - **App key** (`DROPBOX_APP_KEY`)
   - **App secret** (`DROPBOX_APP_SECRET`)

## 4. Gerar Refresh Token (Acesso Permanente)
Como a aplicação corre num servidor, precisamos de um token que não expire.

1. Configure as chaves no seu `.env` local:
   ```env
   DROPBOX_APP_KEY=sua_app_key
   DROPBOX_APP_SECRET=seu_app_secret
   ```
2. Inicie a aplicação (`make run-web`).
3. Abra `http://localhost:8080/dropbox/setup` (ou o domínio do seu servidor).
4. Clique em **"Obter Código de Acesso"**.
   - A Dropbox pedirá aprovação. **Verifique se ela lista as permissões de leitura**.
5. Copie o **Access Code** fornecido.
6. Execute o comando `curl` mostrado na página de setup (no seu terminal).
   Exemplo:
   ```bash
   curl https://api.dropbox.com/oauth2/token \
       -d code=CODIGO_COPIADO \
       -d grant_type=authorization_code \
       -d client_id=SUA_APP_KEY \
       -d client_secret=SEU_APP_SECRET
   ```
7. A resposta será um JSON. Copie o valor de `"refresh_token"`.

## 5. Configuração Final
Adicione o token ao `.env`:
```env
DROPBOX_REFRESH_TOKEN=valor_do_refresh_token_copiado
```
Reinicie a aplicação. A integração funcionará agora perpetuamente.
