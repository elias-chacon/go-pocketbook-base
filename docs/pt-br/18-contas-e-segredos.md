# 18. Contas e segredos

[← Apps de produtividade: decisões](17-decisoes-apps-de-produtividade.md) · [Índice](README.md)

O contexto `internal/accounts` guarda as **contas** da pessoa (um endereço de e-mail e aquilo a que ele dá acesso) e os seus
**segredos**, para os apps de e-mail, tarefas e calendário compartilharem: uma conta serve os três, e é cadastrada uma só vez.
Ainda não há telas de cadastro: só o domínio, os casos de uso e o armazenamento (tarefas T52 e T57).

## O domínio

| Conceito | O que é |
|---|---|
| `Account` | uma identidade num provedor: `id` (slug único), nome, endereço, provedor, autenticação e os **serviços** |
| `Service` | para que a conta serve: `mail`, `calendar` ou `tasks` |
| `Endpoint` | como um serviço é alcançado: `imap`, `smtp`, `graph` (Microsoft), `ews` (Exchange local), `caldav`, `google` |
| `Secret` | a senha ou os tokens OAuth2. **Nunca aparece num log**: `fmt` mostra `Secret{***}` em qualquer formato, e um teste garante |
| `Preset` | o que se sabe de um provedor (Outlook.com e Microsoft 365, Google, iCloud, Fastmail): a pessoa digita o endereço e a senha, não nomes de servidor |

Regras (`Account.Validate`, que devolve todos os problemas de uma vez): o `id` é um slug; o endereço tem `@`; **conexão sempre
protegida** (IMAP e SMTP só com `tls` ou `starttls`, CalDAV e EWS só com `https`); o e-mail por IMAP precisa dos dois endpoints (IMAP
e SMTP); `graph` e `google` não têm o que configurar.

Os provedores conhecidos são achados pelo domínio do endereço (`PresetFor("ana@hotmail.com")` dá o do Outlook). O Outlook usa
OAuth2; o Google, o iCloud e o Fastmail, uma **senha de app** (o fluxo OAuth do Google para dispositivos não permite o Gmail nem o Calendar; o Gmail é só e-mail). Um provedor desconhecido é o preset `custom`, que começa sem serviços.

## Os casos de uso (`Accounts`)

`Add` valida, confere que o `id` está livre e guarda o segredo e a conta (se um passo falhar, o outro é desfeito: não sobra segredo
sem conta nem conta sem segredo). `Remove` apaga o segredo, os dados em cache (porta `Eraser`) e **por último** a conta, então uma
remoção que falhou no meio pode ser repetida. `Credentials` entrega a conta com o segredo a quem vai conectar, e `Refresh` guarda
os tokens renovados.

## Onde fica no leitor

| Arquivo | Conteúdo |
|---|---|
| `applications/.pbgo/accounts.json` | as contas (sem segredos), por nome, gravado por um arquivo temporário |
| `applications/.pbgo/secrets.bin` | o cofre: os segredos cifrados com **AES-256-GCM** |
| `applications/.pbgo/device.key` | a chave do cofre quando **não há PIN** (permissão 0600) |

## O que o cofre protege, com franqueza

- **Com PIN** (`SetPIN`): a chave vem do PIN (PBKDF2-HMAC-SHA-256, 20 000 iterações) e o arquivo é inútil sem ele. O app pede o PIN
  (`Unlock`); um PIN errado é `ErrLocked`, e o cofre bloqueado não aceita nem leitura nem gravação.
- **Sem PIN**: a chave é um arquivo aleatório ao lado do cofre. Isso só protege contra ler os segredos por engano ou copiar o
  `secrets.bin` sozinho. Quem lê os dois arquivos abre o cofre, porque **o leitor não tem *keystore***. Os apps devem dizer isso na tela.
- Cada gravação usa um *nonce* novo; um arquivo adulterado (um bit trocado) é recusado; sem a chave, o cofre fica fechado e **não é
  sobrescrito** por um novo.
- O PBKDF2 está escrito aqui (a biblioteca padrão só o tem em Go recente) e é conferido contra os vetores de teste da RFC 7914.

## Entrar por código de dispositivo (OAuth2)

O leitor não tem navegador, então a conta da Microsoft entra pelo **fluxo de código de dispositivo** (RFC 8628): o leitor pede um código
(`Authorizer.Start`), mostra `microsoft.com/devicelogin` e o código curto, e você o digita no celular. Enquanto isso o leitor consulta o
servidor a cada poucos segundos (`Wait`): `authorization_pending` é esperar, `slow_down` aumenta o intervalo em 5 s, e negar, expirar ou
cancelar terminam sem tokens (`ErrDenied`, `ErrExpired`, `ErrCancelled`). Os tokens recebidos vão para o cofre, e `Refresh` renova o token de
acesso sem pedir nada de novo. O código do dispositivo nunca é mostrado.

A configuração da Microsoft (`oauthdevice.Microsoft(clientID)`) usa o *tenant* `common` (contas pessoais e de trabalho) e os escopos dos
apps (e-mail, calendário e tarefas). Ela precisa de um **client ID**: registre um aplicativo no Entra ID como **cliente público** e ligue
"permitir fluxos de cliente público". O Google não entra aqui: o fluxo por dispositivo dele não permite o Gmail nem o Calendar.

## O código

| Camada | Pacote | Guarda |
|---|---|---|
| domain | `internal/accounts/domain` | `Account`, `Endpoint`, `Secret`, `Preset`, as portas `AccountStore`, `Vault` e `Eraser` |
| application | `internal/accounts/application` | `Accounts`: adicionar, atualizar, remover, listar por serviço, credenciais, renovar tokens |
| infrastructure | `…/infrastructure/jsonaccounts` | o `AccountStore` em `accounts.json` |
| | `…/infrastructure/securefile` | o `Vault` cifrado, o PIN e o PBKDF2 |
| | `…/infrastructure/oauthdevice` | o `Authorizer`: o fluxo de código de dispositivo e a renovação, com a configuração da Microsoft |

O contexto só depende do núcleo compartilhado (`kernel`): o teste de arquitetura (capítulo 14) confere.
