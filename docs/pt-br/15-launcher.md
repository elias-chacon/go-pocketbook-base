# 15. O Launcher

[← Arquitetura (DDD)](14-arquitetura.md) · [Índice](README.md)

O `apps/launcher` é o App Launcher do ecossistema: a tela inicial dos apps feitos sobre esta base. É um app como os outros
(segue o [contrato](13-apps-e-o-contrato.md)) e a raiz de composição do contexto `internal/launcher`.

> **Estado.** A tela inicial (a grade de apps instalados) e a execução de um app estão prontas. A busca com as seções
> *Installed / Available*, o catálogo do GitHub e a tela de configurações vêm a seguir.

## O que faz hoje

- Lista os apps **instalados** no leitor numa grade paginada de **ícone + título** (4 × 5 blocos por página). Os apps são os que
  deixaram um registro em `applications/.pbgo` na primeira vez que rodaram (veja o contrato); o Launcher não lista a si mesmo.
- Tocar num bloco **executa o app**. As teclas esquerda e direita viram a página (as setas de baixo também).
- Quando o app pede para sair (o menu *Sair*, uma tecla Voltar) o leitor **volta ao Launcher**; o Home sempre vai para a tela
  inicial do sistema.
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
| domain | `internal/launcher/domain` | `Tile`, `TilesFrom` (traduzido, ordenado sem ligar para maiúsculas e acentos), `Grid` e paginação, a porta `AppLauncher` |
| application | `internal/launcher/application` | `Home`: os blocos dos apps instalados e a abertura de um |
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

O `install.ps1` copia os apps **e os registra** (ícones, entrada no `view.json` e o registro que o Launcher lê: `tools/register`). Ejete, **desconecte o cabo** e os ícones aparecem na tela inicial; abra o Launcher e ele lista os outros. Um app copiado à mão continua se registrando sozinho na primeira execução.

[← Arquitetura (DDD)](14-arquitetura.md) · [Índice](README.md)
