package handler

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fruigodinho/caetano/saldos-esperados/internal/adapter/parser"
	"github.com/fruigodinho/caetano/saldos-esperados/internal/core/domain"
	"github.com/fruigodinho/caetano/saldos-esperados/internal/service"
	"github.com/gin-gonic/gin"
)

type DataHandler struct {
	validator    *service.Validator
	csvParser    *parser.CSVParser
	auditService *service.AuditService
}

func NewDataHandler(db *sql.DB) *DataHandler {
	return &DataHandler{
		validator:    service.NewValidator(db),
		csvParser:    parser.NewCSVParser(),
		auditService: service.NewAuditService(db),
	}
}

func (h *DataHandler) ShowForm(c *gin.Context) {
	c.HTML(http.StatusOK, "saldos-esperados/data", AddCommonData(c, gin.H{
		"Title": "Upload Balancete",
	}))
}

func (h *DataHandler) ShowManualForm(c *gin.Context) {
	c.HTML(http.StatusOK, "saldos-esperados/data_manual", AddCommonData(c, gin.H{
		"Title": "Dados Manual",
	}))
}

// ShowDropboxForm renders the Dropbox file selection form
func (h *DataHandler) ShowDropboxForm(c *gin.Context) {
	c.HTML(http.StatusOK, "saldos-esperados/data_dropbox", AddCommonData(c, gin.H{
		"Title": "Importar do Dropbox",
	}))
}

// ListDropboxFiles lists files from the configured Dropbox folder
func (h *DataHandler) ListDropboxFiles(c *gin.Context) {
	srv := service.NewDropboxService()
	files, err := srv.ListFiles("") // List root
	if err != nil {
		fmt.Printf("DEBUG: ListFiles error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro ao listar ficheiros: %v", err)})
		return
	}

	// Filter only CSV files
	var csvFiles []service.DropboxFile
	for _, f := range files {
		if strings.HasSuffix(strings.ToLower(f.Name), ".csv") {
			csvFiles = append(csvFiles, f)
		}
	}

	c.JSON(http.StatusOK, gin.H{"files": csvFiles})
}

// ProcessDropboxFileRequest defines the payload for processing a Dropbox file
type ProcessDropboxFileRequest struct {
	Path     string `json:"path" binding:"required"`
	Filename string `json:"filename"` // Optional, passed from frontend
}

// ProcessDropboxFile downloads and processes a file from Dropbox
func (h *DataHandler) ProcessDropboxFile(c *gin.Context) {
	var req ProcessDropboxFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
		return
	}

	srv := service.NewDropboxService()

	// 1. Download File
	content, err := srv.DownloadFile(req.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro ao descarregar ficheiro: %v", err)})
		return
	}

	// Generate a filename
	filename := req.Filename
	if filename == "" {
		filename = fmt.Sprintf("dropbox_import_%d.csv", time.Now().Unix())
	}
	// Ensure filename ends with .csv
	if !strings.HasSuffix(strings.ToLower(filename), ".csv") {
		filename += ".csv"
	}

	// 2. Process File logic (Reuse from previous implementation)
	tempFile, err := os.CreateTemp("", "dropbox-upload-*.csv")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao criar ficheiro temporário"})
		return
	}
	defer os.Remove(tempFile.Name()) // Clean up

	if _, err := tempFile.Write(content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao escrever ficheiro temporário"})
		return
	}
	tempFile.Close()

	// Extract Label
	label, err := h.csvParser.ExtractLabel(tempFile.Name())
	if err != nil {
		fmt.Printf("Failed to extract label: %v\n", err)
		label = ""
	}

	// Encrypt label
	if label != "" {
		encryptedLabel, err := service.EncryptText(label)
		if err != nil {
			fmt.Printf("Failed to encrypt label: %v\n", err)
			label = ""
		} else {
			label = encryptedLabel
		}
	}

	// Create Upload Record
	uploadID, err := h.validator.CreateUploadRecord(c.Request.Context(), filename, label)
	if err != nil {
		fmt.Printf("DEBUG: CreateUploadRecord Error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro ao criar registo de upload: %v", err)})
		return
	}

	// Process CSV
	err = h.csvParser.ParseBalanceSheet(tempFile.Name(), 100, func(batch []domain.AccountEntry) error {
		return h.validator.ProcessBatch(c.Request.Context(), uploadID, batch)
	})

	if err != nil {
		_ = h.validator.UpdateUploadStatus(c.Request.Context(), uploadID, "failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro ao processar CSV: %v", err)})
		return
	}

	// Update Status
	if err := h.validator.UpdateUploadStatus(c.Request.Context(), uploadID, "processed"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar estado"})
		return
	}

	// Audit Log
	currentUserID := CurrentUserID(c)
	_ = h.auditService.Log(c.Request.Context(), currentUserID, "UPLOAD", "UPLOAD", strconv.Itoa(int(uploadID)), "Dropbox Import processed", c.ClientIP())

	// Delete file from Dropbox after successful processing
	if err := srv.DeleteFile(req.Path); err != nil {
		// Log error but don't fail the response as data is already processed
		fmt.Printf("WARNING: Failed to delete file from Dropbox after processing: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Ficheiro processado com sucesso"})
}

