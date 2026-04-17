package security

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"
)

// CertChecker 证书有效期检查器
type CertChecker struct {
	certPaths  []string
	warnBefore time.Duration // 提前多久告警
	alertFunc  func(ctx context.Context, msg string) error
}

// CertCheckResult 证书检查结果
type CertCheckResult struct {
	Path      string    `json:"path"`
	Subject   string    `json:"subject"`
	NotBefore time.Time `json:"not_before"`
	NotAfter  time.Time `json:"not_after"`
	DaysLeft  int       `json:"days_left"`
	Expired   bool      `json:"expired"`
	Warning   bool      `json:"warning"`
}

// NewCertChecker 创建证书检查器
func NewCertChecker(certPaths []string, warnDays int, alertFunc func(ctx context.Context, msg string) error) *CertChecker {
	return &CertChecker{
		certPaths:  certPaths,
		warnBefore: time.Duration(warnDays) * 24 * time.Hour,
		alertFunc:  alertFunc,
	}
}

// CheckAll 检查所有证书
func (c *CertChecker) CheckAll(ctx context.Context) ([]CertCheckResult, error) {
	var results []CertCheckResult
	for _, path := range c.certPaths {
		result, err := c.CheckCert(path)
		if err != nil {
			results = append(results, CertCheckResult{
				Path:    path,
				Expired: true,
				Warning: true,
			})
			continue
		}
		results = append(results, *result)

		// 发送告警
		if result.Warning || result.Expired {
			c.sendAlert(ctx, result)
		}
	}
	return results, nil
}

// CheckCert 检查单个证书文件
func (c *CertChecker) CheckCert(certPath string) (*CertCheckResult, error) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cert file %s: %w", certPath, err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from %s", certPath)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate %s: %w", certPath, err)
	}

	now := time.Now()
	daysLeft := int(cert.NotAfter.Sub(now).Hours() / 24)
	expired := now.After(cert.NotAfter)
	warning := !expired && cert.NotAfter.Sub(now) < c.warnBefore

	return &CertCheckResult{
		Path:      certPath,
		Subject:   cert.Subject.CommonName,
		NotBefore: cert.NotBefore,
		NotAfter:  cert.NotAfter,
		DaysLeft:  daysLeft,
		Expired:   expired,
		Warning:   warning,
	}, nil
}

func (c *CertChecker) sendAlert(ctx context.Context, result *CertCheckResult) {
	if c.alertFunc == nil {
		return
	}

	status := "即将过期"
	if result.Expired {
		status = "已过期"
	}

	msg := fmt.Sprintf("⚠️ 证书%s告警\n路径: %s\n主题: %s\n到期时间: %s\n剩余天数: %d",
		status, result.Path, result.Subject,
		result.NotAfter.Format("2006-01-02"), result.DaysLeft)

	c.alertFunc(ctx, msg)
}

// StartPeriodicCheck 启动定期检查（每天检查一次）
func (c *CertChecker) StartPeriodicCheck(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// 启动时立即检查一次
	c.CheckAll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.CheckAll(ctx)
		}
	}
}
