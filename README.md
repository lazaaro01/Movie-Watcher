# 🎬 Movie Watcher CLI

Monitore sessões de cinema (Cinépolis, e outros parceiros do **Ingresso.com**) e receba alertas no **Telegram** quando houver mudanças de preço ou disponibilidade — em um ou vários cinemas ao mesmo tempo.

---

## 📋 Sumário

- [Pré-requisitos](#pré-requisitos)
- [Instalação](#instalação)
- [Configuração do Telegram](#configuração-do-telegram)
- [Primeiros passos](#primeiros-passos)
- [Cinemas monitorados](#cinemas-monitorados)
- [Comandos](#comandos)
- [Monitoramento contínuo](#monitoramento-contínuo)
- [Resumo diário](#resumo-diário)
- [Histórico de preços](#histórico-de-preços)
- [Deploy em produção](#deploy-em-produção)
- [FAQ / Solução de problemas](#faq--solução-de-problemas)
- [Detalhes técnicos e mudanças recentes](#detalhes-técnicos-e-mudanças-recentes)

---

## Pré-requisitos

- **Go** 1.25+ instalado ([download](https://go.dev/dl/))
- Uma conta no **Telegram**
- Acesso ao **BotFather** no Telegram

---

## Instalação

```bash
# Clone ou entre na pasta do projeto
cd movie-watcher

# Baixar dependências
go mod tidy

# Build do executável
go build -o movie-watcher.exe .
```

---

## Configuração do Telegram

### 1. Criar o bot no BotFather

1. Abra o Telegram e procure por **@BotFather**
2. Envie `/newbot` e siga as instruções
3. Escolha um nome (ex: `Movie Watcher Fortaleza`)
4. Escolha um username (ex: `meu_movie_watcher_bot`)
5. O BotFather vai te dar um **token** — guarde-o

### 2. Descobrir seu Chat ID

1. No Telegram, procure pelo bot que você criou e envie `/start`
2. Depois, procure por **@userinfobot** e envie qualquer mensagem
3. Ele vai responder com seu ID numérico (ex: `123456789`)
4. **Importante**: é o ID do usuário, não o nome do bot

### 3. Configurar o arquivo config.json

```bash
# Gera o arquivo config.json padrão
movie-watcher.exe config init
```

Edite o `config.json` com seus dados:

```json
{
  "telegram": {
    "bot_token": "7890123456:AAHd...sEU_token_AQUI",
    "chat_id": 123456789
  },
  "monitor": {
    "interval_minutes": 5
  },
  "database": {
    "path": "./movie-watcher.db"
  },
  "city_id": 36,
  "theater_id": 1446,
  "partnership": "cinepolis"
}
```

`city_id`, `theater_id` e `partnership` funcionam como o cinema **padrão**:
usados sempre que nenhum cinema for explicitamente marcado com `cinemas add`
(veja [Cinemas monitorados](#cinemas-monitorados)).

> **Alternativa mais segura para o token**: em vez de deixar `bot_token` no
> `config.json`, você pode configurar tudo por variável de ambiente — não
> precisa nem existir um `config.json` no diretório. Prefixo `MW_` + o
> caminho do campo em maiúsculas, trocando "." por "_":
> ```
> MW_TELEGRAM_BOT_TOKEN=...
> MW_TELEGRAM_CHAT_ID=...
> MW_DATABASE_PATH=./movie-watcher.db
> MW_CITY_ID=36
> MW_THEATER_ID=1446
> MW_PARTNERSHIP=cinepolis
> ```
> Essencial pra rodar em containers/CI, onde normalmente não se quer
> guardar segredo em disco (veja [Deploy em produção](#deploy-em-produção)).

### 4. Testar a notificação

```bash
movie-watcher.exe notify test
```

Se tudo estiver correto, você receberá uma mensagem no Telegram.

---

## Primeiros passos

### Listar filmes em cartaz

```bash
movie-watcher.exe movies list
```

Saída esperada:
```
  ID  TÍTULO                      GÊNERO     DURAÇÃO   CLASSIFICAÇÃO
  1   Mestres Do Universo         Ação       133 min   14 anos
  2   O Diabo Veste Prada 2       Comédia    120 min   12 anos
  3   Todo Mundo Em Pânico (2026) Comédia    96 min    18 anos
  ...
```

Para forçar atualização dos dados da API:
```bash
movie-watcher.exe movies list --refresh
# ou de forma abreviada:
movie-watcher.exe movies list -r
```

### Buscar filme por nome

```bash
movie-watcher.exe movies search "Mestres"
```

### Ver sessões de um filme

```bash
movie-watcher.exe sessions list "Mestres do Universo"
```

Saída esperada:
```
  ID  FILME                  SALA    HORÁRIO        PREÇO      STATUS
  4   Mestres Do Universo    Sala 1  08/06 18:30    R$ 34.20   Disponível
  5   Mestres Do Universo    Sala 1  08/06 21:30    R$ 34.20   Disponível
  6   Mestres Do Universo    Sala 5  08/06 20:00    R$ 34.20   Disponível
```

> **Anote o `ID` da sessão** (primeira coluna) — você vai usá-lo para monitorar.
> Diferente de versões antigas, esse ID agora **permanece estável** entre
> execuções de `sessions list` (veja [detalhes técnicos](#detalhes-técnicos-e-mudanças-recentes)).

Se você tiver mais de um cinema marcado em `cinemas add`, uma coluna extra
**Cinema** aparece na tabela para indicar de qual cinema é cada sessão.

---

## Cinemas monitorados

Por padrão o CLI opera sobre um único cinema, o definido em `theater_id` no
`config.json`. Se você quiser acompanhar **vários cinemas da mesma cidade**
ao mesmo tempo, use os comandos `cinemas`:

```bash
# 1. Descobrir os cinemas disponíveis na cidade configurada (city_id)
movie-watcher.exe cinemas sync

# 2. Ver a lista (com o ID interno de cada um)
movie-watcher.exe cinemas list
```

Saída esperada:
```
  ID  NOME                          BAIRRO         MONITORADO
  1   Cinépolis RioMar Kennedy      Kennedy        Não
  2   Cinemark Iguatemi             Papicu         Não
```

```bash
# 3. Marcar os cinemas que você quer acompanhar
movie-watcher.exe cinemas add 1
movie-watcher.exe cinemas add 2

# Para parar de acompanhar um cinema:
movie-watcher.exe cinemas remove 2
```

A partir do momento em que **pelo menos um** cinema é marcado com
`cinemas add`, os comandos `sessions list`, `monitor start-movie` e
`monitor check` passam a considerar **todos os cinemas marcados**
automaticamente — não é mais necessário informar nada extra. Se nenhum
cinema for marcado, o comportamento padrão (usar `theater_id` do
`config.json`) continua funcionando exatamente como antes.

Para restringir uma busca a um único cinema específico (mesmo com vários
marcados), use `--cinema <id>`:

```bash
movie-watcher.exe sessions list "Mestres do Universo" --cinema 1
```

---

## Comandos

### Filmes

| Comando | Descrição |
|---------|-----------|
| `movies list` | Lista filmes em cartaz (cache local) |
| `movies list --refresh` | Lista filmes forçando busca na API |
| `movies list --notify` | Também envia a lista para o Telegram |
| `movies search <nome>` | Busca filme por nome |

### Cinemas

| Comando | Descrição |
|---------|-----------|
| `cinemas sync` | Busca a lista de cinemas disponíveis na cidade configurada |
| `cinemas list` | Lista cinemas conhecidos e quais estão sendo monitorados |
| `cinemas search <nome>` | Busca cinema por nome |
| `cinemas add <id>` | Marca um cinema (ID interno) para ser monitorado |
| `cinemas remove <id>` | Remove um cinema do monitoramento |

### Sessões

| Comando | Descrição |
|---------|-----------|
| `sessions list <filme>` | Lista sessões de um filme nos cinemas monitorados (ou no padrão do config) |
| `sessions list <filme> --cinema <id>` | Restringe a busca a um único cinema |
| `sessions list <filme> --notify` | Também envia o resultado para o Telegram |

### Monitoramento

| Comando | Descrição |
|---------|-----------|
| `monitor start <session_id>` | Inicia monitoramento de uma sessão |
| `monitor start-movie <filme>` | Inicia monitoramento de **todas** as sessões de um filme de uma vez |
| `monitor stop <session_id>` | Para o monitoramento de uma sessão |
| `monitor status` | Lista monitores ativos |
| `monitor check` | Executa verificação única (para agendador) |
| `monitor history <session_id>` | Mostra o histórico de preços de uma sessão monitorada |

### Resumo diário

| Comando | Descrição |
|---------|-----------|
| `digest run` | Envia no Telegram o resumo dos filmes em cartaz hoje, com botão de compra por sessão (veja [Resumo diário](#resumo-diário)) |

### Configuração

| Comando | Descrição |
|---------|-----------|
| `config init` | Gera `config.json` padrão |

### Notificação

| Comando | Descrição |
|---------|-----------|
| `notify test` | Envia mensagem de teste no Telegram |

### Utilitários

| Comando | Descrição |
|---------|-----------|
| `sync` | Atualiza dados de filmes e cinemas da API (equivalente a `cinemas sync` + `movies list --refresh`) |

---

## Monitoramento contínuo

### Como funciona

O Movie Watcher não fica rodando em segundo plano. Em vez disso, ele é executado **sob demanda** via agendador de tarefas.

O fluxo é:

```
Você → monitor start <id>  (ou monitor start-movie "<filme>")
          ↓
     (salva no banco de dados)
          ↓
Agendador (a cada 5 min) → monitor check
                            ↓
                        Consulta API do Ingresso
                            ↓
                     Compara com último estado
                            ↓
                     Se mudou → notifica no Telegram
```

`monitor check` é seguro para rodar em paralelo: se uma execução anterior
ainda estiver em andamento (trava por até 10 minutos), a nova chamada é
recusada em vez de gerar notificações duplicadas ou dados inconsistentes.

### Monitorar todas as sessões de um filme de uma vez

Em vez de anotar e monitorar sessão por sessão:

```bash
movie-watcher.exe monitor start-movie "Mestres do Universo"
```

Isso carrega as sessões do filme em todos os cinemas monitorados (ou no
cinema padrão, se nenhum estiver marcado) e inicia o monitoramento de cada
uma, enviando uma única notificação resumo no Telegram.

### No Windows (Task Scheduler)

1. Abra o **Agendador de Tarefas** (taskschd.msc)
2. Clique em **Criar Tarefa Básica**
3. **Nome**: `Movie Watcher Check`
4. **Gatilho**: `Diariamente`, repetir a cada `5 minutos`
5. **Ação**: `Iniciar um programa`
   - **Programa/script**: caminho completo do `movie-watcher.exe`
   - **Argumentos**: `monitor check`
   - **Iniciar em**: pasta onde está o `config.json`
6. Finalizar

### No Linux (cron)

```bash
# Editar crontab
crontab -e

# Adicionar linha (executa a cada 5 minutos)
*/5 * * * * /caminho/para/movie-watcher monitor check
```

---

## Resumo diário

Diferente do `monitor check` (que só avisa sobre sessões que você registrou
manualmente), o `digest run` é proativo: uma vez por dia, monta e envia
sozinho um resumo de tudo que está em cartaz **hoje** nos cinemas
monitorados — agrupado por filme, com um botão por horário levando direto
pra página de compra da sessão no Ingresso.com.

```bash
movie-watcher.exe digest run
```

Cada sessão sem seleção de assento cai direto na finalização da compra; a
única etapa manual que sobra pra você é pagar.

### Promoção

Sessões marcam "🔻 promoção" quando o preço caiu desde a última vez que a
mesma sessão foi vista pelo próprio `digest run`. Não existe um campo de
desconto nativo na API do Ingresso.com — a promoção é sempre inferida por
queda de preço. Para funcionar de verdade (ter algo pra comparar), o
`digest run` registra o preço de **todas as sessões futuras** dos cinemas
monitorados a cada execução, não só as de hoje — assim, quando uma sessão
"vira hoje", ela já tem histórico dos dias anteriores pra comparar. Preços
sem histórico anterior nunca aparecem como promoção (não tem com o que
comparar ainda).

### Diferença para `monitor check`

| | `monitor check` | `digest run` |
|---|---|---|
| Quando notifica | Só sessões registradas com `monitor start`/`start-movie` | Tudo que está em cartaz hoje, automaticamente |
| Frequência recomendada | A cada 5 min | 1x por dia |
| Precisa de ação manual antes | Sim (`monitor start`) | Não |
| Lock de execução | `<database>.lock` | `<database>.digest.lock` (arquivo separado — os dois nunca disputam) |

Os dois continuam sendo comandos **stateless disparados por agendador**;
nenhum dos dois exige o app rodando continuamente (os botões de compra são
de URL — o Telegram abre o link direto, sem o bot precisar reagir a nada).

---

## Histórico de preços

Toda vez que uma sessão é monitorada (`monitor start` / `start-movie`) ou
verificada (`monitor check`), o preço e a disponibilidade no momento ficam
registrados. Para consultar essa série histórica:

```bash
movie-watcher.exe monitor history 4
```

Saída esperada:
```
Histórico de preços — Mestres Do Universo — Sala 1 18:30

  DATA/HORA           PREÇO      STATUS        VARIAÇÃO
  08/06 10:00:00       R$ 34.20   Disponível    —
  08/06 15:00:05       R$ 31.90   Disponível    ▼ R$ 2.30
  08/06 18:00:02       R$ 31.90   Disponível    =
```

Útil para decidir se vale a pena esperar o preço cair antes de comprar.

---

## Deploy em produção

O Movie Watcher é um CLI stateless — não é um servidor web, não escuta
porta nenhuma. Ele é feito pra ser disparado por um agendador (cron/Task
Scheduler localmente, ou **Cron Schedule** de uma plataforma como a
Railway), rodar até o fim e sair. Isso vale tanto pro `monitor check`
quanto pro `digest run` — inclusive o botão de compra do resumo diário é um
botão de **URL** do Telegram, que não chama o bot de volta, então nenhum
dos dois comandos precisa de um serviço sempre ligado.

Não precisa de `docker-compose`: o `Dockerfile` já existente builda a
imagem sozinho: a plataforma de deploy escolhida é quem orquestra quando
rodar (`docker-compose` serve pra orquestrar vários containers na sua
própria infra — não é o caso aqui).

### Passo a passo (Railway, mas o raciocínio vale pra qualquer PaaS com Cron Schedule)

1. **Conectar o repositório**: *New Project* → *Deploy from GitHub repo* →
   selecione este repositório. A plataforma detecta o `Dockerfile` e builda
   automaticamente.

2. **Adicionar um Volume** (crítico): monte um volume persistente, ex. em
   `/data`. É nele que o `movie-watcher.db` precisa morar — sem isso, o
   disco reseta a cada execução agendada e o app nunca detecta mudança de
   preço nem monta histórico de promoção nenhum.

3. **Variáveis de ambiente** (ver [Alternativa mais segura para o
   token](#configuração-do-telegram) acima):
   ```
   MW_TELEGRAM_BOT_TOKEN=...
   MW_TELEGRAM_CHAT_ID=...
   MW_DATABASE_PATH=/data/movie-watcher.db
   ```

4. **Dois Cron Schedules**, apontando pra mesma imagem/volume:
   - Um a cada 5 min, *Custom Start Command* = `monitor check`.
   - Outro 1x por dia (horário de sua preferência), *Custom Start Command*
     = `digest run`.

5. **Configuração inicial**: `digest run` funciona sozinho assim que
   houver sessões em cartaz — nada extra a fazer. Já `monitor check`
   depende de sessões registradas com `monitor start`/`start-movie`; use o
   *Shell* da plataforma (ou `railway ssh`) pra rodar esse comando uma vez
   contra o mesmo volume, se quiser também usar o monitoramento silencioso
   além do resumo diário.

### O que NÃO vai acontecer sozinho

Rodar isso agendado **não** enche seu Telegram de mensagens soltas: o
`digest run` manda uma mensagem por filme, 1x por dia, e o `monitor check`
só fala alguma coisa quando uma sessão registrada muda de preço/status. A
lista completa de filmes só sai pro Telegram se você pedir explicitamente
com `movies list --notify`/`sessions list --notify`.

---

## FAQ / Solução de problemas

### "Não recebo notificação no Telegram"

Verifique nesta ordem:

1. **Testar configuração**:
   ```bash
   movie-watcher.exe notify test
   ```
   Se receber "notifier desabilitado", o problema está no config.json.

2. **Verificar config.json**:
   ```bash
   type config.json
   ```
   Confirme que `bot_token` e `chat_id` estão preenchidos corretamente.

3. **Mensagem não chega?** O bot precisa ter recebido pelo menos uma mensagem sua primeiro (envie `/start` para o bot).

4. **Token inválido?** No BotFather, use `/mybots` → selecione seu bot → `API Token` para ver o token correto.

### "Nenhum filme encontrado"

Execute com `--refresh` para forçar busca na API:
```bash
movie-watcher.exe movies list -r
```

### "Sessão não encontrada"

As sessões expiram após o horário da sessão. Execute `sessions list` novamente para recarregar:
```bash
movie-watcher.exe sessions list "Nome do Filme"
```

Se isso acontecer logo depois de iniciar um monitoramento, não deveria mais
ocorrer: o ID de uma sessão monitorada agora é preservado entre atualizações
(veja [detalhes técnicos](#detalhes-técnicos-e-mudanças-recentes)). O erro só
aparece de fato quando a sessão saiu de cartaz.

### "já existe uma verificação em andamento"

O `monitor check` está protegido contra execuções simultâneas. Se você vir
essa mensagem, é porque outra chamada de `monitor check` ainda está em
andamento (ou travou há menos de 10 minutos). Normalmente basta aguardar a
próxima janela do agendador.

### Quais mudanças são detectadas?

- **Preço** alterado (ex: R$ 34,20 → R$ 31,90)
- **Disponibilidade** alterada (sessão ativada/desativada)

---

## Detalhes técnicos e mudanças recentes

Resumo do que foi corrigido/adicionado na última revisão do projeto, para
quem já conhecia versões anteriores:

- **Correção: `sessions list` não apaga mais sessões de outros cinemas.**
  Antes, toda atualização de sessões apagava a tabela inteira antes de
  recarregar; agora só são removidas as sessões do cinema que está sendo
  atualizado e que realmente saíram do cartaz.
- **Correção: o ID de uma sessão monitorada agora é estável.** Antes, cada
  `sessions list` recriava todas as linhas do zero (mesmo as que não tinham
  mudado), o que trocava o ID interno e quebrava monitoramentos em
  andamento — essa era a causa mais provável do erro "Sessão não
  encontrada". Agora sessões já existentes são atualizadas no lugar.
- **Correção: erros de banco não são mais ignorados silenciosamente** em
  `monitor check` (ex: falha ao desativar um monitor órfão).
- **Correção: `monitor check` não roda mais em paralelo consigo mesmo.** Um
  lock em arquivo (`<database>.lock`, válido por até 10 min) evita
  notificações duplicadas se o agendador disparar uma nova checagem antes da
  anterior terminar.
- **Qualidade**: mensagens de erro/diagnóstico agora vão para *stderr* (e
  não se misturam mais com a saída "de verdade" no *stdout*, o que facilita
  redirecionar/logar); renderização de tabelas e rótulos repetidos
  (status, classificação etária) foram unificados em um único helper.
- **Testes**: novos testes cobrem o comportamento de `LoadSessions` (não
  apagar sessões de outro cinema, preservar ID em refresh, remover apenas
  sessões que saíram do cartaz), o lock de `monitor check` e o registro de
  histórico de preços.
- **Nova funcionalidade**: `monitor start-movie` monitora todas as sessões
  de um filme de uma vez.
- **Nova funcionalidade**: suporte a múltiplos cinemas (`cinemas sync/list/
  search/add/remove`), com `sessions list`, `monitor start-movie` e
  `monitor check` operando sobre todos os cinemas marcados.
- **Nova funcionalidade**: histórico de preços por sessão
  (`monitor history <id>`).
- **Nova funcionalidade**: resumo diário proativo (`digest run`) com
  detecção de queda de preço e botão de compra direto — veja [Resumo
  diário](#resumo-diário).
- **Correção: variáveis de ambiente (`MW_...`) agora funcionam mesmo sem
  `config.json` no diretório.** Antes, a ausência do arquivo fazia o app
  cair num fallback que ignorava totalmente as variáveis de ambiente —
  quebrava exatamente o cenário de configurar tudo por env var num
  container/deploy.
- **Correção: `sessions list` (e `digest run`) não dependem mais de alguém
  ter rodado `movies list` antes.** `LoadSessions` só mantém uma sessão se
  já existir o filme correspondente localmente; agora os filmes são
  carregados primeiro, então a primeira execução num banco novo já traz
  sessões, sem precisar de uma chamada "de aquecimento" manual.
- **Correção: log de "record not found" do GORM não polui mais a saída.**
  Esperado com frequência pelo novo registro de histórico de preço (toda
  sessão vista pela primeira vez cai nesse caso); o logger do banco agora
  ignora esse erro específico e manda avisos reais para *stderr*.
- **Refatoração**: o lock de arquivo (antes só em `monitor check`) foi
  promovido para `internal/lock`, reaproveitado também pelo `digest run`
  (com arquivo de lock próprio, para os dois nunca disputarem).

---

## 🛠 Tecnologias

- **Go** 1.25 — Linguagem
- **Cobra** — CLI
- **Viper** — Configuração (JSON)
- **GORM** + **SQLite** (pure Go) — Banco de dados
- **go-telegram-bot-api** — Notificações Telegram
- **Ingresso.com API** — Dados de filmes e sessões

---

##  💻 Desenvolvido por Lázaro Vasconcelos
