package notifier

import (
	"fmt"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/opencode/movie-watcher/configs"
)

type Notifier struct {
	bot     *tgbotapi.BotAPI
	chatID  int64
	enabled bool
}

func New() *Notifier {
	cfg := configs.AppConfig

	if cfg.Telegram.BotToken == "" {
		fmt.Fprintln(os.Stderr, "[notifier] Telegram não configurado: bot_token vazio em config.json")
		return &Notifier{enabled: false}
	}

	if cfg.Telegram.ChatID == 0 {
		fmt.Fprintln(os.Stderr, "[notifier] Telegram não configurado: chat_id = 0 em config.json")
		return &Notifier{enabled: false}
	}

	bot, err := tgbotapi.NewBotAPI(cfg.Telegram.BotToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[notifier] Erro ao conectar no Telegram: %v\n", err)
		return &Notifier{enabled: false}
	}

	return &Notifier{
		bot:     bot,
		chatID:  cfg.Telegram.ChatID,
		enabled: true,
	}
}

func (n *Notifier) Test() error {
	if !n.enabled {
		return fmt.Errorf("notifier desabilitado — verifique bot_token e chat_id no config.json")
	}

	msg := tgbotapi.NewMessage(n.chatID, "✅ *Movie Watcher CLI*\nTeste de notificação funcionando!")
	msg.ParseMode = "Markdown"

	_, err := n.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("erro ao enviar mensagem de teste: %w", err)
	}

	return nil
}

func (n *Notifier) SendMessage(text string) error {
	if !n.enabled {
		fmt.Println("[notifier] Telegram not configured. Message:")
		fmt.Println(text)
		return nil
	}

	msg := tgbotapi.NewMessage(n.chatID, text)
	msg.ParseMode = "Markdown"

	_, err := n.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("error sending telegram message: %w", err)
	}

	return nil
}

func (n *Notifier) IsEnabled() bool {
	return n.enabled
}

// Button is a Telegram inline keyboard button that links straight to a URL
// — Telegram opens the link client-side, it never calls back to the bot,
// so callers don't need any interactive/webhook infrastructure to use it.
type Button struct {
	Label string
	URL   string
}

// SendMessageWithButtons behaves like SendMessage but attaches one inline
// URL button per row. When the notifier is disabled, it prints the text
// and the button URLs to stdout, mirroring SendMessage's fallback.
func (n *Notifier) SendMessageWithButtons(text string, buttons []Button) error {
	if !n.enabled {
		fmt.Println("[notifier] Telegram not configured. Message:")
		fmt.Println(text)
		for _, b := range buttons {
			fmt.Printf("  [%s] %s\n", b.Label, b.URL)
		}
		return nil
	}

	msg := tgbotapi.NewMessage(n.chatID, text)
	msg.ParseMode = "Markdown"

	if len(buttons) > 0 {
		rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(buttons))
		for _, b := range buttons {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL(b.Label, b.URL),
			))
		}
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	}

	if _, err := n.bot.Send(msg); err != nil {
		return fmt.Errorf("error sending telegram message: %w", err)
	}

	return nil
}
