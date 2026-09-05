package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"os"
)

var encryptionKey []byte

// InitEncryption inicializa o sistema de encriptação com a chave do ambiente.
//
// A chave de encriptação (ENCRYPTION_KEY) deve ter exatamente 32 bytes (256 bits)
// para AES-256. Se a chave não estiver definida, a encriptação não será inicializada
// e operações de encriptação/desencriptação falharão.
//
// Returns:
//   - bool: true se inicializado com sucesso, false caso contrário
func InitEncryption() bool {
	key := os.Getenv("ENCRYPTION_KEY")
	if key == "" {
		log.Println("⚠️  WARNING: ENCRYPTION_KEY not set - TOTP encryption disabled")
		return false
	}

	if len(key) != 32 {
		log.Printf("⚠️  WARNING: ENCRYPTION_KEY must be exactly 32 characters (current: %d) - encryption disabled", len(key))
		return false
	}

	encryptionKey = []byte(key)
	log.Println("Encryption initialized (AES-256-GCM)")
	return true
}

// EncryptTOTPSecret encripta um segredo TOTP usando AES-256-GCM.
//
// Parameters:
//   - plaintext: segredo TOTP em plaintext (base32)
//
// Returns:
//   - string: segredo encriptado em base64
//   - error: erro se a encriptação falhar
//
// O formato retornado é: base64(nonce + ciphertext)
// onde nonce é usado pelo GCM para garantir unicidade.
func EncryptTOTPSecret(plaintext string) (string, error) {
	if encryptionKey == nil {
		return "", errors.New("encryption not initialized - call InitEncryption() first")
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Gerar nonce aleatório
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encriptar (nonce é prefixado ao ciphertext)
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptTOTPSecret desencripta um segredo TOTP usando AES-256-GCM.
//
// Parameters:
//   - ciphertextBase64: segredo encriptado em base64 (formato: base64(nonce + ciphertext))
//
// Returns:
//   - string: segredo TOTP em plaintext (base32)
//   - error: erro se a desencriptação falhar
//
// Erros comuns:
//   - "encryption not initialized" se InitEncryption() não foi chamada
//   - "ciphertext too short" se dados estão corrompidos
//   - "cipher: message authentication failed" se a chave estiver errada
func DecryptTOTPSecret(ciphertextBase64 string) (string, error) {
	if encryptionKey == nil {
		return "", errors.New("encryption not initialized - call InitEncryption() first")
	}

	// Decodificar base64
	data, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	// Extrair nonce e ciphertext
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// EncryptText encripta um texto qualquer usando AES-256-GCM.
//
// Parameters:
//   - plaintext: texto a encriptar
//
// Returns:
//   - string: texto encriptado em base64
//   - error: erro se a encriptação falhar
func EncryptText(plaintext string) (string, error) {
	return EncryptTOTPSecret(plaintext) // Reutiliza a mesma lógica
}

// DecryptText desencripta um texto encriptado com EncryptText.
//
// Parameters:
//   - ciphertextBase64: texto encriptado em base64
//
// Returns:
//   - string: texto original
//   - error: erro se a desencriptação falhar
func DecryptText(ciphertextBase64 string) (string, error) {
	return DecryptTOTPSecret(ciphertextBase64) // Reutiliza a mesma lógica
}
