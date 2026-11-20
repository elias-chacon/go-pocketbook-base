# 24. O app Bíblia

O app **Bíblia** (`apps/bible`, contexto `internal/bible`) lê a Bíblia em várias versões e idiomas, acompanha o progresso de
leitura e mostra o **versículo do dia**. As versões vêm da API pública [Midvash](https://api.midvash.com) (sem chave, 119 versões
em 69 idiomas), são **baixadas uma vez** para o banco SQLite do leitor e depois lidas **sem internet**. Segue o contrato dos apps
(`apps/hello` é o modelo) e tem as mesmas três abas do Hello Go.

## Como usar

| Aba | O que tem |
|---|---|
| **Início** | a porcentagem de leitura da Bíblia (`0%` a `100%`), o card **Versículo do dia** com a caixa "Já li" e o botão para **continuar a leitura** de onde parou |
| **Bíblia** | a barra de seleção (versão atual, idioma, versão, busca) e embaixo os livros, os capítulos, os números dos versículos e a leitura |
| **Mais** | **Configurações**, **Sobre** e **Sair** |

**Primeira vez.** Sem nenhuma versão no leitor, o Início oferece "Baixar a Bíblia": um toque busca a lista de versões, baixa a do
idioma do aparelho (Almeida Livre em português, King James em inglês) e começa. Depois, na aba **Bíblia**, escolha outro idioma e
outra versão; uma versão que ainda não está no leitor pergunta antes de baixar (são alguns megabytes, em cerca de 60 pedidos).

**Ler.** Livro → capítulo (a lista mostra o início de cada capítulo, ou só o número, conforme as configurações) → a grade com os
números dos versículos, para saltar direto a um deles, ou **Ler o capítulo**. A leitura é por páginas (as teclas de página do
leitor também viram); cada versículo que **aparece na tela** conta como lido, e as setas do alto passam ao capítulo anterior ou
seguinte, mesmo de um livro para outro. A lista de livros vem na ordem original e o botão **Ordem alfabética** alterna.

**Versículo do dia.** Em datas especiais (Natal e a véspera, Ano-Novo, Epifania, Domingo de Ramos, Sexta-feira Santa, Páscoa,
Ascensão e Pentecostes, com a Páscoa calculada para cada ano) o versículo é o da data; nos outros dias é sorteado da versão em uso,
sempre o mesmo durante o dia e guardado, então não muda se você trocar de versão. Marcar "Já li" troca a caixa pelo ícone de
concluído e soma o versículo ao progresso. (A tela do leitor é em tons de cinza: o "verde" é o quadrado escuro com o visto.)

**Configurações.** Tamanho da fonte (24 a 56, de 2 em 2, com a prévia ao vivo), tipo de fonte (Go, Go Medium, Go Mono), alto
contraste (inverte as cores), leitura em voz alta, prévia dos capítulos e idioma do app (português ou inglês; começa no do aparelho).

## O que fica guardado

Tudo em `applications/.pbgo/bible.db`, um SQLite (o mesmo `internal/storage` dos outros apps, com as migrações do módulo `bible`):

| Tabela | Conteúdo |
|---|---|
| `bible_books`, `bible_versions` | o catálogo (livros com nomes em pt-BR e en; versões, com o que já foi baixado) |
| `bible_verses` | o texto, **um versículo por linha**, chave `(versão, livro, capítulo, versículo)` |
| `bible_read` | o que foi lido, **só pelo lugar** `(livro, capítulo, versículo)`, sem a versão |
| `bible_state` | a posição (último livro, capítulo e versículo) e as preferências |
| `bible_daily` | o versículo escolhido para cada dia (guarda um mês) |

**O progresso é unificado.** Como `bible_read` não conhece a versão, ler João 3:16 em qualquer tradução vale para todas. A porcentagem
é `lidos ÷ total` **da versão em uso**, contando só os versículos que ela tem: uma versão com livros ou capítulos a mais ou a menos
muda o denominador na mesma proporção (e o que foi lido em outra versão só entra quando a versão ativa tem aquele versículo).

## Memória (o leitor tem 1 GB)

- O texto **nunca** é carregado inteiro: a tela guarda a lista de livros e de versões (alguns KB) e, na leitura, **um capítulo** (no
  máximo 176 versículos). Cada página é montada de novo ao ser mostrada.
- O download traz **20 capítulos por pedido** (`/v1/passages`), grava cada lote numa transação e solta a memória; se for
  interrompido, recomeça dos capítulos que faltam.
