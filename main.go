package main

import (
	"fmt"
	"os"

	"github.com/opencode/movie-watcher/cmd"
	"github.com/opencode/movie-watcher/configs"
	"github.com/opencode/movie-watcher/database"
)

func main() {
	if err := configs.Init(); err != nil {
		fmt.Printf("Erro ao carregar config: %v\n", err)
		os.Exit(1)
	}

	if err := database.Init(configs.AppConfig.Database.Path); err != nil {
		fmt.Printf("Erro ao iniciar banco de dados: %v\n", err)
		os.Exit(1)
	}

	cmd.Execute()
}
