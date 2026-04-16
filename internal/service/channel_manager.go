package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	"gopay/internal/models"
	"gopay/pkg/channel"
	"gopay/pkg/channel/alipay"
	"gopay/pkg/channel/wechat"
	"gopay/pkg/errors"
	"gopay/pkg/logger"
)

// ChannelManager 支付渠道管理器
type ChannelManager struct {
	db        *sql.DB
	providers map[string]channel.PaymentChannel // key: appID_channel
	mu        sync.RWMutex
}

// NewChannelManager 创建渠道管理器
func NewChannelManager(db *sql.DB) *ChannelManager {
	return &ChannelManager{
		db:        db,
		providers: make(map[string]channel.PaymentChannel),
	}
}

// GetProvider 获取支付渠道 Provider
func (m *ChannelManager) GetProvider(appID, channelName string) (channel.PaymentChannel, error) {
	key := fmt.Sprintf("%s_%s", appID, channelName)

	// 先尝试从缓存获取
	m.mu.RLock()
	provider, exists := m.providers[key]
	m.mu.RUnlock()

	if exists {
		return provider, nil
	}

	// 缓存未命中，从数据库加载
	m.mu.Lock()
	defer m.mu.Unlock()

	// 双重检查
	if provider, exists := m.providers[key]; exists {
		return provider, nil
	}

	// 从数据库加载配置
	var config models.ChannelConfig
	err := m.db.QueryRow(`
		SELECT id, app_id, channel, config, status, created_at, updated_at
		FROM channel_configs
		WHERE app_id = $1 AND channel = $2
	`, appID, channelName).Scan(
		&config.ID,
		&config.AppID,
		&config.Channel,
		&config.Config,
		&config.Status,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.NewChannelNotFoundError(appID, channelName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load channel config: %w", err)
	}

	// 检查渠道状态
	if config.Status != "active" {
		return nil, errors.NewChannelInactiveError(appID, channelName)
	}

	// 根据渠道类型创建 Provider
	provider, err = m.createProvider(channelName, config.Config)
	if err != nil {
		return nil, err
	}

	// 缓存 Provider
	m.providers[key] = provider

	logger.Info("Channel provider loaded: appID=%s, channel=%s", appID, channelName)

	return provider, nil
}

// ListProvidersByChannelPrefix 列出指定前缀的全部可用 Provider
func (m *ChannelManager) ListProvidersByChannelPrefix(prefix string) ([]channel.PaymentChannel, error) {
	if m.db == nil {
		return nil, fmt.Errorf("channel manager is not initialized")
	}

	rows, err := m.db.Query(`
		SELECT id, app_id, channel, config, status, created_at, updated_at
		FROM channel_configs
		WHERE channel LIKE $1 AND status = 'active'
		ORDER BY updated_at DESC
	`, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to list channel providers: %w", err)
	}
	defer rows.Close()

	var providers []channel.PaymentChannel
	var lastErr error

	for rows.Next() {
		var config models.ChannelConfig
		if err := rows.Scan(
			&config.ID,
			&config.AppID,
			&config.Channel,
			&config.Config,
			&config.Status,
			&config.CreatedAt,
			&config.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan channel config: %w", err)
		}

		key := fmt.Sprintf("%s_%s", config.AppID, config.Channel)
		m.mu.RLock()
		provider, exists := m.providers[key]
		m.mu.RUnlock()
		if exists {
			providers = append(providers, provider)
			continue
		}

		provider, err := m.createProvider(config.Channel, config.Config)
		if err != nil {
			lastErr = err
			logger.Error("Failed to create provider for fallback: appID=%s, channel=%s, err=%v", config.AppID, config.Channel, err)
			continue
		}

		m.mu.Lock()
		m.providers[key] = provider
		m.mu.Unlock()

		providers = append(providers, provider)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate channel providers: %w", err)
	}

	if len(providers) == 0 && lastErr != nil {
		return nil, lastErr
	}

	return providers, nil
}

// createProvider 根据渠道类型创建 Provider
func (m *ChannelManager) createProvider(channelName, configJSON string) (channel.PaymentChannel, error) {
	switch channelName {
	case models.ChannelWechatNative:
		return m.createWechatNativeProvider(configJSON)
	case models.ChannelWechatJSAPI:
		return m.createWechatJSAPIProvider(configJSON)
	case models.ChannelWechatH5:
		return m.createWechatH5Provider(configJSON)
	case models.ChannelWechatApp:
		return m.createWechatAppProvider(configJSON)
	case models.ChannelAlipayQR:
		return m.createAlipayQRProvider(configJSON)
	case models.ChannelAlipayWap:
		return m.createAlipayWapProvider(configJSON)
	case models.ChannelAlipayApp:
		return m.createAlipayAppProvider(configJSON)
	case models.ChannelAlipayFace:
		return m.createAlipayFaceProvider(configJSON)
	default:
		return nil, errors.NewInvalidChannelError(channelName)
	}
}

// createWechatNativeProvider 创建微信 Native 支付 Provider
func (m *ChannelManager) createWechatNativeProvider(configJSON string) (channel.PaymentChannel, error) {
	var cfg wechat.Config
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal wechat config: %w", err)
	}

	provider, err := wechat.NewProvider(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create wechat native provider: %w", err)
	}

	return provider, nil
}

