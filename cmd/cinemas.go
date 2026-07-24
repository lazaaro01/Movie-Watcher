package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/opencode/movie-watcher/internal/cinema"
	"github.com/opencode/movie-watcher/internal/models"
	"github.com/spf13/cobra"
)

var cinemasCmd = &cobra.Command{
	Use:   "cinemas",
	Short: "Comandos para gerenciar os cinemas monitorados",
	Long: `Por padrão, o Movie Watcher opera sobre o cinema definido em
"theater_id" no config.json. Esses comandos permitem descobrir outros
cinemas da mesma cidade e marcar vários deles para serem monitorados ao
mesmo tempo (movies/sessions/monitor passam a considerar todos).`,
}

var cinemasSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Busca a lista de cinemas disponíveis na cidade configurada",
	Run: func(cmd *cobra.Command, args []string) {
		service := cinema.NewService()
		if err := service.UpdateCinemas(); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao sincronizar cinemas: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Cinemas sincronizados com sucesso! Use 'cinemas list' para ver.")
	},
}

func renderCinemas(cinemas []models.Cinema) {
	table := newTable([]string{"ID", "Nome", "Bairro", "Monitorado"})

	for _, c := range cinemas {
		monitored := "Não"
		if c.Monitored {
			monitored = "Sim"
		}
		table.Append([]string{
			strconv.Itoa(int(c.ID)),
			c.Name,
			c.Neighborhood,
			monitored,
		})
	}

	table.Render()
}

var cinemasListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista cinemas conhecidos e quais estão sendo monitorados",
	Run: func(cmd *cobra.Command, args []string) {
		service := cinema.NewService()
		cinemas, err := service.GetCinemas()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

		if len(cinemas) == 0 {
			fmt.Println("Nenhum cinema encontrado. Execute 'cinemas sync' primeiro.")
			return
		}

		renderCinemas(cinemas)
	},
}

var cinemasSearchCmd = &cobra.Command{
	Use:   "search [nome]",
	Short: "Busca cinema por nome",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		service := cinema.NewService()
		cinemas, err := service.SearchCinemas(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

		if len(cinemas) == 0 {
			fmt.Println("Nenhum cinema encontrado.")
			return
		}

		renderCinemas(cinemas)
	},
}

var cinemasAddCmd = &cobra.Command{
	Use:   "add [id]",
	Short: "Marca um cinema para ser monitorado",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ID inválido: %v\n", err)
			os.Exit(1)
		}

		service := cinema.NewService()
		if err := service.SetCinemaMonitored(uint(id), true); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Cinema adicionado ao monitoramento.")
	},
}

var cinemasRemoveCmd = &cobra.Command{
	Use:   "remove [id]",
	Short: "Remove um cinema do monitoramento",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ID inválido: %v\n", err)
			os.Exit(1)
		}

		service := cinema.NewService()
		if err := service.SetCinemaMonitored(uint(id), false); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Cinema removido do monitoramento.")
	},
}

func init() {
	rootCmd.AddCommand(cinemasCmd)
	cinemasCmd.AddCommand(cinemasSyncCmd)
	cinemasCmd.AddCommand(cinemasListCmd)
	cinemasCmd.AddCommand(cinemasSearchCmd)
	cinemasCmd.AddCommand(cinemasAddCmd)
	cinemasCmd.AddCommand(cinemasRemoveCmd)
}
