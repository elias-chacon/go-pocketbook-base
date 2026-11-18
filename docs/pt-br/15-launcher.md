# 15. O Launcher

[← Arquitetura (DDD)](14-arquitetura.md) · [Índice](README.md)

O `apps/launcher` é o App Launcher do ecossistema: a tela inicial dos apps feitos sobre esta base. É um app como os outros
(segue o [contrato](13-apps-e-o-contrato.md)) e a raiz de composição do contexto `internal/launcher`.

> **Estado.** A tela inicial (a grade de apps instalados), a execução de um app e a busca estão prontas. O catálogo do GitHub
> (o que preenche *Disponíveis*) e a tela de configurações vêm a seguir.

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
| domain | `internal/launcher/domain` | `Tile`, `TilesFrom` (traduzido, ordenado sem ligar para maiúsculas e acentos), `Grid` e paginação, `Matches`/`FilterTiles`/`FilterListings`, `Listing`, `SearchResult`, as portas `AppLauncher` e `Catalog` |
| application | `internal/launcher/application` | `Home`: os blocos dos apps instalados e a abertura de um; `Search`: apps instalados e disponíveis para uma busca |
| infrastructure | `internal/launcher/infrastructure/processlauncher` | inicia um app substituindo o processo |
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
