# 🎬 Movie Watcher CLI

Monitore sessões de cinema do **Cinépolis RioMar Kennedy (Fortaleza)** e receba alertas no **Telegram** quando houver mudanças de preço ou disponibilidade.

---

## 📋 Sumário

- [Pré-requisitos](#pré-requisitos)
- [Instalação](#instalação)
- [Configuração do Telegram](#configuração-do-telegram)
- [Primeiros passos](#primeiros-passos)
- [Comandos](#comandos)
- [Monitoramento contínuo](#monitoramento-contínuo)
- [FAQ / Solução de problemas](#faq--solução-de-problemas)

---

## Pré-requisitos

- **Go** 1.21+ instalado ([download](https://go.dev/dl/))
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

---

## Comandos

### Filmes

| Comando | Descrição |
|---------|-----------|
| `movies list` | Lista filmes em cartaz (cache local) |
| `movies list --refresh` | Lista filmes forçando busca na API |
| `movies search <nome>` | Busca filme por nome |

### Sessões

| Comando | Descrição |
|---------|-----------|
| `sessions list <filme>` | Lista sessões de um filme no RioMar Kennedy |

### Monitoramento

| Comando | Descrição |
|---------|-----------|
| `monitor start <id>` | Inicia monitoramento de uma sessão |
| `monitor stop <id>` | Para o monitoramento |
| `monitor status` | Lista monitores ativos |
| `monitor check` | Executa verificação única (para agendador) |

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
| `sync` | Atualiza dados de filmes e cinemas da API |

---

## Monitoramento contínuo

### Como funciona

O Movie Watcher não fica rodando em segundo plano. Em vez disso, ele é executado **sob demanda** via agendador de tarefas.

O fluxo é:

```
Você → monitor start <id>
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

### Quais mudanças são detectadas?

- **Preço** alterado (ex: R$ 34,20 → R$ 31,90)
- **Disponibilidade** alterada (sessão ativada/desativada)

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