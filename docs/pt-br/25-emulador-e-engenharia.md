# 25. O emulador do firmware e o app Engenharia

Dois recursos para testar sem depender do aparelho a cada ideia: o **emulador `pbemu`**, que roda o firmware real do PB741 no PC, e o
app **Engenharia** (`apps/eng`, contexto `internal/engineering`), um menu de testes da tela, das cores, do toque e da velocidade que
grava tudo num registro.

> **O que foi verificado e o que não foi.** O emulador roda o firmware 6.8.4079 do PB741 de verdade (a `libinkview`, a tela inicial, o
> `monitor`), e os apps desta base rodam nele: registram o ícone, aparecem em Apps, abrem, desenham em 24 bits e respondem a toque. Mas
> o emulador **não tem painel de tela**: o ioctl que manda a atualização à tela é ignorado, então **não mostra o aspecto do A2, das
> cores do painel Kaleido nem o tempo real de uma atualização**. Esses números só saem do aparelho, com o app Engenharia.

## O emulador (`scripts/pbemu.ps1`)

O [pbemu](https://codeberg.org/datyoma/pbemu) é um emulador em modo usuário: os processos ARM do firmware (o `monitor.app`, os apps) rodam
sob `qemu-arm` num contêiner, e um shim com `LD_PRELOAD` finge o hardware (framebuffer, RTC, entradas de toque). O firmware é o
oficial da PocketBook, **baixado na sua máquina e nunca commitado** (a licença é deles).

```powershell
.\scripts\pbemu.ps1 setup                     # uma vez: clona o pbemu, baixa o firmware do PB741 (378 MB), monta a imagem
.\scripts\pbemu.ps1 start                     # liga o leitor emulado
.\build.ps1 -App hello                        # gera o .app para ARM
.\scripts\pbemu.ps1 install hello             # copia para o leitor emulado
.\scripts\pbemu.ps1 stop; .\scripts\pbemu.ps1 start   # a tela inicial só relê a lista de apps ao ligar
.\scripts\pbemu.ps1 tap 1200 1560             # Apps; depois toque no ícone do app
.\scripts\pbemu.ps1 shot tela.png             # a tela, em PNG (1404 x 1872)
.\scripts\pbemu.ps1 swipe 700 1200 700 400    # arrasta o dedo
.\scripts\pbemu.ps1 log eng                   # os arquivos de registro do app no leitor emulado
.\scripts\pbemu.ps1 crash                     # o último `panic` de Go que o `monitor.log` guardou
```

Requisitos: Windows com WSL2 (com WSLg não é preciso: o emulador roda sem janela e a tela sai por `shot`) e Docker dentro do WSL. O
`setup` usa o commit do pbemu que foi testado (`scripts/pbemu/setup.sh`), traduz as chamadas `podman` do pbemu para `docker`
(`scripts/pbemu/podman-shim.sh`) e compila uma sonda `fb_pgm` que grava o framebuffer. O WSL desliga a VM quando não há sessão aberta e
leva o contêiner junto, por isso o `start` deixa uma sessão `sleep` escondida.

O que serve: verificar o registro do ícone, o ciclo de vida, o desenho em cores, o toque, as intenções entre apps, o disco (a pasta
`applications/.pbgo` do leitor emulado fica em `~/pbemu-work/pbemu/U741_6.8.4079/.live/mnt/ext1`) e **achar crashes que só aparecem
com a `libinkview` de verdade** (foi assim que se achou um estouro de índice no desenho colorido). O que não serve: velocidade e aspecto
da tela de e-ink.

## O app Engenharia

Aparece no Launcher e em Apps como **Engenharia** (`eng.app`, como todo app da Release). Ele **não faz nada sozinho**: cada item do
menu é um teste, e todo número medido vai para `applications/.pbgo/eng.log`, **uma linha JSON por evento** (`t`, `kind` e os campos),
para ler pelo USB e analisar depois.

| Item | O que faz | O que grava |
|---|---|---|
| **Cores** | 8 imagens em tela cheia: barras de cor com 16 cinzas, primárias, 16 tons de cinza, rampa de 256 cinzas, matizes, saturação, branco, preto | `pattern`: o modo, o tempo para gerar, para desenhar e para a tela assentar |
| **Tela** | fundos lisos, xadrez de 1 px e de 8 px, linhas de 1 px, grade de 100 px com bordas, diagonais e cantos (pixels mortos, fantasmas, geometria) | `pattern` |
| **Toque** | uma área que desenha o caminho do dedo e mostra eventos, eventos por segundo, a maior pausa e os pixels | `touch` (fase, x, y e o tempo desde o evento anterior) e `touch_stroke` (o resumo do traço) |
| **Modos de atualização** | redesenha a metade de baixo da tela em cada modo do InkView (`partial`, `dynamic`, `a2`, `du4`, `bw`, `hq`, `full`) e mede a chamada, o tempo até a tela ficar pronta e até assentar; *Amostras por modo* mostra imagens em cada modo para comparar a olho | `bench` (cada chamada) e `bench_mode` (média, p95, mínimo, máximo, total e assentar) |
| **Rolagem** | uma lista de 300 linhas que se arrasta com o dedo; o botão *Modo* escolhe como a tela é atualizada durante a rolagem e *Rolar sozinho* faz 40 passos de 160 px medindo cada um | `scroll` (cada mudança por toque, com o intervalo) e `scroll_step`/`scroll_run` (média, p95, quadros por segundo) |
| **Sistema** | modelo, profundidade do hardware, arquivo da forma de onda, canvas, tela, versão, espaço livre e a pasta do registro | `system` |
| **Registro** | as últimas linhas do registro e um botão para limpá-lo | |

Toque numa imagem em tela cheia: o canto superior esquerdo volta, o canto superior direito troca o modo de atualização, o terço
esquerdo volta uma imagem e o resto avança. A legenda no canto inferior esquerdo diz qual imagem e qual modo. As teclas esquerda e Menu
do aparelho também voltam. O registro começa de novo quando passa de 4 MB, e cada evento chega ao arquivo na hora (menos `touch` e
`scroll`, que esperam o fim do teste para não atrasar o que se mede), então o que foi medido antes de uma queda não se perde.

### O que o app dá de novo à plataforma

- `platform.Env.Panel` (e `platform.Panel`): o painel além do repaint da plataforma. Desenha uma imagem **cinza ou colorida** (24 bits,
  na ordem de bytes B, G, R, confirmada contra o firmware do PB741 no emulador) e atualiza uma região com um modo do InkView
  (`UpdatePartial`, `UpdateDynamic`, `UpdateA2`, `UpdateDU4`, `UpdateBW`, `UpdateHQ`, `UpdateFull`), espera o painel terminar (`Wait`),
  sai do A2 com uma atualização limpa (`Settle`) e escolhe o modo dos repaints da própria plataforma (`SetMode`).
- `platform.Raw`: um app que às vezes pinta a tela sozinho (as imagens do app Engenharia) implementa `Raw(p Panel) bool`; quando devolve
  `true` a plataforma não pinta por cima.
- Os desenhos recusam um retângulo fora da imagem em vez de ler além do fim do buffer.

### Como analisar o registro

```sh
# o tempo médio de cada modo de atualização
grep bench_mode eng.log
# os quadros por segundo da rolagem em cada modo
grep scroll_run eng.log
# a taxa de eventos do toque
grep touch_stroke eng.log
```

Cada linha é JSON: `jq` serve, ou qualquer leitor de JSON Lines.

### Limites

- Nada foi medido no aparelho ainda: os números que o emulador gera (por exemplo ~44 ms por atualização, iguais em todos os modos)
  vêm da CPU emulada e **não valem**.
- O modo A2 e a atualização por cima de uma camada colorida não foram vistos numa tela de e-ink; o app serve justamente para isso.
- O que o `ScrollView` do SDK da PocketBook faz (A2 durante o arrasto, `DitherAreaPattern2Level`, `ExitUpdateA2` ao soltar) foi lido no
  disassembly da `libinkview.so`; o `Scroll` do toolkit ainda **não** usa A2.
