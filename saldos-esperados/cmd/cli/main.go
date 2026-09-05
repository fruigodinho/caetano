// Command saldos-cli agrupa tarefas de manutenção de dados de saldos-esperados
// (importação de regras, processamento de ficheiros) fora do servidor web.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	coreconfig "github.com/fruigodinho/caetano/core/config"
	coredb "github.com/fruigodinho/caetano/core/db"
	"github.com/fruigodinho/caetano/saldos-esperados/internal/adapter/parser"
	"github.com/fruigodinho/caetano/saldos-esperados/internal/core/domain"
	"github.com/fruigodinho/caetano/saldos-esperados/internal/service"
)

func main() {
	configPath := ""

	var rootCmd = &cobra.Command{
		Use:   "saldos-cli",
		Short: "CLI de manutenção de dados de saldos-esperados",
	}
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "caminho para o ficheiro de configuração YAML (default: procura caetano.dev.yaml a subir a árvore)")

	var importRulesCmd = &cobra.Command{
		Use:   "import-rules",
		Short: "Importa regras de saldo esperado a partir de um Excel",
		Run: func(cmd *cobra.Command, args []string) {
			file, _ := cmd.Flags().GetString("file")
			if file == "" {
				log.Fatal("é obrigatório indicar -f/--file")
			}

			conn := mustOpenDB(configPath)
			defer conn.Close()

			validator := service.NewValidator(conn)
			excelParser := parser.NewExcelParser()

			fmt.Println("A importar tipos de saldo da folha LEGENDA...")
			balanceTypes, err := excelParser.ParseBalanceTypes(file)
			if err != nil {
				log.Fatalf("erro ao interpretar tipos de saldo: %v", err)
			}
			if err := validator.ImportBalanceTypes(context.Background(), balanceTypes); err != nil {
				log.Fatalf("erro ao importar tipos de saldo: %v", err)
			}
			fmt.Printf("%d tipos de saldo importados.\n", len(balanceTypes))

			fmt.Println("A importar regras de saldo esperado...")
			rules, err := excelParser.ParseExpectedBalances(file)
			if err != nil {
				log.Fatalf("erro ao interpretar Excel: %v", err)
			}
			if err := validator.ImportRules(context.Background(), rules); err != nil {
				log.Fatalf("erro ao importar regras: %v", err)
			}
			fmt.Printf("%d regras importadas.\n", len(rules))
		},
	}
	importRulesCmd.Flags().StringP("file", "f", "", "caminho para o ficheiro Excel")

	var processFileCmd = &cobra.Command{
		Use:   "process-file",
		Short: "Processa um balancete CSV",
		Run: func(cmd *cobra.Command, args []string) {
			file, _ := cmd.Flags().GetString("file")
			if file == "" {
				log.Fatal("é obrigatório indicar -f/--file")
			}

			conn := mustOpenDB(configPath)
			defer conn.Close()

			validator := service.NewValidator(conn)
			csvParser := parser.NewCSVParser()

			label, err := csvParser.ExtractLabel(file)
			if err != nil {
				log.Printf("aviso: não foi possível extrair o rótulo: %v", err)
				label = ""
			}

			uploadID, err := validator.CreateUploadRecord(context.Background(), file, label)
			if err != nil {
				log.Fatalf("erro ao criar registo de upload: %v", err)
			}
			if label != "" {
				fmt.Printf("a processar ficheiro com rótulo: %s\n", label)
			}

			err = csvParser.ParseBalanceSheet(file, 100, func(batch []domain.AccountEntry) error {
				return validator.ProcessBatch(context.Background(), uploadID, batch)
			})
			if err != nil {
				log.Fatalf("erro ao processar CSV: %v", err)
			}
			fmt.Println("ficheiro processado com sucesso.")
		},
	}
	processFileCmd.Flags().StringP("file", "f", "", "caminho para o ficheiro CSV")

	rootCmd.AddCommand(importRulesCmd, processFileCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func mustOpenDB(configPath string) *sql.DB {
	path := configPath
	if path == "" {
		found, err := coreconfig.FindConfigPath("caetano.dev.yaml")
		if err != nil {
			log.Fatalf("erro ao localizar configuração: %v (use --config)", err)
		}
		path = found
	}

	cfg, err := coreconfig.Load(path)
	if err != nil {
		log.Fatalf("erro ao carregar configuração %s: %v", path, err)
	}

	conn, err := coredb.Open(context.Background(), cfg.DB.DSN)
	if err != nil {
		log.Fatalf("erro ao ligar à base de dados: %v", err)
	}
	return conn
}
