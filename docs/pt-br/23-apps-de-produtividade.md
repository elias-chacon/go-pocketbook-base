# 23. Os apps de produtividade: E-mail, Tarefas e Calendário

Este capítulo é o **guia de leitura rápida** dos três apps de produtividade. Ele diz o que cada um faz hoje, como instalar, o que ainda não funciona e onde
está o código. Os detalhes técnicos ficam nos capítulos [17](17-decisoes-apps-de-produtividade.md) a [22](22-armazenamento-local.md).

> **Estado atual:** os três apps compilam para o leitor, têm testes automáticos (inclusive em ARMv7 emulado) e já estão na Release.
> **Nenhum foi testado no aparelho** e nenhum provedor de e-mail de verdade foi testado ainda.

## Em uma tabela

| App | Pasta | Precisa de conta? | O que já faz | O que falta |
|---|---|---|---|---|
| **E-mail** | `apps/mail` | Sim (IMAP com **senha de app**) | pastas, lista, leitura, responder, encaminhar, nova mensagem, apagar, busca, sincronizar, ler offline | abrir anexos, rascunhos, Cco no formulário, Outlook.com (OAuth) |
| **Tarefas** | `apps/tasks` | Não (ficam no leitor) | visões Meu dia, Importante, Planejado, Atrasadas e Todas; adicionar, concluir, editar, apagar; tarefas que se repetem | lembretes e repetição na tela, sincronização |
| **Calendário** | `apps/calendar` | Não (ficam no leitor) | mês com pontos nos dias com eventos, agenda do dia, criar, editar e apagar eventos (dia inteiro ou com horário) | visões de dia e semana, tocar os lembretes, sincronização, importar `.ics` |

## Como instalar

1. Instale o Launcher e os apps pelo [instalador](16-instalador.md) (ele já copia tudo), ou
2. abra o **Launcher**, busque pelo nome (`E-mail`, `Tarefas`, `Calendário`) e toque na linha em **Disponíveis** para baixar e instalar.

Depois de instalar pelo Launcher, o ícone aparece na tela inicial quando o cabo USB é desconectado.

## E-mail

1. Na primeira vez o app pede uma conta: digite o **endereço** e a **senha de app** (a senha normal não funciona em Gmail, iCloud e similares).
   - Provedor conhecido (Gmail, iCloud…): os servidores são preenchidos sozinhos.
   - Provedor desconhecido: aparecem os campos dos servidores IMAP e SMTP.
2. Toque em **Sincronizar** para trazer as **50 mensagens mais novas** de cada pasta. O resto fica no servidor.
3. A lista mostra **7 mensagens por página** (remetente, data, assunto e prévia; não lidas em negrito). As teclas de página do leitor e as setas viram a página.
4. Tocar numa mensagem a abre **página a página** e a marca como lida. Ali há **Responder**, **Responder a todos**, **Encaminhar**, **Sinalizar**, **Marcar não lida** e **Apagar**.
5. **Nova** abre o formulário (Para, Cc, Assunto, texto). Um endereço inválido ou um envio recusado mantém o formulário aberto com o motivo.

Tudo é lido do **cache offline**: sem rede você continua lendo o que já veio. Apagar vai ao servidor primeiro (se ele recusar, a mensagem fica).

## Tarefas

- O app abre em **Meu dia**. Os chips no alto trocam a visão e mostram quantas tarefas cada uma tem.
- Digite um título no campo e confirme para **adicionar**: em Meu dia a tarefa é para hoje, em Importante já vem marcada, em Planejado vence hoje.
- A caixa à esquerda **conclui** a tarefa (uma tarefa que se repete cria a próxima sozinha). Tocar no texto abre o **editor**: título, notas, etapas, importante,
  Meu dia e prazo (hoje, amanhã, próxima semana, nenhum). Nada é salvo até **Salvar**.
- O menu **Listas** escolhe uma lista (ou todas), cria uma nova e apaga a mostrada.
- **8 tarefas por página.**

