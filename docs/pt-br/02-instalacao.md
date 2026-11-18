# 2. Instalação

[← Primeiros passos](01-primeiros-passos.md) · [Índice](README.md) · [Próxima: Build e versionamento →](03-build-e-versionamento.md)

## Ambiente de desenvolvimento

| Ferramenta | Versão | Para quê |
|---|---|---|
| Go | 1.17+ (qualquer versão mais nova serve no seu PC) | testes, preview, ferramentas |
| Git | qualquer | clonar e commits |
| Docker | qualquer recente | compilar para o leitor (ARM + SDK do PocketBook) |
| WSL 2 (só no Windows) | opcional | Docker dentro do Linux, se você não usa o Docker Desktop |

> **Regra do Go 1.17.** O compilador dentro da imagem Docker é o Go 1.17.13 (o último que funciona com o
> linker antigo do SDK). O código que vai para o app não pode usar generics (`[T any]`), `any` nem funções
> novas da biblioteca padrão. O seu PC pode ter um Go mais novo, mas escreva o código do app no dialeto 1.17.

### Confira o seu ambiente

```sh
go version          # go1.17 ou mais novo
docker version      # (ou: wsl -e docker version)
go test ./...       # tudo passa
```

## Instalando o app no leitor

O PocketBook **não tem instalador**: um app é apenas um arquivo na pasta `applications`. Este projeto vai um passo
além e o app **instala o próprio ícone**.

### 1. Copie o arquivo

1. Compile ([Build e versionamento](03-build-e-versionamento.md)) — você obtém um único arquivo,
   `dist/applications/hello.app`.
2. Conecte o leitor por USB e escolha **PC link**; ele aparece como uma unidade (por exemplo `F:`).
3. Copie o arquivo para `<unidade>/applications/`, ou deixe o script fazer isso:

   ```powershell
   .\scripts\install.ps1 -Drive F:              # a demo (hello)
   .\scripts\install.ps1 -Drive F: -App meuapp  # o seu app
   ```


4. Ejete o leitor com segurança.

### 2. Abra uma vez

No leitor, abra **Aplicativos** e toque no app (ele aparece pelo nome do arquivo, por exemplo *hello*). Nesta
**primeira execução** o app:

1. desenha os dois ícones (normal e "pressionado", 128×128, BMP de 8 bits) com o mesmo código vetorial das
   telas e os salva em `applications/icons/`;
2. se registra na configuração da tela inicial do leitor, criando uma entrada para ele e listando-o no grupo
   **General**;
3. mostra uma mensagem no rodapé da tela dizendo o que fez.

O arquivo de configuração é `system/profiles/default/config/desktop/view.json` (`system/config` é um link para a
mesma pasta). Antes de alterá-lo o app faz um **backup** (`view.json.bak-<data-hora>`), confere que o novo texto é
um JSON válido e troca o arquivo de uma vez. Rodar o app de novo não muda nada; se uma entrada de uma versão
antiga estiver com o caminho errado, ele corrige.

### 3. Desconecte o cabo

A tela inicial só relê a configuração quando o cabo USB é desconectado (ou o leitor reinicia). Depois disso o
ícone **Hello Go** aparece no grupo *General* da tela de Aplicativos.

```text
copiar o .app  →  abrir uma vez  →  desconectar o USB  →  ícone na tela inicial
```

## Atualizando

Copie o novo `.app` por cima do antigo. O app percebe que já está registrado e não mexe na configuração. O
rodapé da demo mostra a versão em execução (`vMAJOR.MINOR.BUILD`), então dá para confirmar qual build está no
leitor.

## Desinstalando

1. Apague `applications/hello.app` e `applications/icons/hello.bmp` / `hello_f.bmp`.
2. Remova a entrada da tela inicial: restaure o backup
   `system/profiles/default/config/desktop/view.json.bak-<data-hora>` sobre o `view.json`, ou apague as entradas
   `U_hello` à mão (uma em `"applications"` e outra na lista `"apps"` de um grupo).
3. Desconecte o cabo para a tela inicial recarregar.

> O primeiro backup de um leitor que nunca rodou o app é o arquivo original de fábrica, então é ele que você
> restaura se quiser tudo exatamente como estava.

## O que é gravado no leitor

| Caminho | O quê | Quando |
|---|---|---|
| `applications/<nome>.app` | o app | você copia |
| `applications/icons/<nome>.bmp`, `<nome>_f.bmp` | ícones de lançamento | primeira execução |
| `system/profiles/default/config/desktop/view.json` | entrada na tela inicial | primeira execução |
| `system/profiles/default/config/desktop/view.json.bak-*` | backup antes da alteração | primeira execução |
| `applications/hello.log` | log de eventos para depuração (só a demo) | a cada execução |

Próxima: [Build e versionamento →](03-build-e-versionamento.md)
