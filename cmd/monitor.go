package cmd

import (
	"fmt"
	"os"
	"strconv"

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
			fmt.Printf("ID inválido: %v\n", err)
			os.Exit(1)
		}

		runner := monitor.NewRunner()
		if err := runner.Start(uint(id)); err != nil {
			fmt.Printf("Erro: %v\n", err)
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
			fmt.Printf("ID inválido: %v\n", err)
			os.Exit(1)
		}

		runner := monitor.NewRunner()
		if err := runner.Stop(uint(id)); err != nil {
			fmt.Printf("Erro: %v\n", err)
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
			fmt.Printf("Erro: %v\n", err)
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
			fmt.Printf("Erro: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(monitorCmd)
	monitorCmd.AddCommand(monitorStartCmd)
	monitorCmd.AddCommand(monitorStopCmd)
	monitorCmd.AddCommand(monitorStatusCmd)
	monitorCmd.AddCommand(monitorCheckCmd)
}
