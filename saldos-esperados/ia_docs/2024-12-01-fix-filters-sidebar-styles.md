# Correção de Estilos - Filtros e Sidebar

**Data**: 2024-12-01  
**Tipo**: Correção de UI/UX

## Problema Identificado

1. **Caixas de pesquisa e filtros** com altura mínima e sem estilo adequado
2. **Espaçamento insuficiente** entre o avatar e o nome do utilizador na sidebar

## Alterações Realizadas

### 1. Ícone de Lupa nas Caixas de Pesquisa

Adicionado ícone SVG de lupa (search icon) dentro das caixas de pesquisa:
- Posicionado absolutamente à esquerda do input
- Cor cinza (--gray-400) por padrão
- Muda para azul (--primary-600) quando o campo está em focus ou preenchido
- Padding-left do input ajustado para 2.75rem para acomodar o ícone
- `pointer-events: none` para não interferir com cliques no input

### 2. Estilos para Filtros e Pesquisa

Adicionadas classes CSS dedicadas para os elementos de filtro:

#### `.filters-bar`
- Container principal dos filtros
- Background branco com borda e sombra
- Padding e margin bottom consistentes
- Border radius arredondado

#### `.filters-form`
- Layout flex horizontal com wrap automático
- Gap de 0.75rem entre elementos
- Alinhamento vertical centrado
- Elimina estilos inline dos templates

#### `.search-box` e `.search-box input`
- Container com `position: relative` para acomodar o ícone
- Largura flexível (flex: 1) com min-width de 200px
- Altura fixa de 42px
- Padding interno: 0.625rem 1rem 0.625rem 2.75rem (espaço para ícone)
- Line-height 1.5 para alinhamento vertical do texto
- Estados de focus com border azul e shadow

#### `.filter-select`
- Altura fixa de 42px (consistente com inputs e botões)
- Padding e line-height idênticos aos inputs
- Min-width 180px (reduzido para melhor ajuste)
- Flex-shrink: 0 para evitar compressão
- Cursor pointer para indicar interatividade
- Estados de focus coordenados

#### `.filter-buttons`
- Container flex para botões de ação
- Gap de 0.5rem entre botões
- Flex-shrink: 0 para manter tamanho fixo

#### `.btn` (classe base)
- Altura fixa de 42px para alinhamento com inputs
- Line-height 1.5 para centrar texto verticalmente
- Padding horizontal de 1.25rem
- Display inline-flex para alinhamento de ícones
- White-space: nowrap para evitar quebra de texto

### 2. Espaçamento na Sidebar

Adicionado `margin-left: 0.75rem` à classe `.user-info-text` para criar espaço visual adequado entre o avatar circular e o bloco de texto (nome + role).

## Resultados Esperados

1. ✅ Caixas de pesquisa, selects e botões alinhados na mesma altura (42px)
2. ✅ Inputs de texto e selects com altura visível e adequada
3. ✅ Botões de filtro visualmente consistentes
4. ✅ Avatar e nome do utilizador visualmente separados na sidebar
5. ✅ Layout responsivo e moderno mantido

## Ficheiros Modificados

### Templates
- `web/templates/pages/balances_list.html`
  - Linhas 33-54: Adicionado ícone de lupa e classe `filters-form`
  - Removidos estilos inline do formulário
  - Adicionado container `.filter-buttons`

- `web/templates/pages/audit_list.html`
  - Linhas 6-38: Adicionado ícone de lupa e classe `filters-form`
  - Removidos estilos inline do formulário
  - Adicionado container `.filter-buttons`

### CSS
- `web/assets/css/style.css`
  - Linhas 174-177: Adicionado margin-left à classe `.user-info-text`
  - Linhas 468-566: Adicionadas classes:
    - `.filters-form`: layout do formulário
    - `.search-box`: container relativo para input + ícone
    - `.search-icon`: posicionamento absoluto do ícone
    - `.search-box input`: padding ajustado para acomodar ícone
    - `.filter-select`: estilos dos selects
    - `.filter-buttons`: container dos botões de ação
    - `.btn`: classe base unificada para todos os botões

## Validação

- ✅ Compilação sem erros: `go build ./cmd/server`
- ✅ Classes CSS adicionadas ao stylesheet principal
- ✅ Estilos aplicam-se globalmente a todas as páginas com filtros
- ⏳ Teste visual pendente no browser

## Notas Técnicas

- Todas as alturas fixadas em 42px para garantir alinhamento visual perfeito
- Line-height de 1.5 garante centramento vertical do texto
- Transições suaves mantidas (0.2s) para estados hover/focus
- Variáveis CSS utilizadas para manter consistência de cores e bordas
- Estilos inline nos templates devem ser gradualmente eliminados em favor destas classes

## Próximos Passos

1. Testar visualmente em browser todas as páginas:
   - `/balances` (Saldos Esperados)
   - `/types` (Tipos de Saldo)
   - `/users` (Utilizadores)
   - `/audit` (Audit Logs)
2. Validar responsividade em mobile
3. Confirmar alinhamento correto dos elementos de filtro
