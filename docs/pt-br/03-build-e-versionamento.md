# 3. Build e versionamento

[← Instalação](02-instalacao.md) · [Índice](README.md) · [Próxima: Conceitos →](04-conceitos.md)

## Compilando o app

O leitor roda Linux ARM de 32 bits e a biblioteca de interface fala com o SDK do PocketBook (`libinkview`) via
cgo; por isso o app é compilado **dentro de uma imagem Docker** que traz o compilador cruzado do SDK e um Go 1.17
corrigido. Você nunca instala o SDK à mão.

### Um comando

```powershell
.\build.ps1                 # Windows: compila ./cmd/hello
.\build.ps1 -App meuapp     # compila ./cmd/meuapp
```
```sh
make dist                   # Linux / macOS / WSL
make dist APP=meuapp
```

O resultado é **um arquivo**: `dist/applications/<app>.app`. O script também:

- escolhe o Docker Desktop, ou o Docker dentro do **WSL** quando `docker` não é um executável do Windows;
- constrói a imagem (`docker/Dockerfile`) na primeira vez (as próximas reaproveitam o cache);
- **incrementa o número do build** e grava a versão no binário;
- falha com clareza se o `.app` não foi gerado (nunca trata um arquivo antigo como sucesso).

### O que há na imagem Docker

| Peça | Para quê |
|---|---|
| SDK do PocketBook 6.3.0 (branch `6.5`, `SDK-B288`) | compilador cruzado `arm-obreey-linux-gnueabi-clang`, headers e `libinkview` |
| Go 1.17.13 com o `cmd/go` corrigido | o linker do SDK rejeita `-Wl,--build-id=none`, que o `cmd/go` adiciona |
| `GOARCH=arm GOARM=7 CGO_ENABLED=1` | o alvo do leitor |

Flags usadas internamente (para referência):

```sh
go build -ldflags=-X=go-pocketbook-base/internal/version.Value=<versão> -o dist/applications/hello.app ./cmd/hello
```

### Alvos do make

| Alvo | O que faz |
|---|---|
| `make dist` | constrói a imagem e o `.app` (incrementa o BUILD) |
| `make test` | `go test ./...` |
| `make preview` | gera as páginas da demo em PNG |
| `make task-done` | **MINOR + 1** (veja abaixo) |
| `make major` | **MAJOR + 1** |
| `make sonar` | análise estática (veja abaixo) |
| `make clean` | remove `dist/` e as imagens de preview |

## Versionamento

A versão fica no arquivo [`.version`](../../.version), no formato **`MAJOR.MINOR.BUILD`**, e começa em `1.0.0`.

| Quando | O que sobe | Como |
|---|---|---|
| a cada **build** | `BUILD` | automático em `build.ps1` / `make dist` |
| uma **task é concluída** | `MINOR` (e o `BUILD` volta a 0) | `make task-done` |
| uma **grande mudança** | `MAJOR` (`MINOR` e `BUILD` voltam a 0) | `make major` |

```sh
go run ./tools/bumpversion build    # 1.2.3 -> 1.2.4
go run ./tools/bumpversion minor    # 1.2.4 -> 1.3.0
go run ./tools/bumpversion major    # 1.3.0 -> 2.0.0
```

A ferramenta imprime a nova versão. O build a grava no binário (`internal/version.Value`) e a demo a mostra no
rodapé, então você sempre sabe qual build está no leitor.

No seu código:

```go
import "go-pocketbook-base/internal/version"

fmt.Println("rodando v" + version.Value) // "dev" quando não foi compilado pelos scripts
```

## Testes

```sh
go test ./...                        # todos os pacotes
go test -cover ./...                 # com cobertura
go test ./internal/widget -run Layout -v
```

A suíte cobre o motor de layout, cada componente, a precisão do toque (o centro de cada elemento interativo
precisa receber o próprio toque), as regras de versionamento, o instalador e todos os exemplos da documentação.

## Ferramentas da documentação

```sh
go run ./tools/docgen       # redesenha docs/components/*.png a partir dos componentes reais
go run ./tools/docsync      # preenche os blocos de código de docs/ a partir de examples/
go run ./tools/docsync -check   # só confere (a suíte de testes também faz isso)
```

Para colocar um exemplo testado numa página, escreva um bloco `go` cuja **primeira linha** é um comentário com a
origem, deixe o resto vazio e rode o `docsync`:

- `// examples/login.go` embute o arquivo inteiro;
- `// examples/components.go#ButtonExample` embute uma função (com o comentário dela).

O bloco é preenchido (e depois atualizado) com o código de verdade. Uma página com bloco desatualizado faz o
`go test ./tools/docsync` falhar, então documentação e código não se separam.

## Análise estática (SonarQube)

Com um servidor SonarQube em `http://localhost:9000` e um token na variável de ambiente `SONAR_GLOBAL_TOKEN`:

```powershell
.\scripts\sonar.ps1        # Windows: testes com cobertura + scanner (Docker no WSL)
```
```sh
scripts/sonar.sh           # Linux / WSL
```

As regras ficam em `sonar-project.properties`. A meta é **0 code smells, bugs e vulnerabilidades** e o quality gate
aprovado.

## Integração contínua (sugestão)

```sh
go vet ./...
go test ./...
go run ./tools/docsync -check
make dist
```

Próxima: [Conceitos →](04-conceitos.md)
