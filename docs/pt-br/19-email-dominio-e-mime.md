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

## O cache de mensagens (SQLite)

A porta `MessageStore` é de onde as telas leem (*offline-first*); a sincronização escreve nela. O `sqlstore` a implementa no banco
compartilhado (capítulo 22), com as suas próprias migrações (módulo `mail`) e as tabelas registradas para **apagar a conta**.

- **Páginas**: `Page` lista uma pasta, **da mais nova para a mais antiga**, com o total, e **sem o corpo** (uma página de 20 mensagens não carrega
  20 corpos): a lista mostra a prévia. `Get` traz uma mensagem com o corpo. O tamanho da página tem padrão e limite (20 e 200).
- **Salvar de novo substitui** a mensagem (a mesma conta, pasta e id remoto), em vez de duplicá-la.
- **Busca**: ignora maiúsculas e acentos (`reuniao` acha "Reunião"), exige **todas** as palavras no assunto, no remetente ou no texto, e um
  `%` ou `_` digitado é só um caractere, não um curinga.
- **Pastas**: `Folders` calcula o total e os **não lidas** na hora (nunca ficam desatualizados), e `SaveFolders` segue o servidor: uma pasta
  que sumiu leva as suas mensagens junto.
- **Marcas** (`SetFlags`: lida, sinalizada...) e `Delete`. Uma linha danificada é **reportada**, não escondida.

## Sincronização e o servidor (Microsoft Graph)

A porta `Backend` é o servidor de e-mail de **uma conta**: listar as pastas e as mensagens recentes, buscar uma mensagem inteira, enviar, marcar
como lida e apagar. `ErrUnauthorized` diz que o servidor recusou as credenciais (o token venceu): quem chamou o renova e tenta de novo.

O caso de uso **`Sync`** liga o servidor ao cache: `Run` traz as pastas e as **50 mensagens mais novas** de cada (o resto fica no servidor), e uma pasta
que falha **não para as outras** (o erro vai no resultado). `Open` abre uma mensagem **do cache** e só vai ao servidor se o corpo ainda não veio (e
guarda, para abrir offline da próxima vez); uma nova sincronização **não apaga o corpo** de uma mensagem já lida. `MarkRead` muda o cache primeiro (a tela
responde na hora) e depois o servidor, e `Delete` apaga do servidor **antes** do cache: se o servidor recusar, a mensagem continua na tela.

O **`graph`** implementa a porta para o Outlook.com e o Microsoft 365 por **HTTP e JSON** (sem o SDK da Microsoft). As pastas ganham o papel pelas
pastas conhecidas (`inbox`, `sentitems`...); as mensagens recentes vêm com a prévia, e a mensagem inteira vem como **MIME**, lida pelo mesmo
`mime.Parse`; o envio manda o MIME em base64. O token vem de uma função (o contexto de contas o renova), um **401 vira `ErrUnauthorized`**, um link
`nextLink` para outro endereço **não é seguido** (o token nunca vai para outro lugar), uma mensagem com mais de 10 MB não é aberta no leitor, e os erros
mostram o código e a mensagem do Graph, **nunca o token**. Testado contra um servidor HTTP de mentira; **ainda não contra o serviço de verdade**, e o
envio com **Cco** parte de uma suposição (o Graph lê os destinatários do cabeçalho do MIME) que precisa ser conferida.

## Qualquer servidor IMAP (e SMTP)

O **`imapmail`** é a mesma `Backend` para Gmail, iCloud, Yahoo, Fastmail ou um servidor de empresa. Ele mantém **uma conexão** enquanto o app fala com o
servidor e a descarta no primeiro erro de rede (uma resposta do servidor, como "pasta não existe", a mantém), então a próxima chamada começa limpa. O login é
uma função: `Password` (LOGIN, uma senha de app) ou `SASL` com **XOAUTH2** (pacote `xoauth2`, o token vem de uma função para poder ser renovado). Um
login recusado vira `ErrUnauthorized` e **nunca leva a senha**. O papel das pastas vem dos **atributos de uso especial** (`Sent`, `Trash`...) e, sem eles,
dos nomes comuns em português e inglês. `Recent` lê só os **cabeçalhos** das mensagens mais novas; `Fetch` lê a mensagem inteira (até 10 MB); `Delete` copia
para a Lixeira e expurga, ou apaga de vez se a mensagem já está na Lixeira.

O **`smtpsend`** envia por SMTP com o `net/smtp` (STARTTLS na 587, TLS na 465), a partir do envelope (Para, Cc e **Cco**) e dos bytes que o `mime.Compose`
montou, então o Cco nunca vai no cabeçalho. O `imapmail` compõe o rascunho, entrega ao `Sender` e, com `SaveSent`, grava uma cópia em Enviados (Gmail e
Outlook fazem isso sozinhos). Testado contra o servidor IMAP em memória do go-imap e um servidor SMTP minúsculo; **ainda não contra um provedor de verdade**.

## O app E-mail

O `apps/mail` é a raiz de composição: abre o banco compartilhado, as contas e o cofre em `applications/.pbgo` e mostra **um só documento** que é a tela de
login enquanto não há conta de e-mail e a caixa de entrada depois. O caso de uso **`Mailbox`** é o `Sync` mais o que as telas leem do cache (pastas com
contagens, uma página de uma pasta, busca). A tela (`presentation`, só widgets do toolkit) mostra o título da pasta com as não lidas, o menu **Pastas**,
**Sincronizar**, a busca, **7 mensagens por página** (remetente, data, assunto e prévia; não lidas em negrito) e as teclas de página. Tocar numa mensagem a abre
**página a página** (`TextPager`) e a marca como lida; há **Marcar não lida** e **Apagar** (servidor primeiro: se ele recusar, a mensagem continua aberta).
Tudo é lido do cache, então funciona offline; **Sincronizar** traz as 50 mais novas de cada pasta. A conta precisa ser de **senha de app** por enquanto (o
OAuth2 vem com o login por código). Falta (T62): escrever, responder, encaminhar, anexos e marcadores. **Ainda não testado no aparelho.**

## O código

| Camada | Pacote | Guarda |
|---|---|---|
| domain | `internal/mail/domain` | `Message`, `Folder`, `Draft`, `Address`, `Reply`, `Forward`, `Quote`, as portas `MessageStore` e `Backend`, `Query` |
| application | `internal/mail/application` | `Sync`: `Run`, `Open`, `MarkRead`, `Delete`; `Mailbox`: o `Sync` mais a leitura do cache |
| presentation | `internal/mail/presentation` | as telas do e-mail: lista, pastas, leitura |
| infrastructure | `internal/mail/infrastructure/mime` | `Parse`, `Compose`, a conversão de HTML em texto |
| | `internal/mail/infrastructure/sqlstore` | o `MessageStore` no SQLite: páginas, busca, pastas, marcas |
| | `internal/mail/infrastructure/graph` | o `Backend` do Outlook.com e do Microsoft 365 (Microsoft Graph) |
| | `internal/mail/infrastructure/imapmail` | o `Backend` de qualquer servidor IMAP (senha ou XOAUTH2) |
| | `internal/mail/infrastructure/smtpsend` | o envio por SMTP (STARTTLS, TLS) |
| | `internal/mail/infrastructure/xoauth2` | o login XOAUTH2 para IMAP e SMTP |

Como os outros contextos, o `mail` não conhece a plataforma nem o toolkit: o teste de arquitetura (capítulo 14) confere.
