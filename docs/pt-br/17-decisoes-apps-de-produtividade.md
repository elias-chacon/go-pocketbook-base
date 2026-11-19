# 17. Decisões de arquitetura: apps de e-mail, tarefas e calendário

[← O instalador e as releases](16-instalador.md) · [Índice](README.md)

Este é o registro de decisões (ADR) da tarefa **T51**, que antecede os três apps de produtividade (T58 a T70). Cada decisão
tem o motivo e, quando deu para medir, a **evidência**. As escolhas foram feitas sem consulta, para não parar o trabalho: **valem
até você vetá-las**, e trocá-las é barato porque cada uma fica atrás de uma porta (interface) do domínio.

## Restrições que decidem tudo

- O código do app é **Go 1.17** (o compilador do SDK do PocketBook): sem generics, sem `any`, sem funções novas da biblioteca
  padrão (`errors.Join`, `strings.Cut`…). Uma biblioteca que as use **não compila**.
- Memória: **1 GB**, um app por vez (o Launcher troca o processo por `execve`). Telas e listas não podem carregar tudo.
- O PC de desenvolvimento **não tem `gcc`**: nada que dependa de cgo pode ser testado localmente (só no Docker). O cgo fica restrito
  ao `inkview`.
- Sem teclado físico (o do sistema só devolve o texto ao confirmar), sem navegador (a autenticação não pode abrir uma página).

## Decisões

| # | Assunto | Decisão | Motivo e evidência |
|---|---|---|---|
| 1 | Armazenamento local (T54) | **SQLite em Go puro**, `modernc.org/sqlite` v1.14.8, atrás de uma porta `store` | Compilou com o Go 1.17 do SDK para ARM em 19 s e funciona no PC sem `gcc`. Custo: **+5,5 MB** por app que o use (um binário de teste foi de 1 MB para 6,8 MB). **Não verificado:** a execução em ARM (o Docker daqui não tem emulação), então o primeiro passo da T54 é rodá-lo no aparelho |
| 2 | Consulta e busca | SQL com índices e `LIKE`/FTS simples; **paginação sempre** | Evita carregar a caixa de entrada inteira; `PRAGMA cache_size` pequeno (cerca de 2 MB) |
| 3 | IMAP (T59) | `github.com/emersion/go-imap` **v1.2.1** | Compila no Go 1.17/ARM. A **v2** exige Go 1.18 e fica de fora |
| 4 | SMTP (T59) | **`net/smtp` da biblioteca padrão**, com um `smtp.Auth` próprio para XOAUTH2 | O `go-smtp` v0.26 **não compila** no Go 1.17 (usa `errors.Join` e `strings.Cut`). O `net/smtp` já faz STARTTLS, PLAIN e LOGIN |
| 5 | MIME e e-mail (T58) | `github.com/emersion/go-message` v0.18.2 | Compila no Go 1.17/ARM; trata *charsets* e partes aninhadas |
| 6 | iCalendar e recorrência (T66) | `github.com/emersion/go-ical` e `github.com/teambition/rrule-go` v1.8.2 | Compilam no Go 1.17/ARM (vêm como dependências do `go-webdav`). O `arran4/golang-ical` exige Go 1.20 e fica de fora |
| 7 | CalDAV e WebDAV (T68) | `github.com/emersion/go-webdav` v0.7.0 (pacote `caldav`) | Compila no Go 1.17/ARM (o conjunto IMAP + MIME + CalDAV ocupa 3,3 MB) |
| 8 | Microsoft Graph (T60, T64, T68) | **HTTP e JSON diretos** (`env.HTTP` e `encoding/json`), sem SDK | O SDK da Microsoft é enorme e exige Go novo; a API REST é pequena |
| 9 | Exchange local (T61) | **EWS** em SOAP/XML (`encoding/xml`) escrito por nós | Não há biblioteca em Go compatível; a fatia necessária (mensagens, calendário, tarefas) é pequena |
| 10 | Autenticação (T53) | **OAuth2 por código de dispositivo** (RFC 8628) para a **Microsoft**, e **senha de app** para os demais (Google, iCloud, Fastmail) | O leitor não tem navegador: ele mostra o código e o endereço, e você autoriza no celular. Isso serve à Microsoft, que aceita os escopos de e-mail, calendário e tarefas nesse fluxo. O **Google não**: o fluxo por código de dispositivo dele só permite escopos limitados (login, Drive, YouTube), sem Gmail nem Calendar. O Gmail usa então IMAP/SMTP com senha de app, e o calendário do Google entra **somente leitura** pelo endereço iCal secreto (T70). Precisa de um *client ID* da Microsoft registrado por você |
| 11 | Tokens e senhas (T57) | Cifrados em repouso com **AES-GCM**; chave derivada de um **PIN opcional** e, sem PIN, de um identificador do aparelho | Sem PIN a cifra só protege contra copiar o arquivo para outro lugar; **não é cofre** (o leitor não tem *keystore*). Isso deve estar dito na tela |
| 12 | Sincronização (T54) | **Offline-first**: a interface lê só do `store`; uma fila de operações sincroniza quando há rede (`Env.HTTP` conecta o Wi-Fi) | O Wi-Fi do leitor demora a conectar e some; a tela nunca espera a rede |
| 13 | Integração entre apps (T55) | **Intents** (`PBGO_INTENT`, na linha do `PBGO_LAUNCHER`) e dados compartilhados no `store` | Um app por vez: não há comunicação em processo, só dados em disco e um app abrindo o outro |
| 14 | Contas (T52) | Um contexto `accounts` compartilhado: uma conta serve e-mail, tarefas e calendário | Evita cadastrar a mesma conta três vezes |

## O que isso muda nas tarefas

- **T54** começa por um *spike*: rodar o SQLite em ARM no aparelho antes de construir o `store` sobre ele. Se falhar, a alternativa
  é um arquivo próprio (registros em JSON com índice em memória), atrás da mesma porta.
- **T59** não usa biblioteca de SMTP: o envio é `net/smtp` mais XOAUTH2.
- **T66** usa `go-ical` e `rrule-go` em vez de escrever o *parser* e a expansão de recorrência do zero.
- **T53** só pode ser concluída depois que você registrar o aplicativo OAuth na Microsoft (Entra ID), como cliente público com "permitir fluxos de cliente público" ligado. O Google não precisa de registro, porque usa senha de app.

## Como cada decisão foi checada

As bibliotecas foram compiladas **de verdade** no contêiner do SDK (Go 1.17.13, `GOARCH=arm`, `CGO_ENABLED=1`), não só lidas pelas
diretivas `go` dos `go.mod`: a diretiva `go 1.13` do `go-smtp` não impediu a falha de compilação. Para repetir, crie um módulo
com `go 1.17`, importe a biblioteca e rode `docker run --rm -v <pasta>:/app go-pocketbook-base-sdk build .`.
