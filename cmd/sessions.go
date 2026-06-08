package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/opencode/movie-watcher/configs"
	"github.com/opencode/movie-watcher/internal/cinema"
	"github.com/opencode/movie-watcher/internal/notifier"
	"github.com/spf13/cobra"
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "Comandos para sessões",
}

var sessionsNotify bool

var sessionsListCmd = &cobra.Command{
	Use:   "list [movie]",
	Short: "Lista sessões de um filme",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		service := cinema.NewService()
		cfg := configs.AppConfig

		_, err := service.LoadSessions(cfg.TheaterID)
		if err != nil {
			fmt.Printf("Erro ao carregar sessões: %v\n", err)
		}

		sessions, err := service.GetSessionsByMovie(args[0])
		if err != nil {
			fmt.Printf("Erro: %v\n", err)
			os.Exit(1)
		}

		if len(sessions) == 0 {
			fmt.Println("Nenhuma sessão encontrada.")
			return
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "Filme", "Sala", "Horário", "Preço", "Status"})
		table.SetBorder(false)
		table.SetColumnSeparator("  ")

		for _, s := range sessions {
			status := "Disponível"
			if !s.Enabled {
				status = "Indisponível"
			}
			table.Append([]string{
				strconv.Itoa(int(s.ID)),
				s.MovieTitle,
				s.Room,
				s.DateTime.Format("02/01 15:04"),
				fmt.Sprintf("R$ %.2f", s.Price),
				status,
			})
		}

		table.Render()

		if sessionsNotify {
			n := notifier.New()
			msg := fmt.Sprintf("*🎬 Sessões de \"%s\" — RioMar Kennedy*\n\n", sessions[0].MovieTitle)
			for _, s := range sessions {
				status := "Disponível"
				if !s.Enabled {
					status = "Indisponível"
				}
				msg += fmt.Sprintf("🪑 *%s* — %s — R$ %.2f — %s\n", s.Room, s.DateTime.Format("15:04"), s.Price, status)
			}
			err := n.SendMessage(msg)
			if err != nil {
				fmt.Printf("Erro ao enviar para Telegram: %v\n", err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(sessionsCmd)
	sessionsCmd.AddCommand(sessionsListCmd)

	sessionsListCmd.Flags().BoolVarP(&sessionsNotify, "notify", "n", false, "Envia resultado para o Telegram")
}
