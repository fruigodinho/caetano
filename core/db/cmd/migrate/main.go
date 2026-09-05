// Command migrate aplica ou reverte as migrações do schema caetano.
package main

import (
	"flag"
	"log"

	"github.com/fruigodinho/caetano/core/config"
	"github.com/fruigodinho/caetano/core/db"
)

func main() {
	configPath := flag.String("config", "", "caminho para o ficheiro de configuração YAML (default: procura caetano.dev.yaml a subir a árvore)")
	direction := flag.String("dir", "up", "direção da migração: up|down")
	flag.Parse()

	path := *configPath
	if path == "" {
		found, err := config.FindConfigPath("caetano.dev.yaml")
		if err != nil {
			log.Fatalf("erro ao localizar configuração: %v (use -config)", err)
		}
		path = found
	}

	cfg, err := config.Load(path)
	if err != nil {
		log.Fatalf("erro ao carregar configuração %s: %v", path, err)
	}

	if err := db.Migrate(cfg.DB.DSN, *direction); err != nil {
		log.Fatalf("erro ao aplicar migrações: %v", err)
	}

	log.Printf("migrações (%s) aplicadas com sucesso", *direction)
}
