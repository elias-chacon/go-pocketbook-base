# 10. Testes

[← Componentes próprios](09-componentes-proprios.md) · [Índice](README.md) · [Próxima: Solução de problemas →](11-solucao-de-problemas.md)

Como a interface é Go puro, você testa **telas, layout e toque** no PC com o `go test` normal, sem leitor. Esse é o
principal motivo de a interface ter sido construída assim.

## As três ferramentas

| Ferramenta | O que dá a você |
|---|---|
| `widget.FakeKeyboard` | um teclado que responde na hora: `Reply` é o texto digitado, `Cancel` simula cancelar, `Last` é o último pedido |
| `doc.HandlePointer(...)` | simula um dedo (down, move, up) num ponto |
| `doc.Render()` / `harness.SavePNG` | desenha a tela para você inspecionar pixels ou olhar um PNG |

## Ajudantes

A maioria dos testes começa com estes ajudantes pequenos (eles ficam em `examples/harness/testkit.go`):

```go
// examples/harness/testkit.go#NewDoc
// NewDoc puts content in a Screen with a fake keyboard (it answers "7") and lays
// it out: the usual start of a test.
func NewDoc(content widget.Widget) (*widget.Document, *widget.FakeKeyboard) {
	kb := &widget.FakeKeyboard{Reply: "7"}
	doc := Screen(1404, 1872, kb, content)
	doc.Layout()
	return doc, kb
}
```

```go
// examples/harness/testkit.go#TapAt
// TapAt touches the screen at p (design pixels): finger down, then up.
func TapAt(doc *widget.Document, p geom.Point) {
	doc.HandlePointer(widget.PointerDown, p.X*doc.Scale, p.Y*doc.Scale)
	doc.HandlePointer(widget.PointerUp, p.X*doc.Scale, p.Y*doc.Scale)
}
```

```go
// examples/harness/testkit.go#Tap
// Tap touches the center of w.
func Tap(doc *widget.Document, w widget.Widget) { TapAt(doc, w.Elem().Bounds.Center()) }
```

```go
// examples/harness/testkit.go#Interactive
// Interactive returns the visible elements that react to touch.
func Interactive(doc *widget.Document) []widget.Widget {
	var out []widget.Widget
	doc.Walk(func(w widget.Widget) bool {
		if _, ok := w.(widget.PointerHandler); ok && w.Elem().Visible() {
			out = append(out, w)
		}
		return true
	})
	return out
}
```

- `newDoc` monta um documento com teclado falso e já faz o layout, então cada elemento tem o seu `Bounds`.
- `tapAt` é um toque completo (down e up) num ponto em **pixels de design**.
- `tap` toca o **centro** de um elemento — o mesmo lugar em que um dedo tocaria.
- `interactive` lista os elementos que reagem a toque.

## Testar um botão

```go
// examples/recipes/recipes_test.go#TestHello
func TestHello(t *testing.T) {
	closed := false
	doc, _ := harness.NewDoc(Hello(func() { closed = true }))
	for _, w := range harness.Interactive(doc) {
		harness.Tap(doc, w)
	}
	if !closed {
		t.Error("the button must call onClose")
	}
}
```

## Testar um formulário com o teclado

Defina `kb.Reply` com o que o "usuário" digita e toque no campo:

```go
// examples/recipes/recipes_test.go#TestLoginFormValidatesBeforeCallingOnLogin
func TestLoginFormValidatesBeforeCallingOnLogin(t *testing.T) {
	var user string
	var remember bool
	f := NewLoginForm(func(u string, r bool) { user, remember = u, r })
	doc, kb := harness.NewDoc(f.Root)
	submit := harness.Interactive(doc)[len(harness.Interactive(doc))-1] // the last element is the button

	harness.Tap(doc, submit)
	if f.Message.Hidden || f.Message.Text != "Type your user name" || user != "" {
		t.Fatalf("empty user: message=%q hidden=%v", f.Message.Text, f.Message.Hidden)
	}

	kb.Reply = "ana"
	harness.Tap(doc, f.User)
	kb.Reply = "123"
	harness.Tap(doc, f.Password)
	harness.Tap(doc, submit)
	if f.Message.Text != "The password needs at least 4 characters" {
		t.Fatalf("short password: %q", f.Message.Text)
	}

	kb.Reply = "1234"
	harness.Tap(doc, f.Password)
	harness.Tap(doc, f.Remember) // untick "Remember me"
	harness.Tap(doc, submit)
	if user != "ana" || remember || !f.Message.Hidden {
		t.Errorf("login: user=%q remember=%v hidden=%v", user, remember, f.Message.Hidden)
	}
}
```

