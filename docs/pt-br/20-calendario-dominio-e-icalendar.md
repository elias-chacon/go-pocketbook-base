# 20. Calendário: domínio e iCalendar

[← E-mail: domínio e MIME](19-email-dominio-e-mime.md) · [Índice](README.md)

O contexto `internal/calendar` é o app de calendário (`apps/calendar`, tarefas T66 a T70): o **domínio**, a leitura e escrita de `.ics`, o **cache de eventos** e as
**telas**; a sincronização vem depois. Um guia rápido de uso está no [capítulo 23](23-apps-de-produtividade.md).

## O domínio

| Conceito | O que é |
|---|---|
| `Event` | algo que acontece num horário: título, descrição, local, início e fim (o fim é exclusivo), fuso, recorrência, exceções, lembretes, organizador, participantes, status e `Sequence` |
| `Occurrence` | uma aparição de um evento (um evento que se repete tem muitas) |
| `Calendar` | uma coleção de eventos com nome; pode ser **somente leitura** (importado ou assinado) |
| `Reminder` | quanto tempo **antes** do início a pessoa quer ser avisada |

Um evento **de dia inteiro** tem início e fim à meia-noite UTC (o fim é o dia seguinte ao último). Um evento que **se repete** mantém a hora
local do seu fuso no horário de verão: um "9h em Nova York toda segunda" continua às 9h depois da mudança, e vira 13h UTC em vez de 14h (há um
teste para isso).

**`Occurrences(de, até)`** dá as aparições que se sobrepõem a um intervalo: um evento simples dá a si mesmo, um que se repete é expandido pela
regra `RRULE` sem as exceções (`EXDATE`), e uma aparição que começou antes do intervalo mas ainda está em andamento (um plantão noturno) aparece.
Uma regra sem fim ("a cada minuto, para sempre") é limitada a 2 000 aparições, para não encher a memória do leitor. **`Expand`** junta vários
eventos, ordena por início e aplica as **alterações de uma só aparição** (`RecurrenceID`): a versão adiada substitui a original, e um evento
cancelado não aparece em lugar nenhum. A expansão usa o `rrule-go`, porque as regras do RFC 5545 são longas e exatas.

## Arquivos `.ics` (iCalendar)

`icalendar.Parse` lê os eventos de um arquivo e devolve também uma lista de **avisos**; `icalendar.Write` escreve o arquivo. A sintaxe (linhas
dobradas, escapes) é do `go-ical`; o significado é daqui.

- **Fusos**: o Outlook escreve nomes do Windows (`E. South America Standard Time`): uma tabela os converte em nomes IANA
  (`America/Sao_Paulo`). Nomes IANA com prefixo de caminho (`/mozilla.org/20050126_1/Europe/Paris`) também são entendidos. Um fuso desconhecido
  vira UTC **com um aviso**: o evento não se perde, só pode ficar deslocado. O banco de fusos vai dentro do programa (`time/tzdata`), pois o
  leitor pode não ter `zoneinfo`.
- **Datas e durações**: UTC (`...Z`), com `TZID`, "flutuante" e datas de dia inteiro; o fim vem de `DTEND`, de `DURATION`, do dia inteiro
  ou do próprio início.
- **Recorrência**: `RRULE`, `EXDATE` (um ou vários valores por linha) e `RECURRENCE-ID` para as aparições alteradas.
- **Lembretes** (`VALARM`): antes do início (`-PT15M`) ou num instante fixo. **Pessoas**: organizador e participantes com nome, e-mail e resposta.
- **Problemas**: um evento sem início, com fim antes do início ou com regra inválida é **pulado com um aviso** e os outros são mantidos; um evento
  sem `UID` ganha um (`imported-N`); um evento sem título aparece como "(sem título)". `VTODO` e outros componentes são ignorados (as tarefas
  terão o seu próprio leitor).
