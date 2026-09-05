package main

import (
	"fmt"
	"log"

	"github.com/xuri/excelize/v2"
)

func main() {
	f, err := excelize.OpenFile("data/saldos-esperados.xlsx")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	fmt.Println("Folhas disponíveis:")
	for i, sheet := range f.GetSheetList() {
		fmt.Printf("%d: %s\n", i, sheet)
	}

	// Try to find LEGENDA sheet
	legendSheet := ""
	for _, sheet := range f.GetSheetList() {
		if sheet == "LEGENDA" || sheet == "Legenda" {
			legendSheet = sheet
			break
		}
	}

	if legendSheet == "" {
		fmt.Println("\nFolha LEGENDA não encontrada")
		return
	}

	fmt.Printf("\n=== Conteúdo da folha %s ===\n", legendSheet)
	rows, err := f.GetRows(legendSheet)
	if err != nil {
		log.Fatal(err)
	}

	for i, row := range rows {
		if i > 50 { // Limitar output
			break
		}
		fmt.Printf("Linha %d: %v\n", i+1, row)
	}
}
