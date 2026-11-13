# 7. Eventos e estado

[← Componentes](06-componentes.md) · [Índice](README.md) · [Próxima: Receitas →](08-receitas.md)

## Visão geral dos callbacks

Todos os callbacks têm o mesmo formato e são campos simples, como o `onclick` do JavaScript:

```go
type Handler func(e *widget.Event)

type Event struct {
    Target widget.Widget // o elemento a que o evento se refere
    X, Y   float64       // posição do toque em pixels de design (eventos de click)
}
func (e *Event) StopPropagation()
```

| Callback | Elementos | Quando |
|---|---|---|
| `OnClick` | todos os elementos | o elemento foi tocado (**sobe** para os ancestrais) |
| `OnChange` | check-box/radio/switch, combo-box, campos, text-area, slider, chip, tabela (`OnSelect`), calendário | o valor mudou pelo usuário |
| `OnInput` | slider | enquanto arrasta |
| `OnSubmit` | campo de busca | a lupa foi tocada ou OK no teclado |
| `OnSelect` | dropdown, tabela, `Menu`, `MenuItem` | um item ou linha foi escolhido |
| `OnClose` | chip | o × foi tocado |

## Lendo valores

O elemento também é o modelo: leia as propriedades dentro do callback (ou a qualquer momento).

| Elemento | Leia |
|---|---|
| campo, text-area | `Value`; para números `Number()`, para máscaras os dígitos em `Value` e o texto em `Display()` |
| check-box, radio, switch | `Checked` (radio: `Value` é o valor da opção) |
| combo-box | `Value()`, `Index` |
| slider | `Value` |
| chips | `chip.Selected`, `grupo.SelectedLabels()` |
| tabela | `Selected` (índice em `Rows`), `Rows[Selected]` |
| calendário | `Selected` (`time.Time`, zero quando não há) |

`e.Target` é o elemento, então um único handler pode servir a vários:

```go
for _, r := range []*widget.Toggle{free, pro} {
	r.OnChange = func(e *widget.Event) {
		fmt.Println("plano:", e.Target.(*widget.Toggle).Value)
	}
}
```

## Mudando a tela

Mude as propriedades e o próximo quadro as mostra. **O toque já pede um novo desenho**, então dentro dos callbacks basta
atribuir:

```go
label.Text = "Salvo"          // texto
errorLabel.Hidden = false     // mostrar / ocultar
button.Disabled = true        // apagar
slider.SetValue(70)           // mover o botão
table.Rows = novasLinhas      // novos dados
table.Page, table.Selected = 0, -1
```

Fora de um callback (por exemplo, depois que o seu código terminou um trabalho) chame `doc.Invalidate()` — ou
`elemento.Invalidate()` — para pedir um novo desenho.

### Adicionando e removendo elementos

```go
lista.AppendChild(widget.NewChip("novo"))   // adiciona no fim
lista.RemoveChild(chip)                     // remove um
lista.ClearChildren()                       // remove todos (útil para trocar de página)
```

Trocar de página é só isso: limpar o contêiner e acrescentar a outra página. A
[receita de navegação](08-receitas.md#navegação-entre-páginas-com-a-menu-bar) faz exatamente isso.

### Encontrando elementos

```go
doc.GetElementByID("salvar")         // como document.getElementById
doc.QuerySelectorAll("perigo")       // todo elemento com ClassName "perigo"
doc.Walk(func(w widget.Widget) bool { /* visita todos */ return true })
```

Defina `ID` e `ClassName` nos elementos que você precisará encontrar depois.

## Interrompendo o bubbling

Um toque num botão dentro de um card executa o `OnClick` do botão, depois o do card e depois o da página. Interrompa
quando o pai não deve reagir:

```go
button.OnClick = func(e *widget.Event) {
	e.StopPropagation() // o OnClick do card não vai rodar
}
```

## Desabilitado e oculto

- `Disabled = true`: desenhado apagado, **não recebe toque**, e os descendentes também não.
- `Hidden = true`: não é desenhado, não ocupa espaço e **não recebe toque**.

## Popups e o teclado

- Combo-boxes, dropdowns e menu bars abrem um **popup** acima de tudo. Um toque fora só o fecha. `doc.HasPopup()` diz se há
  um aberto e `doc.ClosePopups()` o fecha (a tecla Voltar faz isso).
- Os campos de entrada chamam `doc.RequestText(...)`, que usa `doc.Keyboard`. `doc.Focused()` devolve o campo em edição,
  ou nil.
- No leitor, a tecla **Voltar** chega pelo `inkview` (`KeyBack`); a cola do app decide o que ela faz.

## Avançado: toque bruto

Normalmente você não precisa disto, mas um elemento próprio pode receber as três fases do toque:

```go
func (t *TapCounter) HandlePointer(e *widget.PointerEvent) {
	// e.Type é PointerDown, PointerMove, PointerUp ou PointerClick
}
```

Veja [Componentes próprios](09-componentes-proprios.md).

Próxima: [Receitas →](08-receitas.md)
