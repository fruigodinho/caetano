// Package config carrega a configuração do caetano a partir de um ficheiro YAML por
// ambiente e determina se a aplicação está a correr em produção.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config é a configuração completa lida do YAML por ambiente.
type Config struct {
	App        AppConfig        `yaml:"app"`
	DB         DBConfig         `yaml:"db"`
	Cloudflare CloudflareConfig `yaml:"cloudflare"`
	Dropbox    DropboxConfig    `yaml:"dropbox"`
	Encryption EncryptionConfig `yaml:"encryption"`

	// DevAutoLoginEmail só é preenchido quando !IsProduction() — ver load().
	// Em produção fica sempre vazio, mesmo que o YAML tenha o campo definido.
	DevAutoLoginEmail string `yaml:"-"`
}

// AppConfig contém os parâmetros gerais de arranque do servidor HTTP.
//
// O modo Gin (debug/release) não é um campo de configuração: segue sempre
// IsProduction() (GIN_MODE/ENV do processo), para nunca divergir do critério que
// já decide o autologin de dev (RF-3). Os caminhos de xmldri e saldos-esperados
// são fixos (RF-8), por isso não há subpath configurável.
type AppConfig struct {
	Port string `yaml:"port"`
}

// DBConfig contém a connection string do Postgres.
type DBConfig struct {
	DSN string `yaml:"dsn"`
}

// CloudflareConfig contém os parâmetros de validação do Cloudflare Access.
type CloudflareConfig struct {
	TeamDomain string `yaml:"team_domain"`
	AUD        string `yaml:"aud"`
}

// DropboxConfig contém as credenciais da app Dropbox usada para os backups.
type DropboxConfig struct {
	AppKey       string `yaml:"app_key"`
	AppSecret    string `yaml:"app_secret"`
	RefreshToken string `yaml:"refresh_token"`
	Path         string `yaml:"path"`
}

// EncryptionConfig contém a chave AES-256-GCM usada para encriptar backups.
type EncryptionConfig struct {
	Key string `yaml:"key"`
}

// devFields é o subconjunto do YAML que só interessa fora de produção. Fica separado
// da struct Config para que, em produção, o valor nunca chegue a ser desserializado
// para a configuração usada pelo resto da aplicação (RF-3).
type devFields struct {
	Dev struct {
		AutoLoginEmail string `yaml:"auto_login_email"`
	} `yaml:"dev"`
}

// IsProduction indica se a aplicação está a correr em modo de produção.
//
// Fonte única de verdade: GIN_MODE=release ou ENV=production. Usado para decidir
// cookies Secure, HSTS, e sobretudo se o autologin de dev pode ou não ser lido.
func IsProduction() bool {
	return os.Getenv("GIN_MODE") == "release" || os.Getenv("ENV") == "production"
}

// Load lê e valida a configuração a partir do ficheiro YAML em path.
//
// Parameters:
//   - path: caminho para o ficheiro YAML (caetano.dev.yaml ou caetano.prod.yaml)
//
// Returns:
//   - *Config: configuração carregada
//   - error: erro de leitura, parsing, ou validação
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler configuração %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("erro ao interpretar configuração %s: %w", path, err)
	}

	applyDefaults(&cfg)

	// Estruturalmente impossível de ativar em produção: o campo só é lido se
	// !IsProduction(); em produção, cfg.DevAutoLoginEmail fica sempre "".
	if !IsProduction() {
		var dev devFields
		if err := yaml.Unmarshal(raw, &dev); err != nil {
			return nil, fmt.Errorf("erro ao interpretar configuração de dev %s: %w", path, err)
		}
		cfg.DevAutoLoginEmail = dev.Dev.AutoLoginEmail
	}

	if cfg.DB.DSN == "" {
		return nil, fmt.Errorf("configuração inválida: db.dsn em falta")
	}

	return &cfg, nil
}

// applyDefaults preenche valores por omissão seguros quando o YAML os omite.
func applyDefaults(cfg *Config) {
	if cfg.App.Port == "" {
		cfg.App.Port = "8080"
	}
	if cfg.Dropbox.Path == "" {
		cfg.Dropbox.Path = "/CaetanoBackups"
	}
}

// FindConfigPath localiza o ficheiro de configuração, subindo a árvore de diretórios
// a partir do diretório de trabalho atual à procura de name.
func FindConfigPath(name string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("configuração %q não encontrada a partir de %s", name, dir)
		}
		dir = parent
	}
}
