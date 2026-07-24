package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/opencode/movie-watcher/internal/cinema"
	"github.com/opencode/movie-watcher/internal/monitor"
	"github.com/spf13/cobra"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Comandos de monitoramento de sessões",
}

var monitorStartCmd = &cobra.Command{
	Use:   "start [session_id]",
	Short: "Inicia monitoramento de uma sessão",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ID inválido: %v\n", err)
			os.Exit(1)
		}

		runner := monitor.NewRunner()
		if err := runner.Start(uint(id)); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}
	},
}

var monitorStartMovieCmd = &cobra.Command{
	Use:   "start-movie [nome]",
	Short: "Inicia monitoramento de todas as sessões de um filme",
	Long: `Carrega as sessões do filme em todos os cinemas monitorados (ou no
cinema padrão do config.json, se nenhum estiver marcado) e inicia o
monitoramento de cada uma delas de uma só vez.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runner := monitor.NewRunner()
		if err := runner.StartMovie(args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}
	},
}

var monitorStopCmd = &cobra.Command{
	Use:   "stop [session_id]",
	Short: "Para monitoramento de uma sessão",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ID inválido: %v\n", err)
			os.Exit(1)
		}

		runner := monitor.NewRunner()
		if err := runner.Stop(uint(id)); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}
	},
}

var monitorStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Mostra monitores ativos",
	Run: func(cmd *cobra.Command, args []string) {
		runner := monitor.NewRunner()
		if err := runner.Status(); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}
	},
}

var monitorCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Executa verificação única de todos os monitores",
	Long: `Executa uma verificação única em todos os monitores ativos.
Use com cron/task scheduler para monitoramento periódico.`,
	Run: func(cmd *cobra.Command, args []string) {
		runner := monitor.NewRunner()
		if err := runner.CheckAll(); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}
	},
}

var monitorHistoryCmd = &cobra.Command{
	Use:   "history [session_id]",
	Short: "Mostra o histórico de preços de uma sessão monitorada",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ID inválido: %v\n", err)
			os.Exit(1)
		}

		service := cinema.NewService()

		session, err := service.GetSessionByID(uint(id))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro: sessão não encontrada: %v\n", err)
			os.Exit(1)
		}

		history, err := service.GetPriceHistory(uint(id))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

		if len(history) == 0 {
			fmt.Println("Nenhum histórico encontrado. Use 'monitor start' para começar a monitorar.")
			return
		}

		fmt.Printf("Histórico de preços — %s — %s %s\n\n", session.MovieTitle, session.Room, session.DateTime.Format("02/01 15:04"))

		table := newTable([]string{"Data/Hora", "Preço", "Status", "Variação"})

		var prevPrice float64
		for i, h := range history {
			variation := "—"
			if i > 0 {
				diff := h.Price - prevPrice
				switch {
				case diff > 0:
					variation = fmt.Sprintf("▲ R$ %.2f", diff)
				case diff < 0:
					variation = fmt.Sprintf("▼ R$ %.2f", -diff)
				default:
					variation = "="
				}
			}
			table.Append([]string{
				h.CreatedAt.Format("02/01 15:04:05"),
				fmt.Sprintf("R$ %.2f", h.Price),
				statusLabel(h.Enabled),
				variation,
			})
			prevPrice = h.Price
		}

		table.Render()
	},
}

func init() {
	rootCmd.AddCommand(monitorCmd)
	monitorCmd.AddCommand(monitorStartCmd)
	monitorCmd.AddCommand(monitorStartMovieCmd)
	monitorCmd.AddCommand(monitorStopCmd)
	monitorCmd.AddCommand(monitorStatusCmd)
	monitorCmd.AddCommand(monitorCheckCmd)
	monitorCmd.AddCommand(monitorHistoryCmd)
}
