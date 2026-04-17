package security

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestCert(t *testing.T, dir, name string, notBefore, notAfter time.Time) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: name},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	path := filepath.Join(dir, name+".pem")
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()

	err = pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	require.NoError(t, err)

	return path
}

func TestCertChecker_ValidCert(t *testing.T) {
	dir := t.TempDir()
	path := createTestCert(t, dir, "valid",
		time.Now().Add(-24*time.Hour),
		time.Now().Add(365*24*time.Hour))

	checker := NewCertChecker([]string{path}, 30, nil)
	result, err := checker.CheckCert(path)
	require.NoError(t, err)

	assert.False(t, result.Expired)
	assert.False(t, result.Warning)
	assert.True(t, result.DaysLeft > 300)
}

func TestCertChecker_ExpiredCert(t *testing.T) {
	dir := t.TempDir()
	path := createTestCert(t, dir, "expired",
		time.Now().Add(-365*24*time.Hour),
		time.Now().Add(-24*time.Hour))

	checker := NewCertChecker([]string{path}, 30, nil)
	result, err := checker.CheckCert(path)
	require.NoError(t, err)

	assert.True(t, result.Expired)
	assert.True(t, result.DaysLeft < 0)
}

func TestCertChecker_WarningCert(t *testing.T) {
	dir := t.TempDir()
	path := createTestCert(t, dir, "warning",
		time.Now().Add(-24*time.Hour),
		time.Now().Add(15*24*time.Hour)) // 15 days left, warn at 30

	checker := NewCertChecker([]string{path}, 30, nil)
	result, err := checker.CheckCert(path)
	require.NoError(t, err)

	assert.False(t, result.Expired)
	assert.True(t, result.Warning)
}

func TestCertChecker_CheckAll_WithAlert(t *testing.T) {
	dir := t.TempDir()
	expiredPath := createTestCert(t, dir, "expired",
		time.Now().Add(-365*24*time.Hour),
		time.Now().Add(-24*time.Hour))

	var alertCalled bool
	alertFn := func(ctx context.Context, msg string) error {
		alertCalled = true
		return nil
	}

	checker := NewCertChecker([]string{expiredPath}, 30, alertFn)
	results, err := checker.CheckAll(context.Background())
	require.NoError(t, err)

	assert.Len(t, results, 1)
	assert.True(t, alertCalled)
}

func TestCertChecker_NonExistentFile(t *testing.T) {
	checker := NewCertChecker([]string{"/nonexistent/cert.pem"}, 30, nil)
	_, err := checker.CheckCert("/nonexistent/cert.pem")
	assert.Error(t, err)
}
