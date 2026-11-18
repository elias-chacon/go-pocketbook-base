# 14. Arquitetura (DDD)

[← Apps e o contrato](13-apps-e-o-contrato.md) · [Índice](README.md)

O repositório segue o **Domain-Driven Design**, aplicado com pragmatismo: o código é dividido em **contextos
delimitados**, cada um com seu vocabulário e suas camadas, e há **uma regra de dependência** que um teste faz valer. O
mapa e a regra estão escritos como código em `internal/architecture`; esta página os explica.

## Os contextos delimitados

| Contexto | O que é | Pacotes | Pode usar |
|---|---|---|---|
| **kernel** | o núcleo compartilhado: objetos de valor minúsculos que todo contexto precisa | `geom` (Rect, Point, Insets), `version` | nada |
| **toolkit** | o toolkit de UI, o coração da biblioteca | `gfx` (desenho vetorial), `widget` (Document, elementos, layout, toque) | kernel |
| **device** | o leitor como sistema externo (camada anticorrupção) | `inkview` (binding cgo), `desktop` (registro na tela inicial) | kernel, toolkit |
| **localization** | idiomas | `i18n` (pt-BR primeiro, fallback em inglês) | kernel |
| **platform** | o que um app do projeto *é* | `domain` · `application` · `infrastructure` · a fachada `platform` | kernel, localization, toolkit, device |
| **showcase** | o conteúdo do app hello | `showcase` | kernel, localization, toolkit |

Fora do `internal/`: os `apps/<id>` são **raízes de composição** (um por app: ligam os contextos e podem usar qualquer
um), o `examples/` são amostras da documentação e `tools/`, `cmd/` são ferramentas de desenvolvimento.

## Camadas dentro de um contexto: a plataforma

```
infrastructure ──► application ──► domain
 (adaptadores)     (casos de uso)   (regras + portas)
```

| Camada | Guarda | Exemplo |
|---|---|---|
| **domain** | as regras e o vocabulário, sem I/O, mais as **portas** (interfaces de que o domínio precisa) | `Manifest.Validate`, `Manifest.TitleIn(idioma)`, `InstalledApp`, `Settings`; portas `AppRegistry`, `SettingsStore`, `DesktopRegistrar` |
| **application** | os **casos de uso** que orquestram o domínio e o toolkit por meio de portas | `Runtime` (liga os eventos do leitor ao documento do app), `FirstRun` (registra um app e o anota) |
| **infrastructure** | os **adaptadores** que implementam as portas com o mundo real | `jsonfiles` (arquivos JSON em `applications/.pbgo`), `reader` (InkView, teclado do sistema, tela inicial) |

Uma porta é dita nas palavras do domínio e implementada fora: o `domain.AppRegistry` diz "salve um app instalado, liste os
instalados"; o `jsonfiles.Apps` faz com arquivos; um teste faz com uma lista. É por isso que o `Runtime` e o `FirstRun`
são testados no PC com dublês e não precisam de leitor nem de disco.

O pacote `platform` é a **raiz de composição** do contexto: constrói os adaptadores, entrega-os aos casos de uso e
reexporta o que um app precisa, para o app ter um único import.

## Linguagem ubíqua

| Termo | Significado |
|---|---|
| **Document** | a árvore de elementos de uma tela; a raiz do agregado do toolkit |
| **Element** | um nó do Document (botão, input, painel…); implementa `widget.Widget` |
| **Design pixel** | a unidade de toda medida (largura de referência 1404), escalada pelo Document |
| **App** | um programa do ecossistema: uma pasta em `apps/`, um arquivo `<id>.app` |
| **Manifest** | a identidade do app, no `app.json` (id, títulos, descrição) |
| **InstalledApp** | um App que já rodou neste leitor: manifesto + versão + arquivos |
| **Settings** | preferências compartilhadas por todo App (hoje o idioma) |
| **Porta** / **Adaptador** | uma interface que o domínio define / a implementação dela na infraestrutura |
| **Plugin** | um App que o Launcher lista, instala e executa |

## A regra de dependência

As dependências apontam **para dentro**. Um contexto nunca mexe nas entranhas de outro, o domínio só depende do kernel e
do módulo de idiomas, e nada em `internal/` importa `apps/`, `examples/`, `tools/` ou `cmd/`. A tabela exata é a
`architecture.Rules`; o `go test ./internal/architecture` lê todo import do repositório e falha com uma mensagem como:

```
breaks the dependency rule (see internal/architecture): internal/platform/domain/settings.go imports internal/toolkit/widget
```

Um segundo teste falha quando aparece uma pasta nova diretamente em `internal/` sem estar declarada em
`architecture.Contexts`: um contexto novo é uma decisão, tomada de propósito e documentada.

## Onde vai o meu código?

| Estou escrevendo… | Vai em |
|---|---|
| um widget, um recurso de layout, uma propriedade de estilo | `internal/toolkit/widget` (primitivas de desenho em `gfx`) |
| uma regra sobre o que é um app (validação, qual texto para um idioma) | `platform/domain` |
| um passo que combina regras para fazer algo (executar um app, registrá-lo) | `platform/application` |
| ler ou gravar arquivos, falar com o InkView, a rede | `platform/infrastructure` (um binding cru: `device`) |
| a tela de um app | a pasta do app, `apps/<id>` |
| um valor minúsculo usado por todo contexto | `internal/kernel` (mantenha minúsculo) |
| texto que o usuário lê | os `locales/*.json` do app, pt-BR primeiro (veja [i18n](12-internacionalizacao.md)) |

## As amostras seguem o mesmo mapa

O `examples/` é organizado pelo capítulo, isto é, pelo contexto, que cada amostra ensina:

```
examples/components/    capítulo 6: um arquivo por componente (button.go, table.go, modal.go…)
examples/recipes/       capítulo 8: telas completas (login, settings, catalog, order…)
examples/custom/        capítulo 9: o seu próprio elemento
examples/translation/   capítulo 12: i18n com arquivos JSON
examples/harness/       o que as amostras e seus testes compartilham: Screen, SavePNG, o kit de testes
```

## Acrescentando um contexto ou um app

- **Um app:** `cp -r apps/hello apps/<id>` ([Apps e o contrato](13-apps-e-o-contrato.md)); o teste de contrato o confere.
- **Um contexto:** crie `internal/<nome>/` com as camadas de que precisar (`domain`, `application`, `infrastructure` e uma
  camada de apresentação para telas), acrescente as regras dele em `architecture.Rules` e a pasta em
  `architecture.Contexts`, e descreva-o na documentação de `internal/architecture`. O Launcher será um: o
  `internal/launcher`, com o `apps/launcher` como raiz de composição.

[← Apps e o contrato](13-apps-e-o-contrato.md) · [Índice](README.md)
