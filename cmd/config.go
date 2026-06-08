package cmd

import (
	"fmt"
	"os"

	"github.com/opencode/movie-watcher/configs"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Comandos de configuração",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Gera arquivo config.json padrão",
	Run: func(cmd *cobra.Command, args []string) {
		if err := configs.WriteDefaultConfig(); err != nil {
			fmt.Printf("Erro ao gerar config.json: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("config.json criado com sucesso!")
		fmt.Println("Edite o arquivo e configure seu token do Telegram.")
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
}
