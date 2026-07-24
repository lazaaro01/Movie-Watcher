package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/opencode/movie-watcher/internal/cinema"
	"github.com/opencode/movie-watcher/internal/models"
	"github.com/opencode/movie-watcher/internal/notifier"
	"github.com/spf13/cobra"
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "Comandos para sessões",
}

var (
	sessionsNotify   bool
	sessionsCinemaID uint
)

func renderSessions(sessions []models.Session, cinemaNames map[int]string) {
	showCinemaColumn := len(cinemaNames) > 1

	headers := []string{"ID", "Filme", "Sala", "Horário", "Preço", "Status"}
	if showCinemaColumn {
		headers = []string{"ID", "Filme", "Cinema", "Sala", "Horário", "Preço", "Status"}
	}
	table := newTable(headers)

	for _, s := range sessions {
		row := []string{strconv.Itoa(int(s.ID)), s.MovieTitle}
		if showCinemaColumn {
			name := cinemaNames[int(s.TheaterID)]
			if name == "" {
				name = fmt.Sprintf("Cinema #%d", s.TheaterID)
			}
			row = append(row, name)
		}
		row = append(row, s.Room, s.DateTime.Format("02/01 15:04"), fmt.Sprintf("R$ %.2f", s.Price), statusLabel(s.Enabled))
		table.Append(row)
	}

	table.Render()
}

var sessionsListCmd = &cobra.Command{
	Use:   "list [movie]",
	Short: "Lista sessões de um filme",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		service := cinema.NewService()

		var theaterIDs []int
		if sessionsCinemaID != 0 {
			theaterID, err := service.GetCinemaTheaterID(sessionsCinemaID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
				os.Exit(1)
			}
			theaterIDs = []int{theaterID}
		} else {
			ids, err := service.ResolveTheaterIDs()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao resolver cinemas: %v\n", err)
				os.Exit(1)
			}
			theaterIDs = ids
		}

		if _, err := service.LoadSessionsForTheaters(theaterIDs); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao carregar sessões: %v\n", err)
		}

		sessions, err := service.GetSessionsByMovie(args[0], theaterIDs...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

		if len(sessions) == 0 {
			fmt.Println("Nenhuma sessão encontrada.")
			return
		}

		cinemaNames := service.GetCinemaNamesByTheaterIDs(theaterIDs)
		renderSessions(sessions, cinemaNames)

		if sessionsNotify {
			n := notifier.New()
			msg := fmt.Sprintf("*🎬 Sessões de \"%s\"*\n\n", sessions[0].MovieTitle)
			for _, s := range sessions {
				msg += fmt.Sprintf("🪑 *%s* — %s — R$ %.2f — %s\n", s.Room, s.DateTime.Format("15:04"), s.Price, statusLabel(s.Enabled))
			}
			if err := n.SendMessage(msg); err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao enviar para Telegram: %v\n", err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(sessionsCmd)
	sessionsCmd.AddCommand(sessionsListCmd)

	sessionsListCmd.Flags().BoolVarP(&sessionsNotify, "notify", "n", false, "Envia resultado para o Telegram")
	sessionsListCmd.Flags().UintVarP(&sessionsCinemaID, "cinema", "c", 0, "Restringe a busca a um cinema específico (ID de 'cinemas list')")
}
