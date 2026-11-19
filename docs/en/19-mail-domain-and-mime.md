# 19. Mail: domain and MIME

[← Accounts and secrets](18-accounts-and-secrets.md) · [Index](README.md)

The `internal/mail` context is the base of the future mail app (`apps/mail`, task T58). For now it has the **domain** and the **MIME
reader and writer**; the protocols (IMAP, SMTP, Graph, EWS), the offline cache (it needs the `store`, T54) and the screen come later.

## The domain

| Concept | What it is |
|---|---|
| `Message` | a mail as the person reads it: sender, recipients, subject, date, text, attachments (name, type and size), `Flags` (read, flagged, answered) and the thread fields (`Message-ID`, `In-Reply-To`, `References`) |
| `Folder` | where messages are kept, with a **role** (`inbox`, `sent`, `drafts`, `trash`, `junk`, `archive`): that is how the apps find the Inbox, whatever the provider calls it |
| `Draft` | a message being written; `Validate` checks sender, recipient, addresses and the size of the attachments (20 MB); `Recipients` is everyone, once (the SMTP envelope, which includes Bcc) |
| `Address` | a person or mailbox: `Display()` is the name (or the email), `Same` ignores case |

**Replying and forwarding** are domain rules, with tests: `Reply` answers the `Reply-To` (or the sender), never yourself, and on "reply all"
joins To and Cc without repeating anyone; it keeps the thread (`In-Reply-To` and `References`). The subject gets **one** "Re:" or "Fwd:", however
many came before (`RES:`, `ENC:`, `AW:` are recognized too). The quote puts `> ` on every line, one more level per reply. The text that introduces
the quote ("On 3 Jan, Ana wrote:") comes ready from the caller, in the person's language.

## The MIME reader and writer

`mime.Parse` reads the bytes of a message and returns the `Message`; `mime.Compose` goes the other way, from a `Draft` to the bytes to send. It uses
`go-message` (really built in the Go 1.17/ARM SDK).

- **Text**: it takes the `text/plain` part; with only HTML, it converts it to text (blocks become lines, list items get a `•`, scripts and styles
  vanish, entities are decoded). It is a view for reading, not a faithful rendering.
- **Encodings**: headers with encoded words (`=?UTF-8?Q?…?=`), quoted-printable, base64 and old charsets (ISO-8859-1 and others) become UTF-8. An
  unknown charset is shown as it came, and **a bad header does not hide the message**.
- **Attachments**: only name, type and size, **without loading the content**: the reader has 1 GB for everything.
- **Limits**: the text of a part is read up to 2 MB; the list preview is 140 characters.
- **Writing**: UTF-8 in quoted-printable, a `Message-ID` generated from the sender's domain (`NewMessageID`), the thread kept, attachments in
  `multipart/mixed`. **Bcc never goes in the headers**: those recipients only get the message through the envelope.

## The message cache (SQLite)

The `MessageStore` port is where the screens read from (*offline-first*); synchronization writes to it. `sqlstore` implements it in the shared
database (chapter 22), with its own migrations (module `mail`) and its tables registered so that **erasing an account** clears them.

- **Pages**: `Page` lists a folder, **newest first**, with the total, and **without the body** (a page of 20 messages does not load 20 bodies): the list
  shows the preview. `Get` returns one message with its body. The page size has a default and a cap (20 and 200).
- **Saving again replaces** the message (the same account, folder and remote id), instead of duplicating it.
- **Search**: ignores case and accents (`reuniao` finds "Reunião"), needs **every** word in the subject, the sender or the text, and a typed
  `%` or `_` is just a character, not a wildcard.
- **Folders**: `Folders` computes the total and the **unread** count on the spot (they are never stale), and `SaveFolders` follows the server: a
  folder that went away takes its messages with it.
- **Flags** (`SetFlags`: read, flagged...) and `Delete`. A damaged row is **reported**, not hidden.

## Synchronization and the server (Microsoft Graph)

The `Backend` port is the mail server of **one account**: list the folders and the recent messages, fetch a whole message, send, mark as read and
delete. `ErrUnauthorized` says the server refused the credentials (the token expired): the caller renews it and tries again.

The **`Sync`** use case connects the server to the cache: `Run` brings the folders and the **50 newest messages** of each (the rest stays on the
server), and a folder that fails **does not stop the others** (the error goes in the result). `Open` opens a message **from the cache** and only goes to
the server if the body has not come yet (and keeps it, to open offline next time); a new synchronization **does not erase the body** of a message already
read. `MarkRead` changes the cache first (the screen answers at once) and then the server, and `Delete` deletes from the server **before** the cache: if the
server refuses, the message stays on the screen.

**`graph`** implements the port for Outlook.com and Microsoft 365 over **HTTP and JSON** (without Microsoft's SDK). Folders get their role from the
well-known folders (`inbox`, `sentitems`...); recent messages come with the preview, and the whole message comes as **MIME**, read by the same
`mime.Parse`; sending posts the MIME in base64. The token comes from a function (the Accounts context renews it), a **401 becomes `ErrUnauthorized`**,
a `nextLink` to another address is **not followed** (the token never goes elsewhere), a message over 10 MB is not opened on the reader, and errors show
Graph's code and message, **never the token**. Tested against a fake HTTP server; **not yet against the real service**, and sending with **Bcc** rests on
an assumption (Graph reads the recipients from the MIME header) that still has to be checked.

## The code

| Layer | Package | Holds |
|---|---|---|
| domain | `internal/mail/domain` | `Message`, `Folder`, `Draft`, `Address`, `Reply`, `Forward`, `Quote`, the `MessageStore` and `Backend` ports, `Query` |
| application | `internal/mail/application` | `Sync`: `Run`, `Open`, `MarkRead`, `Delete` |
| infrastructure | `internal/mail/infrastructure/mime` | `Parse`, `Compose`, the HTML to text conversion |
| | `internal/mail/infrastructure/sqlstore` | the `MessageStore` on SQLite: pages, search, folders, flags |
| | `internal/mail/infrastructure/graph` | the `Backend` for Outlook.com and Microsoft 365 (Microsoft Graph) |

Like the other contexts, `mail` knows neither the platform nor the toolkit: the architecture test (chapter 14) checks it.
