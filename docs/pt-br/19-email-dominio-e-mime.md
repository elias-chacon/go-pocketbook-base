# 19. E-mail: domínio e MIME

[← Contas e segredos](18-contas-e-segredos.md) · [Índice](README.md)

O contexto `internal/mail` é a base do futuro app de e-mail (`apps/mail`, tarefa T58). Por enquanto tem o **domínio** e o **leitor e
escritor de MIME**; os protocolos (IMAP, SMTP, Graph, EWS), o cache offline (precisa do `store`, T54) e a tela vêm depois.

## O domínio

| Conceito | O que é |
|---|---|
| `Message` | um e-mail como a pessoa o lê: remetente, destinatários, assunto, data, texto, anexos (nome, tipo e tamanho), `Flags` (lida, sinalizada, respondida) e os campos da conversa (`Message-ID`, `In-Reply-To`, `References`) |
| `Folder` | onde ficam as mensagens, com um **papel** (`inbox`, `sent`, `drafts`, `trash`, `junk`, `archive`): é assim que os apps acham a Caixa de entrada, não importa como o provedor a chame |
| `Draft` | uma mensagem sendo escrita; `Validate` confere remetente, destinatário, endereços e o tamanho dos anexos (20 MB); `Recipients` é todo mundo, uma vez só (o envelope do SMTP, que inclui o Cco) |
| `Address` | uma pessoa ou caixa: `Display()` é o nome (ou o e-mail), `Same` ignora maiúsculas |

**Responder e encaminhar** são regras do domínio, com testes: `Reply` responde ao `Reply-To` (ou ao remetente), nunca a você mesmo, e
em "responder a todos" junta Para e Cc sem repetir ninguém; mantém a conversa (`In-Reply-To` e `References`). O assunto recebe **um
só** "Re:" ou "Fwd:", não importa quantos vieram antes (`RES:`, `ENC:`, `AW:` também são reconhecidos). A citação põe `> ` em cada linha,
com um nível a mais em cada resposta. O texto que apresenta a citação ("Em 3 jan, Ana escreveu:") vem pronto de quem chama, no idioma da pessoa.

## O leitor e o escritor de MIME

`mime.Parse` lê os bytes de uma mensagem e devolve a `Message`; `mime.Compose` faz o caminho de volta, de um `Draft` para os bytes a
enviar. Usa o `go-message` (compilado de verdade no SDK Go 1.17/ARM).

- **Texto**: pega a parte `text/plain`; se só houver HTML, converte para texto (blocos viram linhas, itens de lista ganham `•`, scripts e
  estilos somem, entidades são decodificadas). É uma visão para ler, não uma renderização fiel.
- **Codificações**: cabeçalhos com palavras codificadas (`=?UTF-8?Q?…?=`), *quoted-printable*, base64 e *charsets* antigos (ISO-8859-1 e
  outros) viram UTF-8. Um *charset* desconhecido é mostrado como veio, e **um cabeçalho ruim não esconde a mensagem**.
- **Anexos**: só nome, tipo e tamanho, **sem carregar o conteúdo**: o leitor tem 1 GB para tudo.
- **Limites**: o texto de uma parte é lido até 2 MB; a prévia da lista tem 140 caracteres.
- **Escrever**: UTF-8 em *quoted-printable*, `Message-ID` gerado do domínio do remetente (`NewMessageID`), conversa preservada, anexos em
  `multipart/mixed`. O **Cco nunca vai nos cabeçalhos**: esses destinatários só recebem pelo envelope.

## O código

| Camada | Pacote | Guarda |
|---|---|---|
| domain | `internal/mail/domain` | `Message`, `Folder`, `Draft`, `Address`, `Reply`, `Forward`, `Quote` |
| infrastructure | `internal/mail/infrastructure/mime` | `Parse`, `Compose`, a conversão de HTML em texto |

Como os outros contextos, o `mail` não conhece a plataforma nem o toolkit: o teste de arquitetura (capítulo 14) confere.
