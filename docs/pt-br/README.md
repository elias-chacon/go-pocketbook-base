# go-pocketbook-base — Documentação

[🇬🇧 English](../en/README.md) · 🇧🇷 Português

O **go-pocketbook-base** é uma base para escrever **aplicativos para PocketBook (leitores Linux ARM) em Go**.
Ele traz:

- uma **biblioteca de componentes desenhada do zero com gráficos vetoriais** (botões, campos, tabelas,
  calendário…), organizada por um motor de **flex/grid** parecido com o CSS e pensada para e-ink;
- um binding enxuto para o leitor (toque, teclado do sistema, tela);
- um app que **se instala sozinho** (ícone de lançamento) na primeira execução;
- **build reproduzível em Docker** com o SDK do PocketBook e **versionamento** automático;
- tudo testável no seu PC, sem precisar do aparelho.

> Todo exemplo de código desta documentação vem da pasta [`examples/`](../../examples): ele compila, tem testes
> e o `go run ./tools/docsync` mantém as páginas idênticas a ele. Os comentários dentro do código ficam em
> inglês (o idioma do código-fonte); o texto ao redor é em português.

## Leia nesta ordem

| # | Página | O que você encontra |
|---|---|---|
| 1 | [Primeiros passos](01-primeiros-passos.md) | Do zero à sua primeira tela em minutos |
| 2 | [Instalação](02-instalacao.md) | Requisitos, instalar no leitor, primeira execução, desinstalar |
| 3 | [Build e versionamento](03-build-e-versionamento.md) | Docker, `build.ps1`/`make`, `.version`, testes, Sonar |
| 4 | [Conceitos](04-conceitos.md) | Como funciona: documento, componentes, coordenadas, desenho, toque |
| 5 | [Layout](05-layout.md) | `Style`, flex, grid, tamanhos, cores |
| 6 | [Componentes](06-componentes.md) | **Todos os componentes**: desenho, propriedades, código |
| 7 | [Eventos e estado](07-eventos-e-estado.md) | Callbacks, ler valores, mudar a tela |
| 8 | [Receitas](08-receitas.md) | Telas completas: login, configurações, catálogo, data, pedido, navegação |
| 9 | [Componentes próprios](09-componentes-proprios.md) | Criar o seu próprio elemento |
| 10 | [Testes](10-testes.md) | Testar telas e toque sem o aparelho |
| 11 | [Solução de problemas](11-solucao-de-problemas.md) | Particularidades do aparelho e problemas comuns |
| 12 | [Internacionalização](12-internacionalizacao.md) | pt-BR primeiro, fallback em inglês, traduções em JSON |
| 13 | [Apps e o contrato](13-apps-e-o-contrato.md) | `apps/<id>`, o manifesto, o ciclo de vida, o `apps/hello` como blueprint |
| 14 | [Arquitetura (DDD)](14-arquitetura.md) | contextos delimitados, camadas, linguagem ubíqua, a regra de dependência |
| 15 | [O Launcher](15-launcher.md) | a tela inicial dos apps: grade de instalados, executar um app e voltar |
| 16 | [O instalador e as releases](16-instalador.md) | o programa para o usuário final (Windows/Linux/macOS), como uma release é feita |

## Uma olhada rápida

```go
// examples/recipes/hello.go
package recipes

import (
	"fmt"

	"go-pocketbook-base/internal/toolkit/widget"
)

// Hello is the smallest useful screen: a card with a message and a button.
func Hello(onClose func()) widget.Widget {
	message := widget.NewLabel("Hello, PocketBook!")

	closeButton := widget.NewButton("Close")
	closeButton.SetVariant(widget.VariantPrimary)
	closeButton.OnClick = func(e *widget.Event) {
		fmt.Println("button tapped at", e.X, e.Y)
		onClose()
	}

	return widget.NewCard("Welcome", "My first screen", message, closeButton)
}
```

![Tela Hello](../components/recipe-hello.png)

## Mapa do projeto

```
apps/hello             o app blueprint (raiz de composição): uma vitrine de componentes
internal/kernel        núcleo compartilhado: geom, version
internal/toolkit       toolkit de UI: widget (componentes, layout, toque), gfx (desenho vetorial)
internal/device        o leitor: inkview (binding cgo), desktop (registro na tela inicial)
internal/localization  i18n: pt-BR primeiro, fallback em inglês
internal/platform      o que é um app: domain · application · infrastructure (+ a fachada platform)
internal/showcase      o conteúdo do app hello
internal/architecture  a regra de dependência, como código e como teste
cmd/preview            gera as páginas da demo em PNG
examples/              amostras testadas da documentação: components, recipes, custom, translation, harness
tools/                 bumpversion, docgen, docsync
docs/                  esta documentação (en, pt-br) e as imagens dos componentes
```
