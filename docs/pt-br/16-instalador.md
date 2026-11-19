# 16. O instalador e as releases

[← O Launcher](15-launcher.md) · [Índice](README.md)

Quem só quer os apps não deveria precisar de Go, Docker ou PowerShell. Cada release publica um **instalador** para Windows,
Linux e macOS: um programa só, com ícone, que leva os apps dentro de si e os coloca no PocketBook.

## Para quem só quer os apps

1. Abra a página de [Releases](https://github.com/elias-chacon/go-pocketbook-base/releases) e baixe o arquivo do seu
   computador: `pbgo-installer_windows_amd64.zip`, `…_linux_amd64.tar.gz`, `…_darwin_arm64.tar.gz` (Apple
   silicon) ou `…_darwin_amd64.tar.gz` (Mac Intel). Descompacte.
2. Conecte o PocketBook pelo cabo USB e escolha, no leitor, o modo de conexão com o PC (*Link com PC* / *PC Link*).
3. Dê dois cliques em **pbgo-installer**. Ele encontra o leitor sozinho, copia os apps (o Launcher e os outros) e coloca os
   ícones na tela inicial do leitor. Se houver mais de um leitor conectado, ele pergunta qual usar.
4. Ejete o leitor com segurança e desconecte o cabo: os ícones aparecem na lista de Aplicativos, no grupo *General*.

O instalador fala o idioma do computador (português ou inglês). Se não achar o leitor, diz o que fazer, e
`pbgo-installer -drive F:\` (ou uma pasta) instala na unidade que você indicar.

| Sistema | O que você pode ver | O que fazer |
|---|---|---|
| Windows | *"O Windows protegeu o computador"* (o programa não é assinado) | **Mais informações → Executar assim mesmo** |
| macOS | *"não pode ser aberto porque o desenvolvedor não pode ser verificado"* | botão direito no arquivo → **Abrir** → **Abrir** |
| Linux | nada | o `tar xzf` mantém o bit de execução; rode `./pbgo-installer` (no gerenciador de arquivos, "Executar") |

## O que ele faz

Para cada app que carrega: grava `applications/<id>.app` no leitor (por um arquivo temporário, então uma cópia interrompida nunca
deixa meio app) e registra o app **do mesmo jeito que o app faz na primeira execução**: os dois ícones, a entrada no
`view.json` (com backup) e o registro em `applications/.pbgo` que o Launcher lê. O registro é o da plataforma
(`platform.RegisterOnDrive`), então há uma única implementação: um app copiado à mão ainda se registra sozinho ao rodar.

## O código (contexto `installer`)

| Camada | Pacote | Guarda |
|---|---|---|
| domain | `internal/installer/domain` | `Reader`, `Package`, `Outcome` e as portas `ReaderFinder`, `PackageSource`, `Deployer` |
| application | `internal/installer/application` | `Install`: achar os leitores, instalar cada pacote (uma falha não impede os outros) |
| infrastructure | `…/infrastructure/drives` | acha um PocketBook: letras de unidade no Windows, `/Volumes` no macOS, `/media`, `/run/media`, `/mnt` no Linux; uma unidade é um leitor quando tem o `view.json` do leitor |
| | `…/infrastructure/embedded` | os apps dentro do executável (`embed.FS` com `<id>.app` + `<id>.json`) |
| | `…/infrastructure/readerfs` | grava o app no leitor e chama o registro |
| | `…/infrastructure/locale` | o idioma do computador |
| presentation | `internal/installer/presentation` | a conversa no console (espera o Enter no fim, para a janela aberta com dois cliques não sumir) |
| raiz de composição | `cmd/pbinstaller` | liga tudo, embute `payload/`, conhece o ícone de cada app |

O contexto não depende da plataforma: a raiz de composição entrega a ele a função de registro, e o teste de arquitetura
(capítulo 14) garante isso.

## Como uma release é feita

```powershell
.\build.ps1 -All          # os apps para o leitor (dist/applications)
go run ./tools/release    # prepara o payload e gera os instaladores em ./release
```

O `tools/release` copia cada `dist/applications/<id>.app` e o seu `apps/<id>/app.json` para `cmd/pbinstaller/payload`, compila o
instalador para Windows (amd64), Linux (amd64) e macOS (amd64, arm64) com a versão no binário, dá ao do Windows o ícone do
Launcher (o `tools/release/syso.go` escreve o arquivo de recursos do Windows por conta própria, então nada é baixado), empacota cada um em zip ou
tar.gz, copia os apps crus (como `<id>.app.bin`, pois o GitHub recusa arquivos de release chamados `*.app`) e os manifestos `<id>.json` (o catálogo do Launcher lista e baixa estes) e escreve o `SHA256SUMS`.

## Integração contínua e releases são dois workflows

| Workflow | Quando | O que faz |
|---|---|---|
| `.github/workflows/ci.yml` | todo push na `main`, todo pull request e antes de uma release | `go vet`, todos os testes no Linux, a cobertura que conta (`tools/coverage`, as mesmas exclusões do Sonar), os dois apps compilados para o leitor; na `main` publica o badge de cobertura (JSON na branch `badges`) |
| `.github/workflows/release.yml` | uma tag `vMAJOR.MINOR.BUILD` é enviada (deve ser igual ao `.version`) | roda o `ci.yml` antes, depois compila **todos os apps** de `apps/`, os instaladores e os checksums, e publica **uma Release do GitHub** com tudo |

A release leva `<id>.app.bin` + `<id>.json` de cada app (o catálogo do Launcher), o instalador de cada sistema com **nome sem versão**
(então `…/releases/latest/download/pbgo-installer_windows_amd64.zip` é sempre o mais novo) e notas com o comando de instalação para
Windows, Linux e macOS, geradas pelo `tools/release`. O README mostra os mesmos comandos para o `latest` e os badges (ci, cobertura,
issues abertas, versão, última atualização).

> **Estado.** Verificado no Windows com um leitor de verdade: o executável mostra o ícone do Launcher, acha o leitor e instala
> (e registra) os apps. Ainda não verificados: os dois workflows no GitHub e os instaladores de Linux e macOS (os testes rodam no Linux, em Docker).
