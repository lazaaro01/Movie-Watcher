package cmd

import (
	"fmt"
	"os"

	"github.com/opencode/movie-watcher/internal/digest"
	"github.com/spf13/cobra"
)

var digestCmd = &cobra.Command{
	Use:   "digest",
	Short: "Comandos do resumo diário",
}

var digestRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Envia o resumo diário de filmes em cartaz para o Telegram",
	Long: `Monta e envia um resumo dos filmes em cartaz hoje nos cinemas
monitorados: horários, preço e um botão de compra por sessão. Sessões cujo
preço caiu desde a última vez que foram vistas são marcadas como promoção.

Pensado para rodar 1x por dia via cron/agendador, separado do
'monitor check'.`,
	Run: func(cmd *cobra.Command, args []string) {
		runner := digest.NewRunner()
		if err := runner.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(digestCmd)
	digestCmd.AddCommand(digestRunCmd)
}
