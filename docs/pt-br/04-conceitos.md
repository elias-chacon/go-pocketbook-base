# 4. Conceitos

[← Build e versionamento](03-build-e-versionamento.md) · [Índice](README.md) · [Próxima: Layout →](05-layout.md)

Esta página explica como as peças se encaixam, para o resto da documentação fazer sentido.

## As camadas

```
as suas telas (examples/, cmd/…)         o que você escreve
        │
internal/widget                          elementos, layout, despacho de toque
        │
internal/gfx  ·  internal/geom           desenho vetorial, geometria
        │
internal/inkview                         o leitor: eventos, tela, teclado (cgo)
```

`widget` e `gfx` são **Go puro**: rodam no seu PC, e é por isso que toda a interface pode ser testada e visualizada
sem aparelho. Só o `inkview` precisa do leitor (no desktop ele tem versões vazias que não fazem nada).

## Document, Widget e Base

- Um **`Document`** (documento) é dono de uma árvore de elementos. Ele organiza a árvore, desenha numa imagem e
  distribui o toque. Crie com `widget.NewDocument(larguraDaTela, alturaDaTela)` e dê uma raiz com `SetRoot`.
- Um **`Widget`** é qualquer elemento. A interface é minúscula:

  ```go
  type Widget interface {
      Elem() *Base                    // as propriedades comuns
      Measure(avail float64) geom.Size // tamanho natural do conteúdo (elementos folha)
      Paint(c *gfx.Canvas)             // desenha o conteúdo do próprio elemento
  }
  ```
- **`Base`** é embutida em todo elemento e dá a ele as propriedades de um elemento web:

  | Propriedade / método | Significado |
  |---|---|
  | `ID`, `ClassName` | identificação, como `id` e `class` |
  | `Style` | layout e aparência (veja [Layout](05-layout.md)) |
  | `Hidden`, `Disabled` | oculto some; desabilitado fica apagado e ignora o toque |
  | `OnClick` | callback do toque (sobe para os ancestrais) |
  | `Bounds` | onde o elemento está na tela, definido pelo layout |
  | `AppendChild`, `RemoveChild`, `ClearChildren`, `Children`, `Parent` | a árvore |
  | `Document()`, `Invalidate()` | o documento dono; pedir um novo desenho |

  Os elementos se aninham: um `Card` contém um `Label`, um `Input` e um `Button`; uma `Row` contém cards… A árvore
  tem uma única raiz.

## Coordenadas: pixels de design

Todos os tamanhos são escritos em **pixels de design**, em que a tela tem **1404 px de largura** (a do PB741). Numa
tela de outra resolução o documento calcula uma escala (`largura do aparelho / 1404`) e converte tudo, então o mesmo
código fica igual em qualquer leitor:

```go
doc := widget.NewDocument(758, 1024) // um leitor menor: Scale = 0,54, altura de design = 1896
```

- `doc.W` e `doc.H` são o tamanho de design (use-os para dimensionar);
- `doc.Scale` converte para pixels reais;
- as coordenadas de toque chegam em pixels reais e são convertidas de volta, então **desenho e toque sempre usam o
  mesmo espaço**: o que você vê é exatamente onde você toca.

Os tamanhos da biblioteca respeitam o e-ink: os controles têm pelo menos **84 px** de altura (cerca de 9 mm no PB741)
e a menor área de toque é de **64 px**.

## Do código aos pixels

Toda vez que a tela precisa mudar, o documento executa três passos (`doc.Render()`):

1. **Layout** — cada elemento recebe o seu retângulo (`Bounds`) do motor flex/grid.
2. **Desenho** — a árvore é desenhada em profundidade: primeiro a caixa de cada elemento (fundo, borda) a partir do
   `Style`, depois o conteúdo (`Paint`), depois os filhos. Os popups abertos são desenhados por último, por cima.
3. **Imagem em tons de cinza** — o resultado é uma imagem de 8 bits; no leitor ela é copiada para a tela e atualizada.

```go
img := doc.Render() // *image.Gray, do tamanho do aparelho; salve como PNG ou envie ao e-ink
```

O visual é em tons de cinza de propósito: alto contraste, bordas nítidas, sem animação. As cores são `widget.Ink`
(preto), `Paper` (branco), `Soft`, `Mid`, `Muted` e `Disabled` (cinzas).

## Do toque aos callbacks

O toque entra por `doc.HandlePointer(fase, x, y)` com três fases: **down** (encostou), **move** (arrastou) e **up**
(soltou).

1. No **down**, o documento acha o elemento mais profundo sob o dedo que reage a toque (um elemento com método
   `HandlePointer` ou com `OnClick`). Um elemento **desabilitado ou oculto** nunca o recebe.
2. **Move** e **up** vão para esse mesmo elemento (é assim que um slider pode ser arrastado).
3. Se o dedo é solto sobre o elemento em que começou, é gerado um **click**: o elemento executa a lógica dele (um
   check-box alterna, um combo-box abre…) e depois o `OnClick` roda no elemento e em cada ancestral até a raiz
   (**bubbling**), até alguém chamar `e.StopPropagation()`.
4. O documento pede um novo desenho.

Os popups (listas do combo-box, menus) são desenhados acima de tudo. Enquanto um está aberto, um toque **fora** dele
apenas o fecha e nunca chega ao elemento que está por baixo.

## Digitar: o teclado do sistema

Os campos de entrada não desenham teclado. Quando tocados, pedem ao `Keyboard` do documento que abra o teclado
do próprio leitor:

```go
type Keyboard interface {
    Request(widget.KeyboardRequest) // Title, Initial, Kind (texto/numérico), MaxLen, Done(text, ok)
}
```

No leitor o app passa o teclado do InkView; em testes e previews você passa um `FakeKeyboard`, que responde na hora.
Um campo em edição fica *focado* (borda mais grossa) até o teclado fechar.

## O app no leitor

`cmd/hello` e `examples/newapp` são uma cola fina entre o InkView e o documento:

| Evento do InkView | O que a cola faz |
|---|---|
| início | desliga o painel do sistema, lê o tamanho da tela, monta o documento |
| exibir | `doc.Render()` e copia a imagem para a tela |
| toque down / move / up | `doc.HandlePointer(...)` |
| tecla Voltar | fecha um popup, ou sai do app |

Dois detalhes que vale conhecer:

- **Os redesenhos são agrupados.** `doc.OnInvalidate` é chamado a cada mudança, mas a cola enfileira no máximo um
  redesenho por vez e ignora os eventos *move* intermediários enquanto a tela alcança. Isso mantém o arrasto do
  slider fluido no e-ink.
- **Mude a tela dentro dos handlers de evento.** Os callbacks rodam na thread principal do leitor, que é o lugar seguro
  para modificar elementos. Esta base ainda não tem API de timer nem de tarefas em segundo plano.

## Por que foi feito assim

- **Interfaces pequenas** (`Widget`, `PointerHandler`, `Keyboard`) para uma tela nunca depender do hardware.
- **Um único motor de layout** para tudo; cada componente só diz o tamanho que quer e como se desenha.
- **Elementos como objetos simples** com nomes de web, prontos para um futuro front-end em HTML/CSS.
- **Testes de primeira classe**: a precisão do toque e o layout são verificados automaticamente.

Próxima: [Layout →](05-layout.md)
