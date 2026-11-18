# 15. O Launcher

[← Arquitetura (DDD)](14-arquitetura.md) · [Índice](README.md)

O `apps/launcher` é o App Launcher do ecossistema: a tela inicial dos apps feitos sobre esta base. É um app como os outros
(segue o [contrato](13-apps-e-o-contrato.md)) e a raiz de composição do contexto `internal/launcher`.

> **Estado.** A tela inicial, a execução de um app, a busca e o catálogo do GitHub com instalação por toque estão prontos no
> código (o catálogo e o download ainda não foram verificados no leitor). A tela de configurações vem a seguir.

## O que faz hoje

- Lista os apps **instalados** no leitor numa grade paginada de **ícone + título** (4 × 5 blocos por página). Os apps são os que
  deixaram um registro em `applications/.pbgo` na primeira vez que rodaram (veja o contrato); o Launcher não lista a si mesmo.
- Tocar num bloco **executa o app**. As teclas esquerda e direita viram a página (as setas de baixo também).
- Quando o app pede para sair (o menu *Sair*, uma tecla Voltar) o leitor **volta ao Launcher**; o Home sempre vai para a tela
  inicial do sistema.
- Um **campo de busca** filtra em tempo real (a cada mudança do texto) por título, descrição e id, sem ligar para maiúsculas e
  acentos. Ao buscar, a resposta tem duas seções: `-- Instalados --` (os blocos que combinam, 3 linhas por página) e
  `-- Disponíveis --` (os apps do catálogo que combinam e ainda não estão instalados, até 4 linhas e "… e mais N"). Sem catálogo
  (ou sem rede) a segunda seção diz o motivo e os instalados continuam respondendo.
- **Instalação com um toque.** As linhas de *Disponíveis* vêm da **última Release do GitHub** do repositório
  (`elias-chacon/go-pocketbook-base`): cada app é publicado como `<id>.app` + `<id>.json`, e o catálogo lista os pares cujo manifesto
  segue o contrato. Tocar numa linha mostra *Baixando…*, baixa o `.app` para `applications/`, confere o tamanho, registra na tela
  inicial (com o ícone padrão: o app só desenha o próprio se for executado, e a tela inicial mantém o primeiro ícone) e o move
  para *Instalados*. A lista fica guardada por 5 minutos. Precisa do Wi-Fi ligado; sem ele (ou sem release) a seção mostra o erro.
- Textos em português (pt-BR) com inglês como fallback, no idioma escolhido nas configurações compartilhadas.

## Como um app é executado, e como volta

O leitor tem uma tela e uma sessão do InkView, então o Launcher não executa o app ao lado dele: ele **se substitui por ele**
(`execve`), no mesmo processo, e avisa ao app quem o iniciou pela variável de ambiente `PBGO_LAUNCHER` (o caminho do Launcher).
Quando o app pede para sair, a plataforma vê a variável e inicia o Launcher de volta, de novo no lugar. Um app iniciado pela tela
inicial não tem a variável e apenas termina; apertar Home nunca volta, porque essa decisão é do sistema, não do app. O Launcher
nunca volta para si mesmo (ele herda a variável do app que voltou).

```
Launcher ──exec──► app          (PBGO_LAUNCHER=/…/launcher.app)
   ▲                 │
   └───────exec──────┘          quando o app pede para sair
```

## O código (contexto `launcher`)

| Camada | Pacote | Guarda |
|---|---|---|
| domain | `internal/launcher/domain` | `Tile`, `TilesFrom` (traduzido, ordenado sem ligar para maiúsculas e acentos), `Grid` e paginação, `Matches`/`FilterTiles`/`FilterListings`, `Listing`, `SearchResult`, as portas `AppLauncher`, `Catalog` e `Installer` |
| application | `internal/launcher/application` | `Home`: os blocos dos apps instalados e a abertura de um; `Search`: apps instalados e disponíveis para uma busca |
| infrastructure | `internal/launcher/infrastructure/processlauncher` | inicia um app substituindo o processo |
| | `…/infrastructure/github` | o `Catalog`: os pares `<id>.app` + `<id>.json` da última release (guardados por 5 min) |
| | `…/infrastructure/appinstaller` | o `Installer`: baixa por um arquivo temporário, confere o tamanho, registra |
| presentation | `internal/launcher/presentation` | a tela inicial, com os widgets do toolkit (`Image`, `Grid`, `Label`…) |
| raiz de composição | `apps/launcher` | liga a plataforma ao contexto; carrega os ícones BMP |

Os ícones são os BMPs que cada app gravou ao se registrar; quando falta um, o Launcher desenha o ícone padrão (a inicial do
título).

## Experimentando no leitor

```powershell
.\build.ps1 -All                       # compila todos os apps de apps/ (um único incremento de versão)
.\scripts\install.ps1 -Drive F: -All    # copia para o leitor
```

Cada app se registra na primeira vez que roda (abra-o uma vez pela lista de *Aplicativos*), ou use o [instalador](16-instalador.md), que registra todos enquanto copia. Depois abra o Launcher: ele lista os outros.

[← Arquitetura (DDD)](14-arquitetura.md) · [Índice](README.md)