- A busca lê as linhas **uma a uma** e para em 60 resultados, ignorando acentos e maiúsculas: o consumo não depende do tamanho da versão.
- O cache de páginas do SQLite é de ~2 MB (`internal/storage`).

## Como é feito

| Camada | Pacote | O que tem |
|---|---|---|
| domínio | `internal/bible/domain` | `Book`, `Version`, `Ref`, `ChapterText`, `Preferences`, o `Canon` (os 66 livros, para listar antes de baixar nada), `FestiveVerse`, `Easter`, e as portas `Catalog`, `Library` e `Speaker` |
| aplicação | `internal/bible/application` | `Bible`: atualizar o catálogo, baixar, ler um capítulo, contar o lido, retomar, versículo do dia, buscar |
| infraestrutura | `internal/bible/infrastructure` | `midvash` (a API por HTTP+JSON) e `sqlstore` (o texto e o progresso no banco) |
| apresentação | `internal/bible/presentation` | as telas, só com widgets do toolkit |
| raiz | `apps/bible` | liga tudo e escolhe o aparelho: cliente HTTP, banco, idioma, voz |

Para a leitura foram criados no toolkit (genéricos, sem saber nada da Bíblia): o **ParagraphPager** ([capítulo 6](06-componentes.md)),
o tipo de letra escolhido por `gfx.SetFamily` e a tela em negativo com `Document.SetInvert`.

## O que foi verificado e o que falta

- **Testado no PC:** domínio, casos de uso, banco (em arquivo e na memória), o cliente da API contra um servidor de mentira, as telas
  (cada página cabe na tela e cada elemento recebe o seu toque) e o ciclo completo no `apps/bible`: baixar → ler → fechar → reabrir
  sem rede, com a posição, o progresso, o versículo lido e as configurações no lugar.
- **Compilado e testado em ARMv7** (`.\scripts\arm-test.ps1`, Go 1.17 + qemu) e compilado com o SDK do leitor.
- **O idioma do leitor** (conferido num PB741 e no SDK): o firmware o guarda no `global.cfg` do armazenamento do usuário
  (`/mnt/ext1/system/config/global.cfg`, `CONFIGPATH` do `inkview.h`), como `language=pt`. O app lê do modo que o SDK oferece
  (`GetGlobalConfig` + `ReadString`, `inkview.GlobalConfigString`), depois o próprio arquivo, depois `LANG`/`LC_*` e por último o idioma
  das configurações compartilhadas dos apps. (`/ebrmain` é a área só de leitura do firmware; as configurações não ficam lá.)
- **Não testado no aparelho:** a velocidade do SQLite e do download no PB741 real e o consumo de memória medido lá.
- **Leitura em voz alta:** o SDK do PocketBook (6.3, `inkview.h`) **não oferece uma API de voz para apps** (só o enum `PB_TTS_STATE` e
  um `hw_isttsplaying` interno). As vozes do próprio firmware estão no leitor (`system/tts/<voz>/`, ex. `pt_br_rafael`, `pt_br_helena`,
  `en_us_julie`; `ttslanguage=`, `ttsspeed=` e `tts_volume=` no `global.cfg`): é o motor **ReadSpeaker VT** (`libvtapi.so`,
  `librsttswrapper.so`, funções `VT_LOADTTS_RS`/`VT_PLAYTTS_RS`), sem header público, com verificação de licença e carregado pelo
  aplicativo de leitura do leitor. Chamá-lo de um app seria engenharia reversa, então não foi feito. A opção de Configurações existe e o
  botão **Ouvir** aparece, mas ele só fala se o leitor tiver um programa de voz (`flite`, `espeak-ng` ou `espeak`); sem eles o app avisa
  em vez de fingir. Quando houver um caminho no aparelho, só o `Speaker` de `apps/bible/device.go` muda.
- **Cores:** o canvas do PB741 é de 24 bits (tela colorida Kaleido), mas o toolkit desenha e envia tons de cinza de 8 bits
  (`inkview.Blit`), então o "verde" do ícone de concluído é um quadrado escuro. Cores seriam uma mudança do toolkit (um pipeline RGB),
  não deste app.
- **Cânones diferentes:** o download pede os 1189 capítulos dos 66 livros; uma versão que tenha capítulos além desses (ex.: `cpdv`,
  1197) ou livros deuterocanônicos fica sem os extras (o denominador já acompanha o que foi baixado).
