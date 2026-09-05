package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"path"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

// maxDropboxUpload é o limite do endpoint de upload simples do Dropbox (150MB).
const maxDropboxUpload = 150 * 1024 * 1024

// backupFilePrefix e backupFileSuffix identificam os ficheiros de backup geridos
// pela rotação.
const (
	backupFilePrefix = "caetano_"
	backupFileSuffix = ".sql.gz.enc"
)

// BackupService efetua backups diários integrais da base de dados caetano para o
// Dropbox. O dump (pg_dump) é comprimido com gzip e encriptado com AES-256-GCM
// antes do upload.
type BackupService struct {
	dropbox     *DropboxService
	crypto      *CryptoService
	databaseURL string // connection string sem search_path (pg_dump rejeita-o)
	keep        int    // número de backups a manter no Dropbox
	now         func() time.Time
}

// NewBackupService cria o serviço de backup a partir do DSN dado.
func NewBackupService(dsn string, dropbox *DropboxService, crypto *CryptoService) (*BackupService, error) {
	dbURL, err := sanitizeDatabaseURL(dsn)
	if err != nil {
		return nil, fmt.Errorf("db.dsn inválido: %w", err)
	}

	return &BackupService{
		dropbox:     dropbox,
		crypto:      crypto,
		databaseURL: dbURL,
		keep:        3,
		now:         time.Now,
	}, nil
}

// sanitizeDatabaseURL remove o parâmetro search_path (específico do pgx; o
// libpq/pg_dump rejeita-o).
func sanitizeDatabaseURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Del("search_path")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// backupFileName gera o nome do ficheiro de backup com timestamp.
func backupFileName(t time.Time) string {
	return backupFilePrefix + t.Format("20060102_150405") + backupFileSuffix
}

// nextBackupTime calcula a próxima 01:00 local a partir de now.
func nextBackupTime(now time.Time) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), 1, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

// Estado global do backup, para exibição em UI.
var (
	backupScheduled atomic.Bool
	lastBackupAt    atomic.Value // time.Time
)

// LastBackupAt devolve o instante do último backup bem-sucedido desta sessão
// (zero value se ainda não correu nenhum).
func LastBackupAt() time.Time {
	if t, ok := lastBackupAt.Load().(time.Time); ok {
		return t
	}
	return time.Time{}
}

// BackupScheduled indica se o scheduler de backups está ativo nesta sessão.
func BackupScheduled() bool {
	return backupScheduled.Load()
}

// Start corre o scheduler: executa o backup todos os dias às 01:00. O cálculo da
// próxima execução é refeito a cada iteração (seguro com mudanças DST).
func (s *BackupService) Start(ctx context.Context) {
	backupScheduled.Store(true)
	for {
		next := nextBackupTime(s.now())
		log.Printf("[BACKUP] próximo backup agendado para %s", next.Format("2006-01-02 15:04:05"))

		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if err := s.RunBackup(ctx); err != nil {
				log.Printf("[BACKUP] erro no backup: %v", err)
			}
		}
	}
}

// RunBackup executa um backup integral: pg_dump -> gzip -> encrypt -> upload -> rotação.
func (s *BackupService) RunBackup(ctx context.Context) error {
	start := s.now()
	log.Println("[BACKUP] a iniciar backup integral da base de dados...")

	if _, err := exec.LookPath("pg_dump"); err != nil {
		return fmt.Errorf("pg_dump não encontrado no PATH: %w", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	cmd := exec.CommandContext(ctx, "pg_dump", s.databaseURL,
		"--no-owner", "--no-privileges", "--clean", "--if-exists")
	cmd.Stdout = gz
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump falhou: %w - %s", err, strings.TrimSpace(stderr.String()))
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("erro ao comprimir dump: %w", err)
	}

	enc, err := s.crypto.Encrypt(buf.Bytes())
	if err != nil {
		return fmt.Errorf("erro ao encriptar dump: %w", err)
	}
	if len(enc) > maxDropboxUpload {
		return fmt.Errorf("backup demasiado grande para upload simples (%d bytes > 150MB)", len(enc))
	}

	dropboxPath := path.Join(s.dropbox.BackupsPath(), backupFileName(s.now()))
	if err := s.dropbox.Upload(dropboxPath, enc); err != nil {
		return fmt.Errorf("erro no upload do backup: %w", err)
	}

	log.Printf("[BACKUP] backup concluído em %s: %s (%d bytes)",
		time.Since(start).Round(time.Second), dropboxPath, len(enc))
	lastBackupAt.Store(s.now())

	if err := s.rotate(); err != nil {
		log.Printf("[BACKUP] aviso: erro na rotação de backups: %v", err)
	}
	return nil
}

// rotate mantém apenas os `keep` backups mais recentes na pasta de backups do Dropbox.
func (s *BackupService) rotate() error {
	entries, err := s.dropbox.ListFolder(s.dropbox.BackupsPath())
	if err != nil {
		return err
	}

	var backups []DropboxFile
	for _, e := range entries {
		if e.Tag == "file" && strings.HasPrefix(e.Name, backupFilePrefix) && strings.HasSuffix(e.Name, backupFileSuffix) {
			backups = append(backups, e)
		}
	}
	if len(backups) <= s.keep {
		return nil
	}

	// O timestamp no nome ordena lexicograficamente - mais recente primeiro.
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Name > backups[j].Name
	})

	for _, old := range backups[s.keep:] {
		target := path.Join(s.dropbox.BackupsPath(), old.Name)
		if err := s.dropbox.Delete(target); err != nil {
			log.Printf("[BACKUP] erro ao eliminar backup antigo %s: %v", old.Name, err)
		} else {
			log.Printf("[BACKUP] backup antigo eliminado: %s", old.Name)
		}
	}
	return nil
}
