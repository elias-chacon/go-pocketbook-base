# 21. Tarefas: domínio

[← Calendário: domínio e iCalendar](20-calendario-dominio-e-icalendar.md) · [Índice](README.md)

O contexto `internal/tasks` é a base do futuro app de tarefas (`apps/tasks`, tarefa T63). Por enquanto só tem o **domínio**; a tela, o
armazenamento (T54), a sincronização (T64) e a integração com o e-mail (T65) vêm depois.

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

## O código

| Camada | Pacote | Guarda |
|---|---|---|
| domain | `internal/tasks/domain` | `Task`, `List`, `Step`, `View`, `Complete`, `Filter`, `Count` |

O domínio usa o `rrule-go` para a repetição e só depende do núcleo compartilhado; o teste de arquitetura (capítulo 14) confere.
