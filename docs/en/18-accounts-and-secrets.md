# 18. Accounts and secrets

[← Productivity apps: decisions](17-productivity-apps-decisions.md) · [Index](README.md)

The `internal/accounts` context keeps the person's **accounts** (an e-mail address and what it gives access to) and their
**secrets**, for the mail, tasks and calendar apps to share: one account serves the three, and is registered once. There are no
sign-in screens yet: only the domain, the use cases and the storage (tasks T52 and T57). The shared sign-in screen came later (see below).

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

Known providers are found by the domain of the address (`PresetFor("ana@hotmail.com")` gives Outlook's). Outlook uses
OAuth2; Google, iCloud and Fastmail an **app password** (Google's OAuth flow for devices does not allow Gmail or Calendar; Gmail is mail only). An unknown provider is the `custom` preset, which starts with no services.

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

## Signing in by device code (OAuth2)

The reader has no browser, so the Microsoft account signs in by the **device code flow** (RFC 8628): the reader asks for a code
(`Authorizer.Start`), shows `microsoft.com/devicelogin` and the short code, and you type it on your phone. Meanwhile the reader asks the
server every few seconds (`Wait`): `authorization_pending` means keep waiting, `slow_down` raises the interval by 5 s, and denying,
expiring or cancelling end without tokens (`ErrDenied`, `ErrExpired`, `ErrCancelled`). The tokens received go into the vault, and `Refresh`
renews the access token without asking anything again. The device code is never shown.

Microsoft's configuration (`oauthdevice.Microsoft(clientID)`) uses the `common` tenant (personal and work accounts) and the scopes of the
apps (mail, calendar and tasks). It needs a **client ID**: register an application in Entra ID as a **public client** and turn on
"allow public client flows". Google is not here: its device flow does not allow Gmail or Calendar.

## Adding an account (the shared screen)

`Accounts.Register` turns what the person types (address, password, and the servers when the provider is unknown) into an account: the provider comes
from the address (**Gmail, iCloud...**), an unknown one needs its IMAP and SMTP servers (993 with TLS, 587 with STARTTLS), a provider that signs in with
OAuth2 (Outlook.com) is refused (it goes through `RegisterOAuth`, below), and the password goes to the vault. `internal/accounts/presentation` has the
**sign-in screen** every app of the suite shares. It starts by asking for the **provider**: Outlook.com / Microsoft 365, Gmail, iCloud or another IMAP
server. Gmail, iCloud and the others ask for the address and an **app password** (the servers only appear for another server or an unknown domain); every
failure is explained and the password never appears in a message.

**Outlook.com / Microsoft 365 signs in by code (OAuth2)**, with no password typed on the reader: the screen shows a code and an address, you open the address on
a phone or a computer and type the code, then tap **I approved it** (`CheckOAuth` asks the provider once; "not yet" is an answer, not an error).
`RegisterOAuth` then keeps the account and its tokens in the vault, and `AccessToken` hands out a valid access token, renewing it with the refresh token (and
keeping the new tokens) when it is about to expire. This needs an **app registered at Microsoft** (Entra ID, a *public client* with "allow public client
flows"): its client id goes in `applications/.pbgo/oauth.json` as `{"microsoft": "<client id>"}`. **Without that file the Outlook button explains this
instead of signing in.** Google does not allow the code flow for Gmail, so Gmail uses an app password.

## The code

| Layer | Package | Holds |
|---|---|---|
| domain | `internal/accounts/domain` | `Account`, `Endpoint`, `Secret`, `Preset`, the `AccountStore`, `Vault` and `Eraser` ports |
| application | `internal/accounts/application` | `Accounts`: add, register (from an address), update, remove, list by service, credentials, refresh tokens |
| presentation | `internal/accounts/presentation` | the sign-in screen shared by the apps |
| infrastructure | `…/infrastructure/jsonaccounts` | the `AccountStore` in `accounts.json` |
| | `…/infrastructure/securefile` | the encrypted `Vault`, the PIN and PBKDF2 |
| | `…/infrastructure/oauthdevice` | the `Authorizer`: the device code flow and token renewal, with Microsoft's configuration |

The context depends only on the shared kernel: the architecture test (chapter 14) checks it.

## Editing an account

The mail app has an **Account** button next to the search field. It opens the account editor (`internal/accounts/presentation/editor.go`): the name shown, the login (when it is not the address), a **new password** (blank keeps the stored one), the **incoming (IMAP)** and **outgoing (SMTP)** servers each with host, port and protection (**TLS** or **STARTTLS**), and the **signature** that ends every new message. **Test** connects with what is typed and says why it failed (for example "app password required"); **Remove account** asks to be tapped twice. An OAuth2 account (Outlook.com) only edits the name and the signature. POP is not offered: IMAP keeps the mail on the server and is what every provider listed here supports.

Google: Gmail does not allow its mail scope on the code flow a reader can do, so a Google account signs in with an **app password** (2-step verification on, then "App passwords" in the Google account); the normal password is refused by the server, and **Test** shows that.
