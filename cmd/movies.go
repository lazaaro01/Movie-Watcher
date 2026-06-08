package cmd

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/opencode/movie-watcher/internal/cinema"
	"github.com/opencode/movie-watcher/internal/models"
	"github.com/opencode/movie-watcher/internal/notifier"
	"github.com/spf13/cobra"
)

var moviesCmd = &cobra.Command{
	Use:   "movies",
	Short: "Comandos para listar filmes",
}

var (
	moviesRefresh bool
	moviesNotify  bool
)

var moviesListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista filmes em cartaz",
	Run: func(cmd *cobra.Command, args []string) {
		service := cinema.NewService()

		var movies []models.Movie
		var err error
		if moviesRefresh {
			movies, err = service.LoadMovies()
		} else {
			movies, err = service.ListMovies()
		}
		if err != nil {
			fmt.Printf("Erro: %v\n", err)
			os.Exit(1)
		}

		if len(movies) == 0 {
			fmt.Println("Nenhum filme encontrado.")
			return
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "Título", "Gênero", "Duração", "Classificação"})
		table.SetBorder(false)
		table.SetColumnSeparator("  ")

		for _, m := range movies {
			rating := m.ContentRating
			if rating == "" {
				rating = "Livre"
			}
			table.Append([]string{
				fmt.Sprintf("%d", m.ID),
				m.Title,
				m.Genre,
				fmt.Sprintf("%d min", m.Duration),
				rating,
			})
		}

		table.Render()

		if moviesNotify {
			n := notifier.New()
			msg := "*🎬 Filmes em cartaz — RioMar Kennedy*\n\n"
			for _, m := range movies {
				rating := m.ContentRating
				if rating == "" {
					rating = "Livre"
				}
				msg += fmt.Sprintf("• *%s*\n  %s · %d min · %s\n", m.Title, m.Genre, m.Duration, rating)
			}
			err := n.SendMessage(msg)
			if err != nil {
				fmt.Printf("Erro ao enviar para Telegram: %v\n", err)
			}
		}
	},
}

var moviesSearchCmd = &cobra.Command{
	Use:   "search [nome]",
	Short: "Busca filme por nome",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		service := cinema.NewService()
		movies, err := service.SearchMovies(args[0])
		if err != nil {
			fmt.Printf("Erro: %v\n", err)
			os.Exit(1)
		}

		if len(movies) == 0 {
			fmt.Println("Nenhum filme encontrado.")
			return
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "Título", "Gênero", "Duração", "Classificação"})
		table.SetBorder(false)
		table.SetColumnSeparator("  ")

		for _, m := range movies {
			rating := m.ContentRating
			if rating == "" {
				rating = "Livre"
			}
			table.Append([]string{
				fmt.Sprintf("%d", m.ID),
				m.Title,
				m.Genre,
				fmt.Sprintf("%d min", m.Duration),
				rating,
			})
		}

		table.Render()
	},
}

func init() {
	rootCmd.AddCommand(moviesCmd)
	moviesCmd.AddCommand(moviesListCmd)
	moviesCmd.AddCommand(moviesSearchCmd)

	moviesListCmd.Flags().BoolVarP(&moviesRefresh, "refresh", "r", false, "Força atualização dos dados da API")
	moviesListCmd.Flags().BoolVarP(&moviesNotify, "notify", "n", false, "Envia resultado para o Telegram")
}
