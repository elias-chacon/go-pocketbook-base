# 13. Apps e o contrato

[← Internacionalização](12-internacionalizacao.md) · [Índice](README.md)

Todo app deste projeto (o Launcher e seus plugins) vive em **uma pasta dentro de `./apps/`** e segue o mesmo contrato.
O `apps/hello` é o blueprint: é um app de verdade, funcionando, e o modelo de todo app novo.

## Uma pasta, um arquivo

```
apps/<id>/
  app.json     o manifesto (embutido no programa)
  main.go      embute o app.json e chama appkit.Run
  *.go         o app: uma fábrica que monta a tela
  locales/     (opcional) traduções, pt-BR e en
```

Compilar gera **um único arquivo**, `<id>.app`. Instalar é copiar esse arquivo para a pasta `applications` do leitor;
nada mais. Um app é instalado, atualizado e executado sozinho: o Launcher é uma comodidade, não uma exigência.

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

O `appkit.ParseManifest` valida e o `TitleIn(idioma)` escolhe o texto como o [i18n](12-internacionalizacao.md) faz.

## O ciclo de vida

1. O `main` chama `appkit.Run(config, fábrica)`.
2. **Primeira execução:** o kit grava os ícones, coloca o app no desktop do leitor (`view.json`, com backup) e grava um
   **registro** `applications/.pbgo/<id>.json` (o manifesto mais versão, caminho do arquivo e do ícone): é o que o Launcher
   lê para saber o que está instalado. Isso roda a cada início, então o registro sempre tem a versão atual.
3. O kit lê as **configurações** compartilhadas (`applications/.pbgo/settings.json`: hoje o idioma) e inicia o InkView.
4. Quando o tamanho da tela é conhecido o kit chama a sua **fábrica** com um `appkit.Env`: tamanho da tela, teclado do
   sistema, `Exit`, o aviso da primeira execução, o manifesto, o idioma e a versão. Ela devolve um `appkit.App`, que só
   precisa entregar o seu `*widget.Document`.
5. Daí em diante o kit cuida da instalação elétrica: toque, teclado do sistema, **repintura parcial** (refresh completo a cada
   10 atualizações para limpar o ghosting do e-ink) e as teclas: esquerda e Menu fecham um popup e depois uma modal
   (`doc.HandleBack()`); as teclas que o kit não usa vão para o app, se ele implementar `appkit.KeyHandler`.

```go
// apps/hello/main.go#main
func main() {
	err := appkit.Run(appkit.Config{
		Manifest: manifest,
		// An app may draw its own launcher icon; without Icon the kit draws one
		// from the title (appkit.DefaultIcon).
		Icon: func(size int, focused bool) image.Image { return ui.RenderIcon(size, focused) },
	}, newHello)
	if err != nil {
		log.Fatal(err)
	}
}
```

```go
// apps/hello/hello.go#newHello
// newHello is the appkit.Factory: the kit calls it once the screen size is
// known, with everything the app may need in env.
func newHello(env appkit.Env) appkit.App {
	return &hello{demo: demo.New(env.Width, env.Height, demo.Config{
		Version:  env.Version,
		Keyboard: env.Keyboard,
		Exit:     env.Exit,
		Notice:   env.Notice,
		Lang:     env.Lang,
		// The language chosen here becomes the language of every app.
		OnLangChange: func(lang string) { _ = appkit.SaveSettings(env.Root, appkit.Settings{Lang: lang}) },
	})}
}
```

O método `Key(code int) bool` do próprio app (em `hello.go`) é como ele reage às teclas que o kit deixa passar.

## O idioma é compartilhado

`appkit.LoadSettings(root).Lang` é o idioma escolhido para todo o ecossistema; o kit o entrega à sua fábrica
(`env.Lang`). Quando o usuário o troca dentro de um app, salve com `appkit.SaveSettings` para que os outros apps já
iniciem nele (o `apps/hello` faz isso em `OnLangChange`).

## Criar um app

1. `cp -r apps/hello apps/meuapp`
2. Edite `apps/meuapp/app.json`: `"id": "meuapp"`, títulos e descrições em pt-BR e en.
3. Troque a tela em `newHello` (renomeie) e apague o que não precisar.
4. `go test ./...`: o **teste de contrato** (`apps/contract_test.go`) confere toda pasta de `apps/`: manifesto válido, id igual
   à pasta, `main.go` embutindo o `app.json` e chamando `appkit.Run`, e `locales/` completos.
5. `.\build.ps1 -App meuapp` e copie `dist/applications/meuapp.app` para o leitor.

Escreva os textos em **português (pt-BR)** com inglês como fallback, pelo [`internal/i18n`](12-internacionalizacao.md).

[← Internacionalização](12-internacionalizacao.md) · [Índice](README.md)