Para simular o usuário cancelando o teclado, defina `kb.Cancel = true`: o campo mantém o valor.

## Testar que o toque cai no elemento

A regra do projeto é que **um toque no centro de cada elemento interativo deve chegar nesse elemento** (nada o cobre, nada o
corta). O teste da demo aplica isso a todas as páginas:

```go
for _, w := range interactive(doc) {
	if w.Elem().Disabled {
		continue // um elemento desabilitado ignora o toque corretamente
	}
	if got := doc.TargetAt(w.Elem().Bounds.Center()); got != w {
		t.Errorf("um toque no centro de %T chega em %T", w, got)
	}
}
```

`doc.TargetAt(ponto)` devolve o elemento que receberia um toque (popups primeiro).

## Testar o layout

Depois de `doc.Layout()` (ou `newDoc`) cada elemento tem `Bounds` (`X`, `Y`, `W`, `H`, em pixels de design):

```go
doc, _ := newDoc(widget.NewRow(12, a, b))
if b.Bounds.X <= a.Bounds.Right() {
	t.Error("b deve ficar à direita de a")
}
```

## Testar o que é desenhado

```go
img := doc.Render() // *image.Gray

pretos := 0
for _, p := range img.Pix {
	if p < 128 {
		pretos++
	}
}
if pretos == 0 {
	t.Error("nada foi desenhado")
}
```

Para ver com os olhos, use `harness.SavePNG(doc, "tela.png")` e abra o arquivo.

## Popups

Combo-boxes e menus abrem um popup. Toque no elemento e use `doc.HasPopup()`; toque de novo fora (ou chame
`doc.ClosePopups()`) para fechar:

```go
tap(doc, combo)
if !doc.HasPopup() {
	t.Fatal("a lista deve abrir")
}
doc.ClosePopups()
```

## Exemplos da documentação

O `go test ./tools/docsync` falha quando uma página de `docs/` deixa de bater com o código de `examples/`. Corrija com
`go run ./tools/docsync`.

## Cobertura e qualidade

```sh
go test -cover ./...
.\scripts\sonar.ps1        # cobertura + análise estática no SonarQube local
```

O código que só roda no leitor (`cmd/`, `internal/device/inkview`) fica fora da cobertura; mantenha a sua lógica em pacotes que rodam
no PC e a cola do leitor bem fina.

Próxima: [Solução de problemas →](11-solucao-de-problemas.md)

## Testando num leitor emulado (ARM)

Tudo o que não toca a tela pode ser testado no processador do próprio leitor, sem o leitor: o `.\scripts\arm-test.ps1` compila os testes de
todo pacote em Go puro (`internal/**`) para **ARMv7 com o Go 1.17 do SDK** e os executa com o `qemu-arm` num contêiner Alpine (sem privilégios e
sem mexer no Docker). O contêiner **não tem zoneinfo nem bibliotecas do sistema**, como o leitor, então um teste que esqueceu de levar o que precisa
falha ali: foi assim que se descobriu a falta do banco de fusos no domínio do calendário.

| O que ele confere | O que ele não confere |
|---|---|
| o código compila e roda em ARM com o Go do SDK (SQLite, o cofre AES-GCM, MIME, iCalendar, horário de verão, renderização) | o InkView (precisa da `libinkview` e da tela) |
| o comportamento sem zoneinfo e sem biblioteca C | o sistema de arquivos FAT do armazenamento do leitor (precisa de uma montagem privilegiada) |
| | velocidade e memória reais: a emulação é lenta e não diz nada do painel e-ink |

Os 44 pacotes passam assim. Um teste do app inteiro no aparelho continua sendo o único jeito de conferir a tela, o toque, o Wi-Fi e o volume FAT.
