# 13. Apps e o contrato

[← Internacionalização](12-internacionalizacao.md) · [Índice](README.md)

Todo app deste projeto (o Launcher e seus plugins) vive em **uma pasta dentro de `./apps/`** e segue o mesmo contrato.
O `apps/hello` é o blueprint: é um app de verdade, funcionando, e o modelo de todo app novo.

## Abrir outro app: intents

O leitor roda um app por vez, então os apps nunca conversam enquanto rodam. Um app que quer que outro faça algo chama
`env.Send(appID, "tasks.create", dados)`: a plataforma troca o programa em execução por esse app (como o Launcher faz) e entrega uma
**intent**. O outro app a encontra em `env.Intent` ao abrir (`Action`, `From` e um mapa `Data` pequeno: passe ids e deixe-o ler o resto
do armazenamento compartilhado; a intent é limitada a 48 KB porque viaja no ambiente). Quando o app aberto pede para sair, o leitor
**volta para o app que pediu** (e não para o Launcher), e a intent é removida do ambiente para que o app de volta não a tome por um pedido
seu. O Home sempre sai.

Um app declara as ações que trata no `app.json`, `"intents": ["tasks.create", "tasks.show"]`, cada uma no formato `app.acao`; o `Send`
falha com `platform.ErrNoSuchApp` ou `platform.ErrNotHandled`, sem sair do app, quando o destino não está instalado ou não trata a ação.

## Uma pasta, um arquivo

```
apps/<id>/
  app.json     o manifesto (embutido no programa)
  icon.png     (opcional) o ícone, 128x128: a release o publica para o Launcher mostrar o app antes de instalá-lo, e os
               instaladores o usam como ícone na tela inicial
  main.go      embute o app.json e chama platform.Run
  *.go         o app: uma fábrica que monta a tela
  locales/     (opcional) traduções, pt-BR e en
```

Compilar gera **um único arquivo**, `<id>.app`. Instalar é copiar esse arquivo para a pasta `applications` do leitor;
nada mais. Um app é instalado, atualizado e executado sozinho: o Launcher é uma comodidade, não uma exigência.

## Todo app segue o blueprint do Hello Go

Todo app do repositório é um **plugin do Launcher que também se instala sem ele**: um `<id>.app` com seu próprio `app.json`, que se registra na primeira execução
(assim o Launcher o lista e a tela inicial mostra seu ícone). O Hello Go é o modelo, e o `apps/contract_test.go` confere:

- a **barra superior padrão**: um menu com o nome do app e **Sobre** (nome, descrição, versão) e **Sair** (volta ao Launcher que o abriu, ou à tela inicial). É o
  `widget.AppChrome` do toolkit; o app a recebe com `platform.Chrome(env)` e a põe acima de cada tela sua (`cfg.Chrome.Wrap`). Um teste falha para o app que não a usa (o Hello é o modelo e o Launcher é o anfitrião em que os apps se plugam, então os dois ficam de fora);
- o manifesto com textos em pt-BR e inglês, o `main.go` que o embute e chama `platform.Run`, e um `icon.png` opcional;
- **se atualiza sozinho**, em segundo plano: assim que abre, o `platform.Run` verifica (fora do laço de eventos) se o leitor está online e, se estiver, se a última Release do GitHub publica uma build deste app diferente da instalada. **Quem decide é o checksum, não a versão**: uma release pode mudar o Launcher ou o Hello e deixar o Tasks como estava, então cada entrada do `catalog.json` leva o `app_sha256` (o SHA-256 do `<id>.app`) e o app o compara com o SHA-256 do próprio arquivo; uma release sem `app_sha256` não oferece nada. Se diferir, um diálogo oferece **Atualizar agora** ou **Depois**: *Atualizar agora* baixa o zip, confere tamanho e SHA-256, tira o `<id>.app` dele (precisa ser um programa), troca o arquivo do próprio app por um temporário e reinicia a nova versão do jeito que a antiga foi aberta (mesma intent, mesmo caminho de volta); *Depois* lembra o checksum daquela build para não oferecer de novo (um diferente é oferecido). Offline, ou qualquer falha ao descobrir, é silencioso e tentado de novo na próxima abertura; uma verificação bem-sucedida é lembrada por dez minutos (`applications/.pbgo/update.json`), então passar de um app a outro não consulta a rede toda vez. Uma versão `dev` nunca é substituída. Nada é preciso em um app: o `platform.Run` faz isso para todos os apps e para o Launcher, e um teste falha se o `Run` deixar de ligar isso (`internal/platform/application/update.go`, `updateflow.go`, `infrastructure/selfupdate`);
- todas as telas feitas com os widgets do toolkit (um componente que falta é criado no toolkit, nunca no app).

## O manifesto (`app.json`)

```json
{
  "id": "hello",
  "title":       { "pt-BR": "Hello Go", "en": "Hello Go" },
  "description": { "pt-BR": "Vitrine de todos os componentes…", "en": "Showcase of every component…" },
  "group": "@General"
}
```

