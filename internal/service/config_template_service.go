package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"gopay/pkg/errors"
)

// ConfigTemplateService 配置模板服务
type ConfigTemplateService struct {
	db *sql.DB
}

// NewConfigTemplateService 创建配置模板服务
func NewConfigTemplateService(db *sql.DB) *ConfigTemplateService {
	return &ConfigTemplateService{
		db: db,
	}
}

// ConfigTemplate 配置模板
type ConfigTemplate struct {
	ID            int64           `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Channel       string          `json:"channel"`
	ConfigSchema  json.RawMessage `json:"config_schema"`
	DefaultValues json.RawMessage `json:"default_values"`
	IsActive      bool            `json:"is_active"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// GetTemplateList 获取模板列表
func (s *ConfigTemplateService) GetTemplateList(ctx context.Context, channel string) ([]*ConfigTemplate, error) {
	query := `
		SELECT id, name, description, channel, config_schema, default_values, is_active, created_at, updated_at
		FROM config_templates
		WHERE is_active = true
	`
	args := []interface{}{}

	if channel != "" {
		query += " AND channel = $1"
		args = append(args, channel)
	}

	query += " ORDER BY channel, name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query templates: %w", err)
	}
	defer rows.Close()

	var templates []*ConfigTemplate
	for rows.Next() {
		var t ConfigTemplate
		err := rows.Scan(
			&t.ID,
			&t.Name,
			&t.Description,
			&t.Channel,
			&t.ConfigSchema,
			&t.DefaultValues,
			&t.IsActive,
			&t.CreatedAt,
			&t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}
		templates = append(templates, &t)
	}

	return templates, nil
}

// GetTemplateByID 根据 ID 获取模板
func (s *ConfigTemplateService) GetTemplateByID(ctx context.Context, id int64) (*ConfigTemplate, error) {
	var t ConfigTemplate
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, channel, config_schema, default_values, is_active, created_at, updated_at
		FROM config_templates
		WHERE id = $1
	`, id).Scan(
		&t.ID,
		&t.Name,
		&t.Description,
		&t.Channel,
		&t.ConfigSchema,
		&t.DefaultValues,
		&t.IsActive,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.NewBusinessError(
			errors.TypeNotFound,
			"TEMPLATE_NOT_FOUND",
			"模板不存在",
			errors.ErrNotFound,
			map[string]string{"template_id": fmt.Sprintf("%d", id)},
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return &t, nil
}

// GetTemplateByChannel 根据渠道获取模板
func (s *ConfigTemplateService) GetTemplateByChannel(ctx context.Context, channel string) (*ConfigTemplate, error) {
	var t ConfigTemplate
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, channel, config_schema, default_values, is_active, created_at, updated_at
		FROM config_templates
		WHERE channel = $1 AND is_active = true
		LIMIT 1
	`, channel).Scan(
		&t.ID,
		&t.Name,
		&t.Description,
		&t.Channel,
		&t.ConfigSchema,
		&t.DefaultValues,
		&t.IsActive,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.NewBusinessError(
			errors.TypeNotFound,
			"TEMPLATE_NOT_FOUND",
			"该渠道没有可用模板",
			errors.ErrNotFound,
			map[string]string{"channel": channel},
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return &t, nil
}

// QuickSetupApp 快速设置应用（创建应用 + 配置渠道）
func (s *ConfigTemplateService) QuickSetupApp(ctx context.Context, req *QuickSetupRequest) error {
	// 开启事务
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. 创建应用
	_, err = tx.ExecContext(ctx, `
		INSERT INTO apps (app_id, app_name, app_secret, callback_url, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'active', NOW(), NOW())
		ON CONFLICT (app_id) DO NOTHING
	`, req.AppID, req.AppName, req.AppSecret, req.CallbackURL)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	// 2. 为每个渠道创建配置
	for _, channelConfig := range req.Channels {
		// 获取模板
		var template ConfigTemplate
		err := tx.QueryRowContext(ctx, `
			SELECT id, name, description, channel, config_schema, default_values, is_active, created_at, updated_at
			FROM config_templates
			WHERE channel = $1 AND is_active = true
			LIMIT 1
		`, channelConfig.Channel).Scan(
			&template.ID,
			&template.Name,
			&template.Description,
			&template.Channel,
			&template.ConfigSchema,
			&template.DefaultValues,
			&template.IsActive,
			&template.CreatedAt,
			&template.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to get template for channel %s: %w", channelConfig.Channel, err)
		}

		// 合并默认值和用户提供的值
		var defaultValues map[string]interface{}
		if err := json.Unmarshal(template.DefaultValues, &defaultValues); err != nil {
			return fmt.Errorf("failed to parse default values: %w", err)
		}

		finalConfig := make(map[string]interface{})
		for k, v := range defaultValues {
			finalConfig[k] = v
		}
		for k, v := range channelConfig.Config {
			finalConfig[k] = v
		}

		// 序列化配置
		configJSON, err := json.Marshal(finalConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		encryptedConfig, err := encryptConfigJSON(string(configJSON))
		if err != nil {
			return fmt.Errorf("failed to encrypt channel config: %w", err)
		}

		// 插入渠道配置
		_, err = tx.ExecContext(ctx, `
			INSERT INTO channel_configs (app_id, channel, config, status, created_at, updated_at)
			VALUES ($1, $2, $3, 'active', NOW(), NOW())
			ON CONFLICT (app_id, channel) DO UPDATE SET
				config = EXCLUDED.config,
				status = EXCLUDED.status,
				updated_at = NOW()
		`, req.AppID, channelConfig.Channel, encryptedConfig)
		if err != nil {
			return fmt.Errorf("failed to create channel config: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// QuickSetupRequest 快速设置请求
type QuickSetupRequest struct {
	AppID       string                 `json:"app_id" binding:"required"`
	AppName     string                 `json:"app_name" binding:"required"`
	AppSecret   string                 `json:"app_secret" binding:"required"`
	CallbackURL string                 `json:"callback_url" binding:"required"`
	Channels    []ChannelConfigRequest `json:"channels" binding:"required,min=1"`
}

// ChannelConfigRequest 渠道配置请求
type ChannelConfigRequest struct {
	Channel string                 `json:"channel" binding:"required"`
	Config  map[string]interface{} `json:"config" binding:"required"`
}
