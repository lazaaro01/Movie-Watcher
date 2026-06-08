package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "movie-watcher",
	Short: "Movie Watcher CLI - Monitore sessões de cinema",
	Long: `Movie Watcher CLI é uma ferramenta para monitorar filmes,
sessões e preços no Cinépolis RioMar Kennedy (Fortaleza),
com notificações via Telegram.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
}