| Campo | Regra |
|---|---|
| `id` | letras minúsculas, dígitos, `-` ou `_`, começando por letra; **igual ao nome da pasta** (ele nomeia o `<id>.app`) |
| `title`, `description` | textos por idioma; **pt-BR é obrigatório** (o idioma primário), `en` é o fallback |
| `group` | grupo do ícone no desktop (padrão `@General`) |

O `platform.ParseManifest` valida e o `TitleIn(idioma)` escolhe o texto como o [i18n](12-internacionalizacao.md) faz.

## O ciclo de vida

1. O `main` chama `platform.Run(config, fábrica)`.
2. **Primeira execução:** a plataforma grava os ícones, coloca o app no desktop do leitor (`view.json`, com backup) e grava um
   **registro** `applications/.pbgo/<id>.json` (o manifesto mais versão, caminho do arquivo e do ícone): é o que o Launcher
   lê para saber o que está instalado. Isso roda a cada início, então o registro sempre tem a versão atual.
3. A plataforma lê as **configurações** compartilhadas (`applications/.pbgo/settings.json`: hoje o idioma) e inicia o InkView.
4. Quando o tamanho da tela é conhecido a plataforma chama a sua **fábrica** com um `platform.Env`: tamanho da tela, teclado do
   sistema, `Exit`, o aviso da primeira execução, o manifesto, o idioma, a versão e o `HTTP`, o cliente de rede que funciona no leitor (o Linux dele não tem `resolv.conf` nem IPv6, que o cliente padrão exige; `internal/device/network`). Use `env.HTTP` em qualquer requisição. `env.After(d, f)` roda `f` no laço de eventos do leitor depois de `d`, nunca no meio de um toque ou de um repaint: um trabalho longo (um download) é feito em passos pequenos, cada um agendado pelo anterior, e a tela continua respondendo aos toques entre eles (uma goroutine mexeria nos widgets na hora errada). Ela devolve um `platform.App`, que só
   precisa entregar o seu `*widget.Document`.
5. Daí em diante a plataforma cuida da instalação elétrica: toque, teclado do sistema, **repintura parcial** (refresh completo a cada
   10 atualizações para limpar o ghosting do e-ink) e as teclas: esquerda e Menu fecham um popup e depois uma modal
   (`doc.HandleBack()`); as teclas que a plataforma não usa vão para o app, se ele implementar `platform.KeyHandler`.

```go
// apps/hello/main.go#main
func main() {
	err := platform.Run(platform.Config{
		Manifest: manifest,
		// An app may draw its own launcher icon; without Icon the platform draws one
		// from the title (platform.DefaultIcon).
		Icon: func(size int, focused bool) image.Image { return showcase.RenderIcon(size, focused) },
	}, newHello)
	if err != nil {
		log.Fatal(err)
	}
}
```

```go
// apps/hello/hello.go#newHello
// newHello is the platform.Factory: the platform calls it once the screen size
// is known, with everything the app may need in env.
func newHello(env platform.Env) platform.App {
	return &hello{demo: showcase.New(env.Width, env.Height, showcase.Config{
		Version:  env.Version,
		Keyboard: env.Keyboard,
		Exit:     env.Exit,
		Notice:   env.Notice,
		Lang:     env.Lang,
		// The language chosen here becomes the language of every app.
		OnLangChange: func(lang string) { _ = env.Settings.Save(platform.Settings{Lang: lang}) },
	})}
}
```

O método `Key(key platform.Key) bool` do próprio app (em `hello.go`) é como ele reage às teclas que a plataforma deixa passar.

## Onde mora o código da plataforma

O `internal/platform` é um contexto delimitado com três camadas (veja [Arquitetura (DDD)](14-arquitetura.md)):
**domain** (`Manifest`, `InstalledApp`, `Settings` e as portas), **application** (`Runtime`, `FirstRun`) e
**infrastructure** (`jsonfiles`, `reader`). Um app importa apenas o pacote `platform`, que as liga.

## O idioma é compartilhado

`env.Settings.Load().Lang` é o idioma escolhido para todo o ecossistema; a plataforma o entrega à sua fábrica
(`env.Lang`). Quando o usuário o troca dentro de um app, salve com `env.Settings.Save` para que os outros apps já
iniciem nele (o `apps/hello` faz isso em `OnLangChange`).

## Criar um app

1. `cp -r apps/hello apps/meuapp`
2. Edite `apps/meuapp/app.json`: `"id": "meuapp"`, títulos e descrições em pt-BR e en.
3. Troque a tela em `newHello` (renomeie) e apague o que não precisar.
4. `go test ./...`: o **teste de contrato** (`apps/contract_test.go`) confere toda pasta de `apps/`: manifesto válido, id igual
   à pasta, `main.go` embutindo o `app.json` e chamando `platform.Run`, `locales/` completos e, se houver `icon.png`, um PNG 128x128. Os
   apps deste repositório desenham o ícone em código; `go run ./tools/appicons` gera o arquivo a partir desse desenho (um teste
   falha quando ele está desatualizado).
5. `.\build.ps1 -App meuapp` e copie `dist/applications/meuapp.app` para o leitor.

Escreva os textos em **português (pt-BR)** com inglês como fallback, pelo [`internal/localization/i18n`](12-internacionalizacao.md).

[← Internacionalização](12-internacionalizacao.md) · [Índice](README.md)