// createWechatJSAPIProvider 创建微信 JSAPI 支付 Provider
func (m *ChannelManager) createWechatJSAPIProvider(configJSON string) (channel.PaymentChannel, error) {
	var cfg wechat.Config
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal wechat config: %w", err)
	}

	provider, err := wechat.NewJSAPIProvider(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create wechat JSAPI provider: %w", err)
	}

	return provider, nil
}

// createWechatH5Provider 创建微信 H5 支付 Provider
func (m *ChannelManager) createWechatH5Provider(configJSON string) (channel.PaymentChannel, error) {
	var cfg wechat.Config
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal wechat config: %w", err)
	}

	provider, err := wechat.NewH5Provider(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create wechat H5 provider: %w", err)
	}

	return provider, nil
}

// createWechatAppProvider 创建微信 APP 支付 Provider
func (m *ChannelManager) createWechatAppProvider(configJSON string) (channel.PaymentChannel, error) {
	var cfg wechat.Config
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal wechat config: %w", err)
	}

	provider, err := wechat.NewAppProvider(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create wechat APP provider: %w", err)
	}

	return provider, nil
}

// createAlipayQRProvider 创建支付宝扫码支付 Provider
func (m *ChannelManager) createAlipayQRProvider(configJSON string) (channel.PaymentChannel, error) {
	var cfg alipay.Config
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alipay config: %w", err)
	}

	provider, err := alipay.NewQRProvider(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create alipay QR provider: %w", err)
	}

	return provider, nil
}

// createAlipayWapProvider 创建支付宝手机网站支付 Provider
func (m *ChannelManager) createAlipayWapProvider(configJSON string) (channel.PaymentChannel, error) {
	var cfg alipay.Config
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alipay config: %w", err)
	}

	provider, err := alipay.NewWapProvider(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create alipay Wap provider: %w", err)
	}

	return provider, nil
}

// createAlipayAppProvider 创建支付宝 APP 支付 Provider
func (m *ChannelManager) createAlipayAppProvider(configJSON string) (channel.PaymentChannel, error) {
	var cfg alipay.Config
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alipay config: %w", err)
	}

	provider, err := alipay.NewAppProvider(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create alipay APP provider: %w", err)
	}

	return provider, nil
}

// createAlipayFaceProvider 创建支付宝当面付 Provider
func (m *ChannelManager) createAlipayFaceProvider(configJSON string) (channel.PaymentChannel, error) {
	var cfg alipay.Config
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alipay config: %w", err)
	}

	provider, err := alipay.NewFaceProvider(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create alipay Face provider: %w", err)
	}

	return provider, nil
}

// ReloadProvider 重新加载 Provider（配置更新后调用）
func (m *ChannelManager) ReloadProvider(appID, channelName string) error {
	key := fmt.Sprintf("%s_%s", appID, channelName)

	m.mu.Lock()
	defer m.mu.Unlock()

	// 删除缓存
	if provider, exists := m.providers[key]; exists {
		if closer, ok := provider.(interface{ Close() error }); ok {
			closer.Close()
		}
		delete(m.providers, key)
	}

	logger.Info("Channel provider cache cleared: appID=%s, channel=%s", appID, channelName)

	return nil
}

// Close 关闭所有 Provider
func (m *ChannelManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, provider := range m.providers {
		if closer, ok := provider.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				logger.Error("Failed to close provider %s: %v", key, err)
			}
		}
	}

	m.providers = make(map[string]channel.PaymentChannel)

	logger.Info("All channel providers closed")

	return nil
}
