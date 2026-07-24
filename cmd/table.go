package cmd

import (
	"os"

	"github.com/olekukonko/tablewriter"
)

func newTable(headers []string) *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.SetBorder(false)
	table.SetColumnSeparator("  ")
	return table
}

func ratingOrDefault(rating string) string {
	if rating == "" {
		return "Livre"
	}
	return rating
}

func statusLabel(enabled bool) string {
	if enabled {
		return "Disponível"
	}
	return "Indisponível"
}
