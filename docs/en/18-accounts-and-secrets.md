# 18. Accounts and secrets

[← Productivity apps: decisions](17-productivity-apps-decisions.md) · [Index](README.md)

The `internal/accounts` context keeps the person's **accounts** (an e-mail address and what it gives access to) and their
**secrets**, for the mail, tasks and calendar apps to share: one account serves the three, and is registered once. There are no
sign-in screens yet: only the domain, the use cases and the storage (tasks T52 and T57).

## The domain

| Concept | What it is |
|---|---|
| `Account` | an identity at a provider: `id` (a unique slug), name, address, provider, authentication and the **services** |
| `Service` | what the account is for: `mail`, `calendar` or `tasks` |
| `Endpoint` | how a service is reached: `imap`, `smtp`, `graph` (Microsoft), `ews` (on-premises Exchange), `caldav`, `google` |
| `Secret` | the password or the OAuth2 tokens. **It never shows up in a log**: `fmt` prints `Secret{***}` in every format, and a test checks it |
| `Preset` | what is known about a provider (Outlook.com and Microsoft 365, Google, iCloud, Fastmail): the person types an address and a password, not server names |

Rules (`Account.Validate`, which returns every problem at once): the `id` is a slug; the address has an `@`; **connections are always
protected** (IMAP and SMTP only with `tls` or `starttls`, CalDAV and EWS only with `https`); mail over IMAP needs both endpoints (IMAP and
SMTP); `graph` and `google` have nothing to configure.

Known providers are found by the domain of the address (`PresetFor("ana@hotmail.com")` gives Outlook's). Outlook and Google use
OAuth2; iCloud and Fastmail an **app password**. An unknown provider is the `custom` preset, which starts with no services.

## The use cases (`Accounts`)

`Add` validates, checks that the `id` is free and keeps the secret and the account (if one step fails the other is undone: no secret
without an account, no account without a secret). `Remove` deletes the secret, the cached data (the `Eraser` port) and the account **last**,
so a removal that failed midway can be repeated. `Credentials` hands the account with its secret to whoever is about to connect, and
`Refresh` keeps renewed tokens.

## Where it lives on the reader

| File | Content |
|---|---|
| `applications/.pbgo/accounts.json` | the accounts (no secrets), by name, written through a temporary file |
| `applications/.pbgo/secrets.bin` | the vault: the secrets encrypted with **AES-256-GCM** |
| `applications/.pbgo/device.key` | the vault's key when there is **no PIN** (permission 0600) |

## What the vault protects, honestly

- **With a PIN** (`SetPIN`): the key comes from the PIN (PBKDF2-HMAC-SHA-256, 20,000 iterations) and the file is useless without it. The
  app asks for the PIN (`Unlock`); a wrong PIN is `ErrLocked`, and a locked vault accepts neither reads nor writes.
- **Without a PIN**: the key is a random file next to the vault. That only protects against reading the secrets by accident or copying
  `secrets.bin` alone. Whoever reads both files opens the vault, because **the reader has no keystore**. The apps must say so on screen.
- Every save uses a fresh nonce; a tampered file (one flipped bit) is refused; without its key the vault stays closed and is **not
  overwritten** by a new one.
- PBKDF2 is written here (the standard library only has it in recent Go) and is checked against the test vectors of RFC 7914.

## The code

| Layer | Package | Holds |
|---|---|---|
| domain | `internal/accounts/domain` | `Account`, `Endpoint`, `Secret`, `Preset`, the `AccountStore`, `Vault` and `Eraser` ports |
| application | `internal/accounts/application` | `Accounts`: add, update, remove, list by service, credentials, refresh tokens |
| infrastructure | `…/infrastructure/jsonaccounts` | the `AccountStore` in `accounts.json` |
| | `…/infrastructure/securefile` | the encrypted `Vault`, the PIN and PBKDF2 |

The context depends only on the shared kernel: the architecture test (chapter 14) checks it.
