# 12. Internacionalização (i18n)

[← Solução de problemas](11-solucao-de-problemas.md) · [Índice](README.md)

Todo app deste projeto é escrito em **português do Brasil (pt-BR)** e cai automaticamente para o **inglês (en)**, pelo
pacote reutilizável `internal/localization/i18n`. Um texto que um idioma não traduz nunca fica em branco: vem do inglês, depois do
português e, por último, aparece a própria chave (assim uma tradução esquecida fica visível).

## As regras

| | |
|---|---|
| Idioma primário | `i18n.Primary` = `pt-BR`: o idioma em que o app é escrito e o padrão |
| Fallback | `i18n.Fallback` = `en`: usado para qualquer chave que o idioma atual não tem |
| Ordem de busca | idioma atual → idioma base (`pt-BR` → `pt`) → `en` → `pt-BR` → a chave |
| Etiquetas | `pt_BR`, `PT-br`, `pt_BR.UTF-8` são todas entendidas (`i18n.Normalize`); `pt` encontra o `pt-BR` registrado, `en-GB` encontra `en` |

## Arquivos

Um JSON simples por idioma, com o nome dele, embutido no programa com `go:embed` (não há arquivos para copiar para o
leitor):

```json
// locales/pt-BR.json
{ "greeting.title": "Olá!", "greeting.hello": "Bem-vinda(o), %s.", "items": "Você tem %d itens" }

// locales/en.json
{ "greeting.title": "Hello!", "greeting.hello": "Welcome, %s.", "items": "You have %d items", "farewell": "See you soon" }
```

Os textos podem ter verbos do `fmt` (`%s`, `%d`), preenchidos pelos argumentos de `T`. Um texto sem argumentos volta
intacto, então um `%` literal é seguro.

## Usando

```go
// examples/translation/translation.go
package translation

import (
	"embed"

	"go-pocketbook-base/internal/localization/i18n"
	"go-pocketbook-base/internal/toolkit/widget"
)

//go:embed locales/*.json
var locales embed.FS

// NewTranslator loads pt-BR (the primary language) and en (the fallback) from
// the embedded JSON files, one <language>.json per language.
func NewTranslator() *i18n.Bundle {
	tr := i18n.New()
	if err := tr.AddFS(locales, "locales"); err != nil {
		panic(err) // the files are embedded: failing here is a programming mistake
	}
	return tr
}

// GreetingScreen shows translated texts and a combo-box that changes the
// language. "farewell" exists only in English, so in pt-BR it falls back to it.
func GreetingScreen(tr *i18n.Bundle, name string) widget.Widget {
	title := widget.NewLabel("")
	title.Style.FontSize, title.Style.Bold = widget.FontTitle, true
	hello, items, bye := widget.NewLabel(""), widget.NewLabel(""), widget.NewLabel("")
	lang := widget.NewComboBox("",
		widget.Option{Label: "Português", Value: "pt-BR"}, widget.Option{Label: "English", Value: "en"})
	lang.SetValue(tr.Lang())

	// Every visible text is set here, so changing the language is one call.
	refresh := func() {
		title.Text = tr.T("greeting.title")
		hello.Text = tr.T("greeting.hello", name)
		items.Text = tr.T("items", 3)
		bye.Text = tr.T("farewell")
		lang.Placeholder = tr.T("language")
	}
	refresh()
	tr.OnChange = refresh
	lang.OnChange = func(e *widget.Event) { tr.SetLang(lang.Value()) }

	return widget.NewColumn(12, title, hello, items, bye, lang)
}
```

| | |
|---|---|
| `i18n.New()` | um bundle vazio cujo idioma é pt-BR |
| `AddFS(fsys, dir)` | carrega todo `<idioma>.json` de uma pasta (um `embed.FS`) |
| `AddJSON(idioma, dados)` · `Add(idioma, mapa)` | um idioma a partir de bytes JSON / de um mapa Go (mescla com o que já existe) |
| `T(chave, args...)` | o texto traduzido |
| `SetLang(etiqueta)` · `Lang()` · `Langs()` | escolhe e lista idiomas; `OnChange` roda quando o idioma muda |
| `Has(chave)` · `Missing(idioma)` | se uma chave existe; quais chaves um idioma ainda não tem |

## Trocando o idioma em tempo de execução

Os widgets guardam strings simples, então depois do `SetLang` os textos precisam ser atribuídos de novo. Mantenha todo
texto traduzido numa função `refresh` (como o exemplo faz) e chame-a em `OnChange`; ou refaça a página (com
[`Tabs`](06-componentes.md#abas-tabs) ela é refeita a cada vez que é mostrada).

## Mantendo um idioma completo

```go
if missing := tr.Missing("en"); len(missing) > 0 {
	t.Errorf("sem tradução: %v", missing)
}
```

Coloque isso num teste do seu app: ele lista as chaves que pt-BR ou en definem e o idioma dado não.

[← Solução de problemas](11-solucao-de-problemas.md) · [Índice](README.md)