type DataRequest struct {
	Filename string `json:"filename" binding:"required"`
	Content  string `json:"content" binding:"required"` // Base64 encoded content
}

func (h *DataHandler) Process(c *gin.Context) {
	isAjax := c.GetHeader("X-Requested-With") == "XMLHttpRequest" || c.GetHeader("Accept") == "application/json"

	var filename string
	var tempFile string

	// Check Content-Type to decide how to process
	if c.ContentType() == "application/json" {
		var req DataRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
			return
		}

		filename = req.Filename

		// Decode Base64
		data, err := base64.StdEncoding.DecodeString(req.Content)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Conteúdo do ficheiro inválido (Base64)"})
			return
		}

		// Save to temp file
		tempDir := os.TempDir()
		tempFile = filepath.Join(tempDir, fmt.Sprintf("upload_%d_%s", time.Now().Unix(), filename))
		if err := os.WriteFile(tempFile, data, 0644); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao guardar ficheiro temporário"})
			return
		}
	} else {
		// Fallback to multipart/form-data (Legacy support or direct POST)
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			if isAjax {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Ficheiro obrigatório"})
			} else {
				c.HTML(http.StatusBadRequest, "saldos-esperados/data", AddCommonData(c, gin.H{
					"Title": "Upload Balancete",
					"Error": "Ficheiro obrigatório",
				}))
			}
			return
		}
		defer file.Close()

		filename = header.Filename
		tempDir := os.TempDir()
		tempFile = filepath.Join(tempDir, fmt.Sprintf("upload_%d_%s", time.Now().Unix(), filename))

		out, err := os.Create(tempFile)
		if err != nil {
			h.handleError(c, isAjax, "Erro ao guardar ficheiro", err)
			return
		}
		defer out.Close()

		_, err = io.Copy(out, file)
		if err != nil {
			out.Close()
			h.handleError(c, isAjax, "Erro ao guardar ficheiro", err)
			return
		}
		out.Close() // Explicitly close to ensure content is flushed to disk
	}

	// Ensure temp file is removed at the end
	defer os.Remove(tempFile)

	// Extract label from CSV first line
	label, err := h.csvParser.ExtractLabel(tempFile)
	if err != nil {
		// Log but continue - label is optional
		label = ""
	}

	// Encrypt label
	if label != "" {
		encryptedLabel, err := service.EncryptText(label)
		if err != nil {
			fmt.Printf("Failed to encrypt label: %v\n", err)
			// Continue with plaintext or empty? Let's continue with plaintext as fallback?
			// No, if encryption fails, better to fail or store empty to avoid leaking plaintext if strict.
			// But requirement says "A label deveria ser encriptada".
			// Let's log error and store empty or fail.
			// Given it's a security feature, let's fail safe -> store empty or fail.
			// Let's store empty to not break the flow, but log error.
			label = ""
		} else {
			label = encryptedLabel
		}
	}

	// Create upload record with label
	uploadID, err := h.validator.CreateUploadRecord(context.Background(), filename, label)
	if err != nil {
		h.handleError(c, isAjax, "Erro ao criar registo de upload", err)
		return
	}

	// Process CSV
	err = h.csvParser.ParseBalanceSheet(tempFile, 100, func(batch []domain.AccountEntry) error {
		return h.validator.ProcessBatch(context.Background(), uploadID, batch)
	})

	if err != nil {
		// Mark as failed
		_ = h.validator.UpdateUploadStatus(context.Background(), uploadID, "failed")
		h.handleError(c, isAjax, fmt.Sprintf("Erro ao processar CSV: %v", err), nil)
		return
	}

	// Mark as processed
	err = h.validator.UpdateUploadStatus(context.Background(), uploadID, "processed")
	if err != nil {
		fmt.Printf("Failed to update status to processed: %v\n", err)
	}

	// Audit Log
	currentUserID := CurrentUserID(c)
	_ = h.auditService.Log(c.Request.Context(), currentUserID, "UPLOAD", "UPLOAD", strconv.Itoa(int(uploadID)), "Uploaded file "+filename, c.ClientIP())

	if isAjax {
		c.JSON(http.StatusOK, gin.H{"message": "Ficheiro processado com sucesso"})
	} else {
		c.Redirect(http.StatusFound, "/saldos-esperados/dashboard")
	}
}

