package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DropboxService gere a interação com a API do Dropbox, usada para os backups
// diários da base de dados e (futuramente) para importação de balancetes.
type DropboxService struct {
	AppKey       string
	AppSecret    string
	RefreshToken string
	BasePath     string

	accessToken string
	tokenExpiry time.Time
	mu          sync.RWMutex

	apiURL     string
	contentURL string
	authURL    string
}

// DropboxFile representa metadados básicos de um ficheiro no Dropbox.
type DropboxFile struct {
	Name           string    `json:"name"`
	PathLower      string    `json:"path_lower"`
	Size           int64     `json:"size"`
	Tag            string    `json:".tag"` // file ou folder
	ServerModified time.Time `json:"server_modified"`
}

// NewDropboxService cria o serviço a partir das credenciais dadas.
//
// Parameters:
//   - appKey, appSecret, refreshToken: credenciais da app Dropbox
//   - basePath: pasta base no Dropbox (ex.: "/CaetanoBackups")
//   - isProduction: separa a árvore de ficheiros em dev/release, para nunca
//     misturar backups de desenvolvimento com os de produção
func NewDropboxService(appKey, appSecret, refreshToken, basePath string, isProduction bool) *DropboxService {
	if strings.TrimSpace(basePath) == "" {
		basePath = "/CaetanoBackups"
	}

	envFolder := "dev"
	if isProduction {
		envFolder = "release"
	}
	basePath = filepath.ToSlash(filepath.Join(basePath, envFolder))

	return &DropboxService{
		AppKey:       strings.TrimSpace(appKey),
		AppSecret:    strings.TrimSpace(appSecret),
		RefreshToken: strings.TrimSpace(refreshToken),
		BasePath:     basePath,
		apiURL:       "https://api.dropboxapi.com",
		contentURL:   "https://content.dropboxapi.com",
		authURL:      "https://api.dropbox.com",
	}
}

// IsConfigured indica se existem credenciais suficientes para usar o Dropbox.
func (s *DropboxService) IsConfigured() bool {
	return s.AppKey != "" && s.AppSecret != "" && s.RefreshToken != ""
}

// BackupsPath devolve a pasta de backups da BD no Dropbox (BasePath já inclui dev/release).
func (s *DropboxService) BackupsPath() string {
	prefix := s.BasePath
	if prefix == "" {
		prefix = "/CaetanoBackups"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	return path.Join(prefix, "backups")
}

// ensureToken garante que temos um token de acesso válido, renovando-o se necessário.
func (s *DropboxService) ensureToken() error {
	s.mu.RLock()
	if s.accessToken != "" && time.Now().Before(s.tokenExpiry.Add(-5*time.Minute)) {
		s.mu.RUnlock()
		return nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.accessToken != "" && time.Now().Before(s.tokenExpiry.Add(-5*time.Minute)) {
		return nil
	}
	if s.RefreshToken == "" {
		return fmt.Errorf("dropbox.refresh_token não configurado")
	}

	log.Println("[DROPBOX] a renovar token de acesso...")

	tokenURL := s.authURL + "/oauth2/token"
	data := fmt.Sprintf("grant_type=refresh_token&refresh_token=%s&client_id=%s&client_secret=%s",
		s.RefreshToken, s.AppKey, s.AppSecret)

	req, err := http.NewRequest("POST", tokenURL, bytes.NewBufferString(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("falha ao renovar token (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if result.AccessToken == "" {
		return fmt.Errorf("dropbox devolveu um token vazio")
	}

	s.accessToken = result.AccessToken
	expiresIn := result.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 3600
	}
	s.tokenExpiry = time.Now().Add(time.Duration(expiresIn) * time.Second)

	log.Println("[DROPBOX] token de acesso renovado com sucesso")
	return nil
}

func (s *DropboxService) token() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return strings.TrimSpace(s.accessToken)
}

// ListFolder lista os ficheiros de uma pasta do Dropbox (não recursivo). Se a
// pasta não existir, devolve uma lista vazia em vez de erro.
func (s *DropboxService) ListFolder(dropboxPath string) ([]DropboxFile, error) {
	if err := s.ensureToken(); err != nil {
		return nil, err
	}

	url := s.apiURL + "/2/files/list_folder"
	payload := map[string]any{"path": dropboxPath, "recursive": false}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.token())
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		if strings.Contains(string(respBody), "path/not_found") {
			return []DropboxFile{}, nil
		}
		return nil, fmt.Errorf("erro ao listar ficheiros (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Entries []DropboxFile `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Entries, nil
}

// Upload envia conteúdo para um caminho absoluto no Dropbox (mode overwrite).
func (s *DropboxService) Upload(dropboxPath string, content []byte) error {
	if err := s.ensureToken(); err != nil {
		return err
	}

	url := s.contentURL + "/2/files/upload"
	arg := map[string]any{
		"path": dropboxPath, "mode": "overwrite", "autorename": false,
		"mute": false, "strict_conflict": false,
	}
	argJSON, _ := json.Marshal(arg)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(content))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.token())
	req.Header.Set("Dropbox-API-Arg", string(argJSON))
	req.Header.Set("Content-Type", "application/octet-stream")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("erro no upload (HTTP %d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// Delete remove um ficheiro de um caminho absoluto no Dropbox (404 não é erro).
func (s *DropboxService) Delete(dropboxPath string) error {
	if err := s.ensureToken(); err != nil {
		return err
	}

	url := s.apiURL + "/2/files/delete_v2"
	payload := map[string]string{"path": dropboxPath}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.token())
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("erro ao eliminar ficheiro (HTTP %d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}
