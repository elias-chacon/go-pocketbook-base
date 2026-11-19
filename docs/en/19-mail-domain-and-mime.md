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

## The code

| Layer | Package | Holds |
|---|---|---|
| domain | `internal/mail/domain` | `Message`, `Folder`, `Draft`, `Address`, `Reply`, `Forward`, `Quote` |
| infrastructure | `internal/mail/infrastructure/mime` | `Parse`, `Compose`, the HTML to text conversion |

Like the other contexts, `mail` knows neither the platform nor the toolkit: the architecture test (chapter 14) checks it.
