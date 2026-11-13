# go-pocketbook-base

Base para criar **aplicativos para PocketBook (e-readers Linux ARM) em Go**, com uma
biblioteca de componentes visuais desenhada do zero em gráficos vetoriais
([`fogleman/gg`](https://github.com/fogleman/gg)), ícone que se registra sozinho e build
reproduzível em Docker. Faça um fork para começar novos apps.

O app de exemplo (`cmd/hello`) é uma vitrine com todos os componentes, em 4 páginas
(**Entradas**, **Escolhas**, **Dados**, **Calendário**) navegadas por uma barra de menus.

> Versão atual: ver [`.version`](.version) · `MAJOR.MINOR.BUILD` (veja [Versionamento](#versionamento)).

## Sumário

- [Requisitos](#requisitos) · [Tasks](#tasks) · [Backlog](#backlog)
- [Versionamento](#versionamento)
- [Arquitetura](#arquitetura)
- [Componentes](#componentes)
- [Desenvolvimento](#desenvolvimento) · [Build e instalação](#build-e-instalação) · [Qualidade (Sonar)](#qualidade-sonar)
- [Notas do aparelho](#notas-do-aparelho-pb741)

## Requisitos

### Funcionais

| ID | Requisito |
|---|---|
| RF-01 | Cada elemento visual é um **objeto com propriedades parecidas com as do JavaScript** (`ID`, `ClassName`, `Style`, `Hidden`, `Disabled`, `Value`, `Checked`, `OnClick`, `OnChange`…), para que no futuro as telas sejam descritas em **HTML** e formatadas com **CSS**. |
| RF-02 | Layout com **painel flex** (direction, wrap, grow, gap, justify, align) e **grid** (colunas px/`fr`, span, gap). |
| RF-03 | Componentes: botão, combo-box, check-box, radio-button, switch, menu bar, input texto, input numérico (máscara, `stepUp`/`stepDown`), input de busca (botão de lupa), text-area, data-table, gauge/progress-bar, slider de tamanho configurável, chips, calendário, card e dropdown. |
| RF-04 | **O toque cai exatamente sobre o elemento desenhado** (desenho e toque usam as mesmas coordenadas) e os alvos têm tamanho mínimo confortável. |
| RF-05 | Entrada de texto usa o **teclado do sistema** do aparelho (numérico para campos numéricos). |
| RF-06 | Design pensado para e-ink: alto contraste, bordas nítidas, sem animação, alvos de toque grandes. |
| RF-07 | O app se **instala sozinho** (ícone e entrada no desktop) na primeira execução. |
| RF-08 | Versionamento em `.version` (começa em `1.0.0`): o build incrementa o **BUILD**, tarefa concluída incrementa o **MINOR**, grande mudança incrementa o **MAJOR**. |

### Não funcionais

| ID | Requisito |
|---|---|
| RNF-01 | SOLID aplicado com pragmatismo, código limpo, DRY e KISS. |
| RNF-02 | Análise estática no **SonarQube local** (`scripts/sonar.ps1`), sem code smells abertos. |
| RNF-03 | Compatível com **Go 1.17** (toolchain do SDK do PocketBook): sem generics. |
| RNF-04 | Lógica de UI testável no PC, sem aparelho (`go test ./...`, `cmd/preview`). |
| RNF-05 | A UI escala para qualquer resolução (referência: 1404 px de largura, PB741). |

## Tasks

Legenda: `[x]` concluída · `[ ]` pendente. Ao concluir uma task, o MINOR é incrementado
(`make task-done` ou `go run ./tools/bumpversion minor`).

**Base**

- [x] T01 Versionamento: `.version`, `tools/bumpversion`, BUILD a cada build, versão no binário e no rodapé
- [x] T02 Camada vetorial `gfx`: canvas escalável, fontes, texto com reticências, formas, ícones
- [x] T03 Modelo DOM: `Base` (ID, ClassName, Hidden, Disabled, OnClick), árvore, `GetElementByID`, `QuerySelectorAll`
- [x] T04 `Style` estilo CSS (box model border-box, cores, fonte, borda, raio)
- [x] T05 Layout **flex** (direction, wrap, grow, gap, justify-content, align-items)
- [x] T06 Layout **grid** (colunas px/`fr`, span, gap, linhas automáticas)
- [x] T07 Eventos de toque: down/move/up/click, captura, bubbling com `StopPropagation`, desabilitado/oculto
- [x] T08 Popups (overlay) e teclado do sistema (`OpenKeyboard`, numérico)

**Componentes**

- [x] T09 Painel flex (`NewPanel`, `NewRow`, `NewColumn`)
- [x] T10 Grid (`NewGrid`)
- [x] T11 Botão (primário, secundário, fantasma, só ícone)
- [x] T12 Combo-box
- [x] T13 Check-box
- [x] T14 Radio-button (grupo por `Name`)
- [x] T15 Menu bar (itens diretos e submenus)
- [x] T16 Input texto (e senha)
- [x] T17 Input numérico com máscara, `StepUp`/`StepDown`, min/max, decimais, prefixo/sufixo
- [x] T18 Input de busca com lupa e limpar
- [x] T19 Text-area
- [x] T20 Data-table (ordenação, seleção, paginação)
- [x] T21 Gauge e progress-bar
- [x] T22 Slider com tamanho configurável (largura, altura e botão)
- [x] T23 Chips (múltiplos, exclusivos, removíveis)
- [x] T24 Switch
- [x] T25 Calendário (pt-BR/en, navegação de mês, hoje, seleção)
- [x] T26 Card
- [x] T27 Dropdown

**Entrega e qualidade**

- [x] T28 Tela de demonstração com todos os componentes (4 páginas) e rodapé com o último toque
- [x] T29 Testes de toque: centro de cada elemento interativo, tamanho mínimo, sem estouro de tela
- [x] T30 Integração com o Sonar local (`sonar-project.properties`, `scripts/sonar.ps1`) e cobertura
- [x] T31 README com requisitos e tasks
- [x] T32 Slide button com 3 posições (esquerda · centro · direita) na demo e teste de encaixe

## Backlog

Ideias para próximas versões (ainda **não** feitas):

- [ ] B01 Parser de **HTML** que gera a árvore de elementos (`<div>`, `<button>`, `<input type=…>`…)
- [ ] B02 Parser de **CSS** (seletores por tag, `.classe`, `#id`; cascata e herança)
- [ ] B03 Atualização parcial da tela (`PartialUpdate`) para interações mais rápidas
- [ ] B04 Container com rolagem (scroll) e `overflow`
- [ ] B05 Tratar rotação (`EVT_ORIENTATION`)
- [ ] B06 Diálogo/modal e toast
- [ ] B07 Abas (tabs) e navegação por pilha de telas
- [ ] B08 Slide-to-confirm (botão deslizante de confirmação)
- [ ] B09 Remover o log de eventos de depuração (`DebugLog`) em builds de release

## Versionamento

Formato `MAJOR.MINOR.BUILD`, guardado em [`.version`](.version) (começa em `1.0.0`):

| Quando | O que sobe | Como |
|---|---|---|
| Cada **build** | `BUILD` | automático em `build.ps1` / `make dist` |
| **Task concluída** | `MINOR` (e `BUILD` volta a 0) | `make task-done` |
| **Grande mudança** | `MAJOR` (`MINOR` e `BUILD` voltam a 0) | `make major` |

```sh
go run ./tools/bumpversion build   # 1.2.3 -> 1.2.4
go run ./tools/bumpversion minor   # 1.2.4 -> 1.3.0
go run ./tools/bumpversion major   # 1.3.0 -> 2.0.0
```

A versão é gravada no binário por `-ldflags -X …/internal/version.Value` e aparece no rodapé do app.

## Arquitetura

```
cmd/hello          app do aparelho: liga InkView ↔ widget.Document (toque, teclado, repaint)
cmd/preview        renderiza as páginas da demo em PNG (sem aparelho)
internal/demo      vitrine de componentes (compartilhada por hello, preview e testes)
internal/widget    componentes + layout flex/grid + despacho de toque (a biblioteca)
internal/gfx       desenho vetorial: Canvas, fontes, texto, ícones, cores
internal/geom      Rect, Point, Size, Insets
internal/inkview   binding cgo mínimo da libinkview (eventos, blit, teclado)
internal/install   auto-instalação (ícones + entrada no view.json do desktop)
internal/ui        ícone de lançamento
internal/version   MAJOR.MINOR.BUILD
tools/bumpversion  incrementa .version
```

Decisões (SOLID/KISS/DRY, sem exageros):

- **Interfaces pequenas**: `Widget` (`Elem`, `Measure`, `Paint`), `PointerHandler`, `Keyboard`. O
  teclado é injetado (`FakeKeyboard` em testes/preview, InkView no aparelho): o código de UI não
  conhece o hardware.
- **Um motor de layout** (`layout.go`) para todos: *block* é um flex em coluna; o `Document` só
  chama `layoutWidget`. Cada componente só implementa `Measure` e `Paint`.
- **DRY**: `ComboBox`, `Dropdown` e `MenuBar` reusam o mesmo popup (`menuList`); `Input` usa
  "adornos" (botões laterais) para − / +, lupa e limpar; `paintBox` desenha fundo/borda de todos.
- **Coordenadas em "pixels de design"** (largura de referência 1404). O `Canvas` converte para a
  tela real, e o toque é convertido de volta: desenho e toque usam o mesmo espaço.
- Elementos com propriedades à la JS; **estilos** em `Style` (nomes do CSS), prontos para um
  futuro parser de CSS.

## Componentes

Todos os elementos embutem `Base`: `ID`, `ClassName`, `Style`, `Hidden`, `Disabled`, `OnClick`,
`AppendChild`, `RemoveChild`, `Bounds`. Os callbacks recebem `*Event` (`Target`, `X`, `Y`,
`StopPropagation()`), como no navegador.

| Componente | Construtor | Principais propriedades |
|---|---|---|
| Painel | `NewPanel`, `NewRow`, `NewColumn`, `NewGrid` | `Style.Display/FlexDirection/FlexWrap/JustifyContent/AlignItems/Gap/GridColumns` |
| Texto | `NewLabel` | `Text`, `Style.FontSize/Bold/Color/TextAlign` |
| Botão | `NewButton`, `NewIconButton` | `Text`, `Icon`, `SetVariant(Primary/Secondary/Ghost)`, `OnClick` |
| Check-box / Radio / Switch | `NewCheckbox`, `NewRadio`, `NewSwitch` | `Label`, `Checked`, `Name` (grupo do radio), `Value`, `OnChange` |
| Combo-box | `NewComboBox` | `Options`, `Index`, `Value()`, `SetValue`, `Placeholder`, `OnChange` |
| Dropdown | `NewDropdown` | `Text`, `Items []MenuItem`, `Chosen`, `OnSelect` |
| Menu bar | `NewMenuBar` | `Menus []*Menu` (`Label`, `Items`, `OnSelect`, `Selected`) |
| Input | `NewInput`, `NewNumberInput`, `NewMaskedInput`, `NewSearchInput` | `Type`, `Value`, `Placeholder`, `MaxLength`, `ReadOnly`, `Min/Max/Step/Decimals`, `Prefix/Suffix`, `Mask`, `NoSteppers`, `StepUp()/StepDown()`, `Number()`, `OnChange`, `OnSubmit` |
| Text-area | `NewTextArea` | `Value`, `Placeholder`, `Rows`, `MaxLength`, `OnChange` |
| Slider | `NewSlider` | `Min/Max/Step/Value`, `ThumbSize`, `Style.Width/Height`, `OnInput`, `OnChange` |
| Progress / Gauge | `NewProgress`, `NewGauge` | `Value/Min/Max`, `Text`, `Unit`, `Label` |
| Chips | `NewChip`, `NewChipGroup` | `Label`, `Selected`, `Closable`, `OnChange`, `OnClose`; grupo `Exclusive`, `SelectedLabels()` |
| Data-table | `NewTable` | `Columns []TableColumn`, `Rows`, `PageSize`, `Page`, `Selected`, `SortCol/SortDesc`, `Summary`, `OnSelect` |
| Calendário | `NewCalendar` | `Year/Month`, `Selected`, `Today`, `Locale` (`LocalePTBR`/`LocaleEN`), `OnChange` |
| Card | `NewCard(title, subtitle, children…)` | é um painel em coluna com borda |

Máscaras de dígitos: `0`/`#` = um dígito, qualquer outro caractere é literal
(`(00) 00000-0000`, `000.000.000-00`).

## Desenvolvimento

```sh
go test ./...                        # testes (layout, componentes, toque, demo, versão)
go run ./cmd/preview                 # gera preview-0..3.png no tamanho do PB741
go run ./cmd/preview -page 2 -w 758 -h 1024   # uma página, outra resolução
```

## Build e instalação

Precisa de Docker (a imagem traz o SDK do PocketBook e o Go 1.17 corrigido).

```sh
make dist          # Linux/macOS/WSL
.\build.ps1        # Windows (Docker Desktop ou Docker dentro do WSL)
```

Gera `dist/applications/hello.app` (arquivo único) e **incrementa o BUILD**.

**Instalar = copiar um arquivo.** Copie `hello.app` para `<aparelho>/applications/` (ou
`.\scripts\install.ps1 -Drive F:`) e abra pela lista de Aplicativos. Na **primeira execução** o app:

1. gera os ícones (normal e focado, 128×128 BMP de 8 bits) em `applications/icons/`;
2. registra a entrada `U_hello` em `system/profiles/default/config/desktop/view.json` (o
   `system/config` é um link para essa pasta) e a chave no grupo `@General`, com **backup**
   (`view.json.bak-<hora>`), validação do JSON e troca atômica; é idempotente e corrige entradas
   antigas com caminho relativo.

O ícone aparece quando o desktop recarrega a configuração (por exemplo, ao tirar o cabo USB).

## Qualidade (Sonar)

Com um SonarQube local em `http://localhost:9000` e um token em `SONAR_GLOBAL_TOKEN`:

```powershell
.\scripts\sonar.ps1      # Windows: testes com cobertura + scanner (Docker no WSL)
```
```sh
scripts/sonar.sh         # Linux/WSL
```

A configuração está em [`sonar-project.properties`](sonar-project.properties).

## Notas do aparelho (PB741)

Desenvolvido e testado em um **PocketBook PB741-N-WW** (InkPad Color 2): 7,8", 1404×1872, canvas de
24 bits, firmware 6.x.

- Com o painel do sistema ligado (padrão), o InkView reserva 137 px e desloca o canvas; o app chama
  `inkview.DisablePanel()` no início para a tela inteira ser do app.
- O `path` de `U_<app>` no `view.json` precisa ser **absoluto**; ícones podem ser relativos.
- O app grava `applications/hello.log` (eventos) para depuração; remova a linha `DebugLog` em release.

## Licença

MIT