- **Escrita**: horários com fuso mantêm o `TZID`; sem fuso saem em UTC; dias inteiros saem como datas. O arquivo **não leva `VTIMEZONE`**: o
  nome IANA basta para os calendários que importam, e quem não o conhece mostra o evento em UTC. Sem eventos não há arquivo (`ErrNothingToWrite`),
  porque um iCalendar vazio não é válido.

## O cache de eventos (SQLite)

A porta `EventStore` é de onde a tela lê; o `sqlstore` a implementa no banco compartilhado (capítulo 22). **`Between(de, até)`** devolve os eventos
que **podem** ter uma aparição no intervalo, já prontos para o `Expand`: os simples que se sobrepõem, os que se repetem e começaram antes do fim
do intervalo, e as **alterações de uma só aparição** (inclusive as que saíram do intervalo para outro dia, e as que vieram de fora, porque a
original precisa ser substituída). Salvar de novo substitui (mesmo calendário, `UID` e `RecurrenceID`); apagar um evento que se repete leva as
alterações dele; um calendário que sumiu do servidor leva os seus eventos; e remover a conta apaga tudo.

## O app Calendário

O `apps/calendar` é a raiz de composição. Ele **não precisa de conta**: os eventos ficam no leitor, no banco compartilhado, sob a conta `local` (um
calendário é criado na primeira execução). Ele se parece com o **calendário do Outlook**: uma barra de ferramentas (**Hoje**, anterior e próximo, o período, o
menu de **visão** e **Novo**) sobre a visão de **Mês**, **Semana**, **Semana útil** ou **Dia**, sob a barra padrão do app (o nome do app com **Sobre** e **Sair**,
como no Hello Go).

- **Mês:** cada dia lista seus eventos (hora e título, "+N" quando há mais). Toque num dia para selecioná-lo e toque de novo para abri-lo na visão de Dia.
- **Dia, Semana útil, Semana:** as horas ao lado e os eventos como blocos posicionados pelo horário (os que se sobrepõem dividem a largura), eventos de dia
  inteiro numa faixa no alto e uma linha na hora atual. Toque num evento para editá-lo, num horário vazio para começar um evento ali, no cabeçalho de um dia
  para abri-lo. As horas se ampliam para mostrar eventos cedo e tarde.
- **O formulário do evento:** título, local, o dia (com setas), dia inteiro, início e fim como `HH:MM`, **repetir**, **lembrete** e **descrição**; um horário
  inválido mantém o formulário aberto.

As duas visões são componentes do toolkit compartilhado (**MonthView** e **TimeGrid**, [capítulo 6](06-componentes.md)). O caso de uso `Agenda` responde ao que
as telas pedem: **Span** (o que acontece em cada um de N dias, numa só consulta), **Add**, **Change** e **Delete**. Um evento de dia inteiro é uma **data**, não um
instante, então nunca escorrega para o dia vizinho em outro fuso. O relógio vem de `TZ` ou `/etc/timezone` (UTC quando o leitor não tem nenhum: a configuração
do fuso ainda falta). **Ainda não testado no aparelho.** Falta (T67): tocar os lembretes, e a sincronização.

## O código

| Camada | Pacote | Guarda |
|---|---|---|
| domain | `internal/calendar/domain` | `Event`, `Occurrence`, `Calendar`, `Occurrences`, `Expand`, as regras de validação |
| application | `internal/calendar/application` | `Agenda`: marcas do mês, um dia, adicionar, alterar, apagar |
| presentation | `internal/calendar/presentation` | as telas do Calendário: o mês com a agenda do dia, e o editor |
| infrastructure | `internal/calendar/infrastructure/icalendar` | `Parse`, `Write`, a tabela de fusos |
| | `internal/calendar/infrastructure/sqlstore` | o `EventStore` no SQLite: `Between`, calendários |

O domínio usa o `rrule-go` (uma biblioteca pequena, sem dependências, como a biblioteca padrão) e não conhece a plataforma nem o toolkit; o teste de arquitetura (capítulo 14) confere o resto.
