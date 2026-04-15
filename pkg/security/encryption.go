package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// KeyManager 密钥管理器
type KeyManager struct {
	masterKey []byte
}

// NewKeyManager 创建密钥管理器
func NewKeyManager(masterKey string) *KeyManager {
	// 使用 SHA-256 生成固定长度的密钥
	hash := sha256.Sum256([]byte(masterKey))
	return &KeyManager{
		masterKey: hash[:],
	}
}

// Encrypt 加密数据
func (km *KeyManager) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(km.masterKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密数据
func (km *KeyManager) Decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(km.masterKey)
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

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// RotateKey 密钥轮转
func (km *KeyManager) RotateKey(newMasterKey string) *KeyManager {
	hash := sha256.Sum256([]byte(newMasterKey))
	return &KeyManager{
		masterKey: hash[:],
	}
}

// APIKeyManager API 密钥管理器
type APIKeyManager struct {
	keyManager *KeyManager
}

// NewAPIKeyManager 创建 API 密钥管理器
func NewAPIKeyManager(keyManager *KeyManager) *APIKeyManager {
	return &APIKeyManager{
		keyManager: keyManager,
	}
}

// EncryptAPIKey 加密 API 密钥
func (m *APIKeyManager) EncryptAPIKey(apiKey string) (string, error) {
	return m.keyManager.Encrypt(apiKey)
}

// DecryptAPIKey 解密 API 密钥
func (m *APIKeyManager) DecryptAPIKey(encryptedKey string) (string, error) {
	return m.keyManager.Decrypt(encryptedKey)
}

// CertificateManager 证书管理器
type CertificateManager struct {
	keyManager *KeyManager
}

// NewCertificateManager 创建证书管理器
func NewCertificateManager(keyManager *KeyManager) *CertificateManager {
	return &CertificateManager{
		keyManager: keyManager,
	}
}

// EncryptCertificate 加密证书
func (m *CertificateManager) EncryptCertificate(cert string) (string, error) {
	return m.keyManager.Encrypt(cert)
}

// DecryptCertificate 解密证书
func (m *CertificateManager) DecryptCertificate(encryptedCert string) (string, error) {
	return m.keyManager.Decrypt(encryptedCert)
}
