# 11. Solução de problemas

[← Testes](10-testes.md) · [Índice](README.md)

Problemas encontrados ao construir este projeto em um PB741, com a causa e a correção.

## No leitor

| Sintoma | Causa | Correção |
|---|---|---|
| O ícone não aparece depois da primeira execução | A tela inicial só relê a configuração quando o cabo USB é desconectado (ou ao reiniciar) | Desconecte o cabo (a mensagem da primeira execução avisa isso) |
| O ícone aparece, mas tocar nele volta para a tela inicial | A entrada usava um caminho de lançamento **relativo** (`applications/x.app`) | Use o build atual: ele grava o caminho absoluto `/mnt/ext1/applications/x.app` e **repara** uma entrada antiga na próxima execução (abra o app pela lista de Aplicativos) |
| O nome aparece sob um ícone vazio | Os arquivos de ícone sumiram (`applications/icons/`) | Abra o app de novo: ele recria os ícones |
| A imagem fica deslocada ~137 px para baixo e o fundo encosta na borda | O painel do sistema está ligado e o InkView desloca o canvas | Chame `inkview.DisablePanel()` no `Init` (o modelo já faz) |
| O Sair não faz nada | `CloseApp()` sozinho não encerrava o app | `inkview.Exit()` chama `CloseApp()` **e** `LeaveInkViewMain()` |
| O toque cai "um pouco fora" do desenho | Desenho e toque usaram coordenadas diferentes | Não escale nem desloque à mão: use pixels de design e `doc.HandlePointer` com as coordenadas do aparelho |
| Digitar não faz nada | Nenhum teclado foi dado ao documento | `doc.Keyboard = env.Keyboard` (o kit entrega à sua fábrica, veja `apps/hello`) |
| Arrastar um slider é lento | O e-ink redesenha a tela inteira | O kit agrupa os redesenhos, ignora eventos de movimento e envia ao painel só as áreas que mudaram (atualização parcial) |
| Um símbolo (✓, →) aparece como quadrado | As fontes embutidas não têm esse glifo | Desenhe um ícone (`c.Icon`) ou use outro símbolo |
| O texto termina em "…" | Não cabe na caixa | Dê mais largura ao elemento (`FlexGrow`, `Width`) ou use um texto menor |
| O app não faz nada ao abrir | Ele travou antes de desenhar | Leia `applications/hello.log` (gravado pela demo) |

### Lendo o log de eventos

A demo grava `applications/hello.log` com cada evento do InkView e o que o app decidiu. Conecte o leitor, abra o arquivo e procure
`main start` (uma nova execução), `info before/after` (estado da tela e do painel), `pointer up (x,y) … hit=` e `install:` (o
resultado da primeira execução). No seu app defina `inkview.DebugLog = "/mnt/ext1/applications/<nome>.log"` e use
`inkview.Logf(formato, args...)`.

## Compilação

| Sintoma | Causa | Correção |
|---|---|---|
| O `docker` imprime a **ajuda** e nenhum `.app` é gerado | Uma função `docker` no perfil do PowerShell que descarta os argumentos (por exemplo `function docker() { wsl -- docker }`) | O `build.ps1` ignora funções e aliases; ou corrija o perfil: `function docker { wsl docker @args }` |
| `docker: command not found` no Windows | O Docker existe só dentro do WSL | O `build.ps1` recorre sozinho a `wsl -e docker` |
| O `apt-get` devolve 404 para `stretch` | A imagem do SDK é baseada no Debian stretch, que foi arquivado | O `Dockerfile` já aponta para `archive.debian.org` |
| `ft2build.h: file not found` | O header do FreeType está fora do caminho padrão de includes | Já tratado nas flags cgo de `internal/inkview` |
| Um caminho como `/mnt/d/...` vira `C:/Program Files/Git/...` | O Git Bash reescreve os caminhos que passa ao `wsl` | Rode os comandos do WSL pelo PowerShell, ou use os scripts `.ps1` |
| `-coverprofile=coverage.out` falha no PowerShell | O PowerShell quebra o argumento no ponto | Coloque entre aspas: `"-coverprofile=coverage.out"` |
| O build falha em `[T any]` ou `any` | O compilador da imagem é o Go 1.17 | Não use generics nem funções novas da biblioteca no código do app |
| "Build finished but … was not created" | O Docker rodou, mas não gerou nada | Leia a saída do Docker acima dessa mensagem |

## Testes e documentação

| Sintoma | Correção |
|---|---|
| `TestRepositoryDocsAreInSync` falha | Uma página de `docs/` está desatualizada: rode `go run ./tools/docsync` |
| As imagens dos componentes parecem antigas | Rode `go run ./tools/docgen` |
| Avisos "LF will be replaced by CRLF" do Git | Inofensivos no Windows (conversão de fim de linha) |
| O Sonar devolve 401 | A variável `SONAR_GLOBAL_TOKEN` não existe ou está errada |

## Ainda travado?

1. Rode `go test ./...` — um teste que falha costuma apontar o problema.
2. Gere a tela em PNG (`examples.SavePNG`) para ver o que o layout produziu.
3. Imprima o `element.Bounds` dos elementos que parecem errados.
4. Abra uma issue com o modelo, a versão do firmware e o conteúdo do arquivo de log.

[← Testes](10-testes.md) · [Índice](README.md)
