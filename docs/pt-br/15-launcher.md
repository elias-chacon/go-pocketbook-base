# 15. O Launcher

[← Arquitetura (DDD)](14-arquitetura.md) · [Índice](README.md)

O `apps/launcher` é o App Launcher do ecossistema: a tela inicial dos apps feitos sobre esta base. É um app como os outros
(segue o [contrato](13-apps-e-o-contrato.md)) e a raiz de composição do contexto `internal/launcher`.

> **Estado.** A tela inicial com todos os apps do catálogo, a execução de um app, a instalação por toque, a atualização por checksum, esconder e remover estão prontos no
> código (o catálogo, o download, a atualização, a remoção e o toque longo ainda não foram verificados no leitor).

## O que faz hoje

- Lista **todos os apps do ecossistema** numa grade paginada de **ícone + título** (4 × 5 blocos por página): os apps **instalados** no leitor (os que
  deixaram um registro em `applications/.pbgo` na primeira vez que rodaram) **e os apps que só o catálogo tem**, tirados da **última Release do GitHub**
  (`catalog.json`, guardado no leitor em `applications/.pbgo/cache/catalog.json`, então aparecem na hora e offline; o Launcher consulta o GitHub em segundo
  plano ao abrir e redesenha quando a resposta chega). O Launcher não lista a si mesmo.
- Tocar num bloco de app instalado **executa o app**; tocar num **não instalado** (diz *Toque para instalar*) **instala**: *Baixando…*, o zip, a
  conferência de tamanho e SHA-256, o `<id>.app` extraído para `applications/` por um arquivo temporário e registrado na tela inicial (com o ícone que o app
  publicou como `icon.png`, ou o padrão). As teclas esquerda e direita viram a página (as setas de baixo também).
- Um **toque longo (2 segundos)** num bloco abre a janela do app: **Atualizar** (só aparece quando o app está instalado e o **SHA-256 do arquivo dele é
  diferente do `app_sha256` da release**: quem decide é o checksum, nunca a versão), **Instalar** (quando falta), **Esconder da tela inicial** (instalado ou não) e
  **Remover do leitor**, que após uma confirmação apaga o app e tudo que ele deixou: a entrada no `view.json` e os ícones, o programa, o registro e o log, os
  arquivos que o manifesto lista em `data` (o banco dele), as tabelas dele no banco compartilhado (o arquivo some quando nenhum outro app tem tabelas nele) e os
  arquivos que ele `shares` com outros apps (as contas) só quando nenhum outro app instalado os compartilha.
- Quando o app pede para sair (o menu *Sair*, uma tecla Voltar) o leitor **volta ao Launcher**; o Home sempre vai para a tela
  inicial do sistema.
- Um **campo de busca** filtra a grade inteira em tempo real (a cada mudança do texto) por título, descrição e id, sem ligar para maiúsculas e acentos.
- As **Configurações** também listam os apps: **Gerenciar apps** (uma `Table` dos apps visíveis, instalados ou não; selecione um e abra a janela dele, como o toque
  longo faz) e **Apps escondidos** (uma `Table` dos escondidos, com **Mostrar de novo o app selecionado**). Os ids escondidos ficam em `launcher.json`.
- **Configurações** (o botão *Configurações* no rodapé, ou a tecla Menu; a tecla esquerda sai sem salvar): a **grade** (colunas e linhas, de 1 a 8), o **tamanho dos ícones**
  (96 a 240), o **idioma** (compartilhado por todos os apps: os outros o leem ao abrir) e uma **imagem de fundo**, escolhida entre as imagens
  (PNG, JPG, GIF, BMP) que a pessoa copiou para `applications/backgrounds` no leitor. As preferências ficam em `applications/.pbgo/launcher.json`; a imagem
  é reduzida à tela ao carregar, e os blocos mantêm o fundo branco para continuar legíveis sobre ela.
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
| domain | `internal/launcher/domain` | `Tile`, `TilesFrom` (traduzido, ordenado sem ligar para maiúsculas e acentos), `Grid` e paginação, `AppsOf` (instalados + catálogo, ordenados), `Visible`/`Hidden`, `Matches`/`FilterTiles`, `Listing` (com `AppSHA256`), `Preferences` (grade, tamanho do ícone, fundo e apps escondidos, sempre dentro dos limites), as portas `AppLauncher`, `Catalog`, `Installer`, `Uninstaller` e `PreferencesStore` |
| application | `internal/launcher/application` | `Home`: os blocos dos apps instalados e a abertura de um; `Search`: apps instalados e disponíveis para uma busca |
| infrastructure | `internal/launcher/infrastructure/processlauncher` | inicia um app substituindo o processo |
| | `…/infrastructure/github` | o `Catalog`: o `catalog.json` da última release e os apps que ele lista (guardados por 5 min) |
| | `…/infrastructure/prefsfile` | o `PreferencesStore`: `applications/.pbgo/launcher.json` |
| | `…/infrastructure/backgrounds` | lista as imagens de `applications/backgrounds` |
| | `…/infrastructure/iconcache` | baixa os ícones (PNG) do catálogo e os guarda no leitor |
| | `…/infrastructure/appinstaller` | o `Installer`: baixa o zip, confere tamanho e SHA-256, extrai o `<id>.app` por um arquivo temporário, registra |
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
