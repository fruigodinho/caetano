// Command decrypt desencripta um backup .sql.gz.enc gerado pelo BackupService,
// usando a encryption.key configurada.
package main

import (
	"flag"
	"log"
	"os"

	"github.com/fruigodinho/caetano/core/config"
	"github.com/fruigodinho/caetano/core/service"
)

func main() {
	configPath := flag.String("config", "", "caminho para o ficheiro de configuração YAML (default: procura caetano.dev.yaml a subir a árvore)")
	inFile := flag.String("file", "", "ficheiro de backup a desencriptar (.sql.gz.enc)")
	outFile := flag.String("outfile", "", "ficheiro de saída (.sql.gz)")
	flag.Parse()

	if *inFile == "" || *outFile == "" {
		log.Fatal("uso: decrypt -file backup.sql.gz.enc -outfile backup.sql.gz [-config caminho]")
	}

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

	crypto, err := service.NewCryptoService(cfg.Encryption.Key)
	if err != nil {
		log.Fatalf("erro ao inicializar encriptação: %v", err)
	}

	raw, err := os.ReadFile(*inFile)
	if err != nil {
		log.Fatalf("erro ao ler %s: %v", *inFile, err)
	}

	plaintext, err := crypto.Decrypt(raw)
	if err != nil {
		log.Fatalf("erro ao desencriptar %s: %v", *inFile, err)
	}

	if err := os.WriteFile(*outFile, plaintext, 0o600); err != nil {
		log.Fatalf("erro ao gravar %s: %v", *outFile, err)
	}

	log.Printf("desencriptado: %s -> %s", *inFile, *outFile)
}
