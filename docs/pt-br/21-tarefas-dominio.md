# 21. Tarefas: domínio

[← Calendário: domínio e iCalendar](20-calendario-dominio-e-icalendar.md) · [Índice](README.md)

O contexto `internal/tasks` é o app de tarefas (`apps/tasks`, tarefa T63): o **domínio**, o **cache de tarefas** e as **telas**. A sincronização (T64) e a
integração com o e-mail (T65) vêm depois. Um guia rápido de uso está no [capítulo 23](23-apps-de-produtividade.md).

## O domínio

| Conceito | O que é |
|---|---|
| `Task` | algo a fazer: título, notas, **prazo**, **Meu Dia**, lembrete, importante, **passos**, **repetição** e o e-mail de origem (`MailID`) |
| `List` | um grupo de tarefas com nome |
| `Step` | uma parte pequena de uma tarefa, marcada sozinha ("2 de 5") |
| `View` | um filtro sobre todas as listas: **Meu Dia**, **Importante**, **Planejadas**, **Atrasadas** e Todas |

As datas são **dias**, e não instantes: uma tarefa vence num dia, seja qual for o fuso (`Day` é a meia-noite UTC do dia do calendário).

**Concluir** (`Complete`) marca a tarefa. Se ela **se repete** (uma `RRULE`, como `FREQ=WEEKLY;BYDAY=FR`), devolve também a **próxima**: a mesma
tarefa, aberta, com o prazo na próxima data da regra, sem os ids, sem Meu Dia e sem lembrete, e com os passos desmarcados. Se foi concluída **com
atraso**, a próxima conta a partir de hoje (concluir uma tarefa semanal um mês depois não deixa uma pilha de cópias atrasadas); se a regra acabou
(`COUNT`), não há próxima. Uma tarefa que se repete precisa de prazo para ter de onde repetir.

As **visões** deixam de fora as tarefas concluídas (menos "Todas") e vêm na ordem que se espera: por prazo em Planejadas e Atrasadas; nas
demais, as importantes primeiro e depois as mais novas. `Count` dá os números ao lado de cada visão.

## O cache de tarefas (SQLite)

A porta `TaskStore` é de onde a tela lê; o `sqlstore` a implementa no banco compartilhado (capítulo 22). `Tasks` lista da mais nova para a mais antiga,
com as abertas por padrão (`IncludeDone` traz as concluídas), e **as visões são aplicadas por `Filter` sobre o que ele devolve**: uma pessoa tem
centenas de tarefas abertas, não milhões. Concluir uma tarefa que se repete grava a concluída e a próxima juntas (`Save(done, *next)`). Uma lista que
sumiu do servidor leva as suas tarefas, e remover a conta apaga tudo.

## O app Tarefas

O `apps/tasks` é a raiz de composição. Ele **não precisa de conta**: as tarefas ficam no leitor, no banco compartilhado, sob a conta `local` (uma lista
"Tarefas" é criada na primeira execução). O caso de uso `Checklist` responde ao que as telas pedem: as **visões** com suas contagens, **Add** (uma tarefa criada
olhando o Meu dia é para hoje, em Importante vem marcada, em Planejado vence hoje), **Toggle** (conclui, guardando a próxima ocorrência de uma tarefa que se
repete, ou reabre), estrela, Meu dia, prazo, salvar e apagar. A tela tem as cinco visões como chips com contagem, um campo para adicionar, **8 tarefas por
página** com uma caixa de marcar (o alvo de toque mais profundo) e o título e o prazo (as atrasadas dizem isso); tocar no texto abre a tarefa para editar:
título, notas, **etapas** (adicionar, marcar, remover), importante, Meu dia e o prazo (hoje, amanhã, próxima semana, nenhum). Nada é salvo até **Salvar**. **Ainda não testado no aparelho.** Falta
(T63): listas, lembretes e repetição na tela.

## O código

| Camada | Pacote | Guarda |
|---|---|---|
| domain | `internal/tasks/domain` | `Task`, `List`, `Step`, `View`, `Complete`, `Filter`, `Count`, a porta `TaskStore` |
| application | `internal/tasks/application` | `Checklist`: visões, adicionar, concluir, estrela, Meu dia, prazo, salvar, apagar |
| presentation | `internal/tasks/presentation` | as telas das Tarefas: a lista com visões e o editor |
| infrastructure | `internal/tasks/infrastructure/sqlstore` | o `TaskStore` no SQLite |

O domínio usa o `rrule-go` para a repetição e só depende do núcleo compartilhado; o teste de arquitetura (capítulo 14) confere.
