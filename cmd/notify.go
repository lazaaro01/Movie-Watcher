package cmd

import (
	"fmt"
	"os"

	"github.com/opencode/movie-watcher/internal/cinema"
	"github.com/opencode/movie-watcher/internal/notifier"
	"github.com/spf13/cobra"
)

var notifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "Comandos de notificação",
}

var notifyTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Envia uma mensagem de teste no Telegram",
	Run: func(cmd *cobra.Command, args []string) {
		n := notifier.New()
		if err := n.Test(); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Mensagem de teste enviada com sucesso!")
	},
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Atualiza dados de filmes e cinemas da API",
	Run: func(cmd *cobra.Command, args []string) {
		service := cinema.NewService()
		if err := service.RefreshDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao sincronizar: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Dados sincronizados com sucesso!")
	},
}

func init() {
	rootCmd.AddCommand(notifyCmd)
	notifyCmd.AddCommand(notifyTestCmd)
	rootCmd.AddCommand(syncCmd)
}
