# 22. Armazenamento local

[← Tarefas: domínio](21-tarefas-dominio.md) · [Índice](README.md)

O pacote `internal/storage` é o banco de dados local que os apps de e-mail, tarefas e calendário compartilham: **um arquivo SQLite**
(`applications/.pbgo/data.db`) com as mensagens, os eventos e as tarefas em cache. As telas leem do disco do leitor e **nunca esperam a
rede** (*offline-first*: a sincronização atualiza o banco, não a tela).

## O que ele faz

- É **Go puro** (`modernc.org/sqlite`, decidido no capítulo 17): compila com o Go 1.17 do SDK para ARM (verificado) e os testes rodam em
  qualquer computador, sem compilador C. O custo é cerca de **5,5 MB** a mais em cada app que o usa.
- **Abre com as configurações de um leitor pequeno**: cache de páginas de cerca de 2 MB, *journal* de reversão que é **truncado** (e não o WAL,
  pois o armazenamento do leitor é um volume FAT, onde o log de escrita e a memória compartilhada dele não são seguros), chaves estrangeiras ligadas
  e uma única conexão (um app roda por vez).
- **Migrações por módulo** (`Migrate`): cada contexto (`mail`, `calendar`, `tasks`) tem os seus passos numerados 1, 2, 3, sem lacunas. Os que
  ainda não rodaram executam em ordem, cada um numa transação junto com o registro da versão, então **um passo que falha não deixa nada pela
  metade**. É seguro chamar a cada abertura. Um banco gravado por um programa **mais novo** é recusado (`ErrDowngrade`), em vez de ser mexido por
  um que não o entende.
- **Apagar uma conta** (`EraseAccount`, ou `Erase`, que é a porta `Eraser` do contexto de contas): cada contexto declara as suas tabelas com
  coluna `account_id` (`RegisterAccountTable`), e remover a conta apaga as linhas dela em todas, **tudo ou nada**.
- `Check` roda a verificação de integridade do SQLite, e `Tables` lista as tabelas (para diagnóstico).

O banco não sabe o que é uma mensagem ou um evento: cada contexto escreve as suas tabelas e as suas consultas em cima do `DB()`.

## O que ainda não foi verificado

**O leitor de verdade.** O SQLite está verificado em **ARMv7 emulado** (todos os testes do armazenamento passam no `qemu-arm` com o Go 1.17 do SDK:
script ARM, já removido), mas não rodou no PocketBook em si, e a emulação não confere o **volume FAT** onde o banco vai ficar. É a primeira coisa a validar
quando um app o usar; se falhar, a alternativa é um arquivo próprio atrás da mesma porta.

## O código

| Pacote | Guarda |
|---|---|
| `internal/storage` | `Open`, `Migrate`, `Version`, `RegisterAccountTable`, `EraseAccount`/`Erase`, `Check`, `Tables` |

Depende só do núcleo compartilhado e do driver; o teste de arquitetura (capítulo 14) confere.
