// Package service reúne os serviços de infraestrutura partilhados por todos os
// módulos do caetano: encriptação, Dropbox e backup diário da base de dados.
package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// CryptoService encripta/desencripta dados com AES-256-GCM, usado para os backups
// diários da base de dados enviados para o Dropbox.
type CryptoService struct {
	key []byte
}

// NewCryptoService cria um CryptoService a partir da chave dada (cfg.Encryption.Key).
// Aceita tanto uma string raw de 32 bytes como uma string codificada em base64.
//
// Uma string de 32 caracteres é sempre usada como bytes crus, mesmo que também
// seja válida em base64 (ex.: uma chave em hexadecimal de 32 caracteres,
// alfabeto subconjunto de base64, decodificaria para 24 bytes por engano). Só
// se recorre a base64 quando o comprimento em bruto não bate certo.
func NewCryptoService(keyStr string) (*CryptoService, error) {
	if keyStr == "" {
		return nil, fmt.Errorf("encryption.key não configurada")
	}

	var key []byte
	if len(keyStr) == 32 {
		key = []byte(keyStr)
	} else if decoded, err := base64.StdEncoding.DecodeString(keyStr); err == nil {
		key = decoded
	} else {
		return nil, fmt.Errorf("encryption.key inválida: tem de ter 32 bytes ou ser base64 de 32 bytes")
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption.key inválida: tem de ter exatamente 32 bytes (AES-256)")
	}

	return &CryptoService{key: key}, nil
}

// Encrypt cifra plaintext com AES-256-GCM, devolvendo nonce+ciphertext.
func (s *CryptoService) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return aesGCM.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt reverte Encrypt.
func (s *CryptoService) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext demasiado curto")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return aesGCM.Open(nil, nonce, ciphertext, nil)
}
