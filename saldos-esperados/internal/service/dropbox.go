package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// DropboxService handles interactions with Dropbox API
type DropboxService struct {
	AppKey       string
	AppSecret    string
	RefreshToken string

	accessToken string
	tokenExpiry time.Time
	mu          sync.RWMutex
}

// DropboxFile represents a file in Dropbox
type DropboxFile struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	PathLower string `json:"path_lower"`
	Size      int64  `json:"size"`
	EntryType string `json:"final_type,omitempty"` // file or folder (from list_folder, it is .tag)
	Tag       string `json:".tag"`
}

// NewDropboxService creates a new service instance
func NewDropboxService() *DropboxService {
	return &DropboxService{
		AppKey:       os.Getenv("DROPBOX_APP_KEY"),
		AppSecret:    os.Getenv("DROPBOX_APP_SECRET"),
		RefreshToken: os.Getenv("DROPBOX_REFRESH_TOKEN"),
	}
}

// ensureToken checks if token is valid, if not, refreshes it using RefreshToken
func (s *DropboxService) ensureToken() error {
	s.mu.RLock()
	if s.accessToken != "" && time.Now().Before(s.tokenExpiry.Add(-5*time.Minute)) {
		s.mu.RUnlock()
		return nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double check after lock
	if s.accessToken != "" && time.Now().Before(s.tokenExpiry.Add(-5*time.Minute)) {
		return nil
	}

	if s.RefreshToken == "" {
		return fmt.Errorf("DROPBOX_REFRESH_TOKEN is missing")
	}

	// Request new token
	tokenURL := "https://api.dropbox.com/oauth2/token"
	data := fmt.Sprintf("grant_type=refresh_token&refresh_token=%s&client_id=%s&client_secret=%s",
		s.RefreshToken, s.AppKey, s.AppSecret)

	req, err := http.NewRequest("POST", tokenURL, bytes.NewBufferString(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to refresh token: %s", string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	s.accessToken = result.AccessToken
	s.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)

	fmt.Println("DEBUG: Dropbox Access Token refreshed successfully")
	return nil
}

// ListFiles lists .csv files in the configured folder
func (s *DropboxService) ListFiles(path string) ([]DropboxFile, error) {
	if err := s.ensureToken(); err != nil {
		return nil, err
	}

	url := "https://api.dropboxapi.com/2/files/list_folder"

	// If path argument is empty, use Env var, else default to root
	if path == "" {
		path = os.Getenv("DROPBOX_PATH")
	}
	// If it's still clean/root, Dropbox expects special handling or empty string?
	// Dropbox API: "path": "" for root.
	// But if Full Dropbox access, "" means ROOT of entire dropbox. We probably don't want that if env var is set.

	// If path is specified in env, use it. e.g. "/SaldosApp"
	// Ensure it starts with / if not empty
	if path != "" && path != "/" && path[0] != '/' {
		path = "/" + path
	}
	if path == "/" {
		path = ""
	}

	payload := map[string]interface{}{
		"path":                                path,
		"recursive":                           false,
		"include_media_info":                  false,
		"include_deleted":                     false,
		"include_has_explicit_shared_members": false,
		"include_mounted_folders":             true,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		// Prepare failure response to check if it is path/not_found
		errorBody, _ := io.ReadAll(resp.Body)
		if bytes.Contains(errorBody, []byte("path/not_found")) {
			// Try to create the folder
			if err := s.createFolder(path); err != nil {
				return nil, fmt.Errorf("failed to create missing folder: %v", err)
			}
			// Retry listing (recursive call, but safe as now folder exists)
			return s.ListFiles(path)
		}
		return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(errorBody))
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Entries []DropboxFile `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Entries, nil
}

// DownloadFile downloads a file by path
func (s *DropboxService) DownloadFile(path string) ([]byte, error) {
	if err := s.ensureToken(); err != nil {
		return nil, err
	}

	url := "https://content.dropboxapi.com/2/files/download"

	// Dropbox content API uses header for args
	arg := map[string]string{"path": path}
	argJSON, _ := json.Marshal(arg)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.accessToken)
	req.Header.Set("Dropbox-API-Arg", string(argJSON))

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("download error %d: %s", resp.StatusCode, string(respBody))

	}

	return io.ReadAll(resp.Body)
}

// createFolder creates a folder at the specified path
func (s *DropboxService) createFolder(path string) error {
	url := "https://api.dropboxapi.com/2/files/create_folder_v2"
	payload := map[string]interface{}{
		"path":       path,
		"autorename": false,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create_folder error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// DeleteFile deletes a file at the specified path
func (s *DropboxService) DeleteFile(path string) error {
	if err := s.ensureToken(); err != nil {
		return err
	}

	url := "https://api.dropboxapi.com/2/files/delete_v2"
	payload := map[string]interface{}{
		"path": path,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete_file error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