// InitDataRequest defines the payload for initializing an upload
type InitDataRequest struct {
	Filename string `json:"filename" binding:"required"`
	Label    string `json:"label"`
}

// InitUpload initializes a new upload session
func (h *DataHandler) InitUpload(c *gin.Context) {
	var req InitDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
		return
	}

	// Encrypt label if present
	label := req.Label
	if label != "" {
		encryptedLabel, err := service.EncryptText(label)
		if err != nil {
			fmt.Printf("Failed to encrypt label: %v\n", err)
			label = ""
		} else {
			label = encryptedLabel
		}
	}

	// Create upload record
	uploadID, err := h.validator.CreateUploadRecord(c.Request.Context(), req.Filename, label)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao criar registo de upload"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"upload_id": uploadID})
}

// BatchDataRequest defines the payload for a batch of entries
type BatchDataRequest struct {
	UploadID int32                 `json:"upload_id" binding:"required"`
	Entries  []domain.AccountEntry `json:"entries" binding:"required"`
}

// BatchUpload processes a batch of account entries
func (h *DataHandler) BatchUpload(c *gin.Context) {
	var req BatchDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
		return
	}

	if err := h.validator.ProcessBatch(c.Request.Context(), req.UploadID, req.Entries); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro ao processar lote: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// FinishDataRequest defines the payload for finishing an upload
type FinishDataRequest struct {
	UploadID int32 `json:"upload_id" binding:"required"`
}

// FinishUpload marks an upload as processed
func (h *DataHandler) FinishUpload(c *gin.Context) {
	var req FinishDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
		return
	}

	// Mark as processed
	if err := h.validator.UpdateUploadStatus(c.Request.Context(), req.UploadID, "processed"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao finalizar upload"})
		return
	}

	// Audit Log
	currentUserID := CurrentUserID(c)
	_ = h.auditService.Log(c.Request.Context(), currentUserID, "UPLOAD", "UPLOAD", strconv.Itoa(int(req.UploadID)), "Upload finished via API", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "Upload concluído com sucesso"})
}

func (h *DataHandler) handleError(c *gin.Context, isAjax bool, msg string, err error) {
	if isAjax {
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
	} else {
		c.HTML(http.StatusInternalServerError, "saldos-esperados/data", AddCommonData(c, gin.H{
			"Title": "Upload Balancete",
			"Error": msg,
		}))
	}
}