## Calendário

- O app abre no **mês de hoje**. Um **ponto** marca os dias que têm eventos; **Hoje** volta para o dia de hoje.
- Tocar num dia mostra a **agenda dele** embaixo (dia inteiro primeiro, depois por horário; **5 eventos por página**).
- **Novo** (ou tocar num evento) abre o formulário: título, local, o dia (com setas), **dia inteiro** e início e fim como `HH:MM` **repetir** (todo dia, semana, mês ou ano) e **lembrar** (de 5 minutos a 1 dia antes).
- Um evento de **dia inteiro** é uma data, não um instante: ele não muda de dia por causa do fuso.
- O fuso vem da variável `TZ` ou de `/etc/timezone`; sem nenhum dos dois, vale UTC (a configuração do fuso ainda falta).

## Onde ficam os dados no leitor

Tudo em `applications/.pbgo/`:

| Arquivo | Conteúdo |
|---|---|
| `accounts.json` | as contas (sem senhas) |
| `secrets.bin` | o cofre: as senhas criptografadas com **AES-256-GCM** (ver [capítulo 18](18-contas-e-segredos.md)) |
| `productivity.db` | o banco SQLite compartilhado: mensagens, tarefas e eventos ([capítulo 22](22-armazenamento-local.md)) |

Remover uma conta apaga também tudo o que ela deixou no banco.

## O que cada parte do código faz

```
apps/mail  apps/tasks  apps/calendar     as raízes de composição (só ligam as peças)
        │
internal/<contexto>/presentation         as telas, feitas só com widgets do toolkit
internal/<contexto>/application          os casos de uso: Mailbox/Sync, Checklist, Agenda
internal/<contexto>/domain               as regras e as portas (Message, Task, Event…)
internal/<contexto>/infrastructure       o que fala com o mundo: imapmail, smtpsend, graph,
                                         xoauth2, sqlstore, icalendar, mime
internal/accounts                        contas, cofre e a tela de login compartilhada
internal/storage                         o banco SQLite compartilhado (migrações por módulo)
```

A regra de dependência (só aponta para dentro) é conferida por `go test ./internal/architecture` ([capítulo 14](14-arquitetura.md)). Todo componente de
tela novo vai para o **toolkit compartilhado**, nunca para dentro de um app: o `Calendar` do toolkit, por exemplo, ganhou `Marked` (o ponto nos dias) e
`OnMonth` para servir ao app de calendário e a qualquer outro.

## O que foi verificado e o que não foi

| Verificado | Como |
|---|---|
| Regras, telas, toque no centro de cada elemento, nada estoura a tela | testes automáticos na máquina |
| Tudo compila e passa em **ARMv7 com Go 1.17**, sem zoneinfo | `.\scripts\arm-test.ps1` (QEMU em modo usuário) |
| Os apps compilam para o leitor | `.\build.ps1 -App <id>` |
| IMAP, SMTP e Graph | contra servidores **de mentira** |

| **Não** verificado | Observação |
|---|---|
| Os três apps **no aparelho** | falta o teste real de toque, velocidade e memória |
| SQLite no volume **FAT** do leitor | o banco usa o diário `TRUNCATE`, nunca `WAL`, por causa disso |
| IMAP/SMTP em **provedores reais** | só servidores de teste |
| **Graph** (Outlook.com / Microsoft 365) | e o envio com Cco depende de uma suposição ainda não conferida |
| Login **OAuth2** do Outlook.com | precisa de um *client ID* do Microsoft Entra (tarefa T53) |

## Próximos passos

Anexos, marcadores e rascunhos no e-mail; lembretes nas tarefas; visões de dia e semana no calendário (com um componente de grade de
horários no toolkit); sincronização (Graph, CalDAV, Exchange); integrações entre os apps (e-mail → tarefa, convite → evento). A lista completa e o
estado de cada item estão nas tarefas **T51 a T70** do [README](../../README.md#tasks).
