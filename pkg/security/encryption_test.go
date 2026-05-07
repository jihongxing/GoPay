package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyManager_EncryptDecrypt(t *testing.T) {
	km := NewKeyManager("test-master-key-12345")

	plaintext := "sensitive-api-key-data"
	encrypted, err := km.Encrypt(plaintext)
	assert.NoError(t, err)
	assert.NotEmpty(t, encrypted)
	assert.NotEqual(t, plaintext, encrypted)

	decrypted, err := km.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestKeyManager_EncryptDecrypt_EmptyString(t *testing.T) {
	km := NewKeyManager("test-key")

	encrypted, err := km.Encrypt("")
	assert.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	decrypted, err := km.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, "", decrypted)
}

func TestKeyManager_EncryptDecrypt_LongData(t *testing.T) {
	km := NewKeyManager("test-key")

	longData := "-----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBgkqhkiG9w0BAQE...\n-----END PRIVATE KEY-----"
	encrypted, err := km.Encrypt(longData)
	assert.NoError(t, err)

	decrypted, err := km.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, longData, decrypted)
}

func TestKeyManager_EncryptProducesDifferentCiphertexts(t *testing.T) {
	km := NewKeyManager("test-key")

	enc1, _ := km.Encrypt("same-data")
	enc2, _ := km.Encrypt("same-data")

	// Due to random nonce, same plaintext should produce different ciphertexts
	assert.NotEqual(t, enc1, enc2)
}

func TestKeyManager_DecryptInvalidBase64(t *testing.T) {
	km := NewKeyManager("test-key")

	_, err := km.Decrypt("not-valid-base64!!!")
	assert.Error(t, err)
}

func TestKeyManager_DecryptTooShort(t *testing.T) {
	km := NewKeyManager("test-key")

	// Valid base64 but too short for nonce + ciphertext
	_, err := km.Decrypt("YQ==")
	assert.Error(t, err)
}

func TestKeyManager_DecryptWrongKey(t *testing.T) {
	km1 := NewKeyManager("key-one")
	km2 := NewKeyManager("key-two")

	encrypted, err := km1.Encrypt("secret")
	assert.NoError(t, err)

	_, err = km2.Decrypt(encrypted)
	assert.Error(t, err)
}

func TestKeyManager_RotateKey(t *testing.T) {
	km1 := NewKeyManager("old-key")
	encrypted, err := km1.Encrypt("secret-data")
	assert.NoError(t, err)

	// Old key can decrypt
	decrypted, err := km1.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, "secret-data", decrypted)

	// Rotated key cannot decrypt old data
	km2 := km1.RotateKey("new-key")
	_, err = km2.Decrypt(encrypted)
	assert.Error(t, err)

	// Rotated key can encrypt/decrypt new data
	newEncrypted, err := km2.Encrypt("new-secret")
	assert.NoError(t, err)
	newDecrypted, err := km2.Decrypt(newEncrypted)
	assert.NoError(t, err)
	assert.Equal(t, "new-secret", newDecrypted)
}

func TestAPIKeyManager_EncryptDecrypt(t *testing.T) {
	km := NewKeyManager("master")
	akm := NewAPIKeyManager(km)

	encrypted, err := akm.EncryptAPIKey("sk_live_abc123")
	assert.NoError(t, err)

	decrypted, err := akm.DecryptAPIKey(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, "sk_live_abc123", decrypted)
}

func TestCertificateManager_EncryptDecrypt(t *testing.T) {
	km := NewKeyManager("master")
	cm := NewCertificateManager(km)

	cert := "-----BEGIN CERTIFICATE-----\nMIIBxTCCAW...\n-----END CERTIFICATE-----"
	encrypted, err := cm.EncryptCertificate(cert)
	assert.NoError(t, err)

	decrypted, err := cm.DecryptCertificate(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, cert, decrypted)
}
