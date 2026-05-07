package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"gopay/internal/models"
)

// ConfigService 配置管理服务
type ConfigService struct {
	db *sql.DB
}

// NewConfigService 创建配置管理服务
func NewConfigService(db *sql.DB) *ConfigService {
	return &ConfigService{db: db}
}

// ========== App 管理 ==========

// ListApps 获取应用列表
func (s *ConfigService) ListApps(ctx context.Context, page, pageSize int, status string) ([]models.App, int, error) {
	offset := (page - 1) * pageSize

	// 构建查询条件
	where := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if status != "" {
		where += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	// 查询总数
	var total int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM apps "+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count apps failed: %w", err)
	}

	// 查询数据
	args = append(args, pageSize, offset)
	query := fmt.Sprintf(`
		SELECT id, app_id, app_name, app_secret, callback_url, status, created_at, updated_at
		FROM apps %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIndex, argIndex+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query apps failed: %w", err)
	}
	defer rows.Close()

	apps := []models.App{}
	for rows.Next() {
		var app models.App
		err := rows.Scan(&app.ID, &app.AppID, &app.AppName, &app.AppSecret,
			&app.CallbackURL, &app.Status, &app.CreatedAt, &app.UpdatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("scan app failed: %w", err)
		}
		apps = append(apps, app)
	}

	return apps, total, nil
}

// GetApp 获取应用详情
func (s *ConfigService) GetApp(ctx context.Context, appID string) (*models.App, error) {
	var app models.App
	err := s.db.QueryRowContext(ctx, `
		SELECT id, app_id, app_name, app_secret, callback_url, status, created_at, updated_at
		FROM apps
		WHERE app_id = $1
	`, appID).Scan(&app.ID, &app.AppID, &app.AppName, &app.AppSecret,
		&app.CallbackURL, &app.Status, &app.CreatedAt, &app.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("app not found: %s", appID)
	}
	if err != nil {
		return nil, fmt.Errorf("get app failed: %w", err)
	}

	return &app, nil
}

// CreateApp 创建应用
func (s *ConfigService) CreateApp(ctx context.Context, app *models.App, operator, ip, userAgent string) error {
	// 验证
	if err := app.Validate(); err != nil {
		return err
	}

	// 开启事务
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}
	defer tx.Rollback()

	// 插入应用
	err = tx.QueryRowContext(ctx, `
		INSERT INTO apps (app_id, app_name, app_secret, callback_url, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`, app.AppID, app.AppName, app.AppSecret, app.CallbackURL, app.Status).
		Scan(&app.ID, &app.CreatedAt, &app.UpdatedAt)

	if err != nil {
		return fmt.Errorf("insert app failed: %w", err)
	}

	// 记录审计日志
	newValue, _ := json.Marshal(app)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO config_audit_logs (operator, action, resource_type, resource_id, new_value, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, operator, models.AuditActionCreate, models.AuditResourceApp, app.AppID, string(newValue), ip, userAgent)

	if err != nil {
		return fmt.Errorf("insert audit log failed: %w", err)
	}

	return tx.Commit()
}

// UpdateApp 更新应用
func (s *ConfigService) UpdateApp(ctx context.Context, appID string, updates map[string]interface{}, operator, ip, userAgent string) error {
	// 开启事务
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}
	defer tx.Rollback()

	// 获取旧值
	oldApp, err := s.GetApp(ctx, appID)
	if err != nil {
		return err
	}

	// 构建更新语句
	query := "UPDATE apps SET "
	args := []interface{}{}
	argIndex := 1

	for key, value := range updates {
		if argIndex > 1 {
			query += ", "
		}
		query += fmt.Sprintf("%s = $%d", key, argIndex)
		args = append(args, value)
		argIndex++
	}

	query += fmt.Sprintf(" WHERE app_id = $%d", argIndex)
	args = append(args, appID)

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update app failed: %w", err)
	}

	// 记录审计日志
	oldValue, _ := json.Marshal(oldApp)
	newValue, _ := json.Marshal(updates)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO config_audit_logs (operator, action, resource_type, resource_id, old_value, new_value, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, operator, models.AuditActionUpdate, models.AuditResourceApp, appID, string(oldValue), string(newValue), ip, userAgent)

	if err != nil {
		return fmt.Errorf("insert audit log failed: %w", err)
	}

	return tx.Commit()
}

// DeleteApp 删除应用（软删除，改为 disabled 状态）
func (s *ConfigService) DeleteApp(ctx context.Context, appID string, operator, ip, userAgent string) error {
	return s.UpdateApp(ctx, appID, map[string]interface{}{"status": "disabled"}, operator, ip, userAgent)
}

// ========== 渠道配置管理 ==========

// ListChannelConfigs 获取渠道配置列表
func (s *ConfigService) ListChannelConfigs(ctx context.Context, appID string) ([]models.ChannelConfig, error) {
	query := `
		SELECT id, app_id, channel, config, status, created_at, updated_at
		FROM channel_configs
		WHERE app_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, appID)
	if err != nil {
		return nil, fmt.Errorf("query channel configs failed: %w", err)
	}
	defer rows.Close()

	configs := []models.ChannelConfig{}
	for rows.Next() {
		var config models.ChannelConfig
		err := rows.Scan(&config.ID, &config.AppID, &config.Channel, &config.Config,
			&config.Status, &config.CreatedAt, &config.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan channel config failed: %w", err)
		}
		configs = append(configs, config)
	}

	return configs, nil
}

// GetChannelConfig 获取渠道配置详情
func (s *ConfigService) GetChannelConfig(ctx context.Context, id int64) (*models.ChannelConfig, error) {
	var config models.ChannelConfig
	err := s.db.QueryRowContext(ctx, `
		SELECT id, app_id, channel, config, status, created_at, updated_at
		FROM channel_configs
		WHERE id = $1
	`, id).Scan(&config.ID, &config.AppID, &config.Channel, &config.Config,
		&config.Status, &config.CreatedAt, &config.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("channel config not found: %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get channel config failed: %w", err)
	}

	return &config, nil
}

// CreateChannelConfig 创建渠道配置
func (s *ConfigService) CreateChannelConfig(ctx context.Context, config *models.ChannelConfig, operator, ip, userAgent string) error {
	// 验证
	if err := config.Validate(); err != nil {
		return err
	}

	encryptedConfig, err := encryptConfigJSON(config.Config)
	if err != nil {
		return fmt.Errorf("encrypt channel config failed: %w", err)
	}

	// 开启事务
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}
	defer tx.Rollback()

	// 插入配置
	err = tx.QueryRowContext(ctx, `
		INSERT INTO channel_configs (app_id, channel, config, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`, config.AppID, config.Channel, encryptedConfig, config.Status).
		Scan(&config.ID, &config.CreatedAt, &config.UpdatedAt)

	if err != nil {
		return fmt.Errorf("insert channel config failed: %w", err)
	}
	config.Config = encryptedConfig

	// 记录审计日志
	auditValue := *config
	auditValue.Config = MaskSensitiveConfigJSON(config.Config)
	newValue, _ := json.Marshal(auditValue)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO config_audit_logs (operator, action, resource_type, resource_id, new_value, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, operator, models.AuditActionCreate, models.AuditResourceChannelConfig, fmt.Sprintf("%d", config.ID), string(newValue), ip, userAgent)

	if err != nil {
		return fmt.Errorf("insert audit log failed: %w", err)
	}

	return tx.Commit()
}

// UpdateChannelConfig 更新渠道配置
func (s *ConfigService) UpdateChannelConfig(ctx context.Context, id int64, updates map[string]interface{}, operator, ip, userAgent string) error {
	// 开启事务
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}
	defer tx.Rollback()

	// 获取旧值
	oldConfig, err := s.GetChannelConfig(ctx, id)
	if err != nil {
		return err
	}

	if rawConfig, ok := updates["config"].(string); ok {
		encryptedConfig, err := encryptConfigJSON(rawConfig)
		if err != nil {
			return fmt.Errorf("encrypt channel config failed: %w", err)
		}
		updates["config"] = encryptedConfig
	}

	// 构建更新语句
	query := "UPDATE channel_configs SET "
	args := []interface{}{}
	argIndex := 1

	for key, value := range updates {
		if argIndex > 1 {
			query += ", "
		}
		query += fmt.Sprintf("%s = $%d", key, argIndex)
		args = append(args, value)
		argIndex++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIndex)
	args = append(args, id)

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update channel config failed: %w", err)
	}

	// 记录审计日志
	oldAuditConfig := *oldConfig
	oldAuditConfig.Config = MaskSensitiveConfigJSON(oldConfig.Config)
	oldValue, _ := json.Marshal(oldAuditConfig)

	auditUpdates := make(map[string]interface{}, len(updates))
	for key, value := range updates {
		auditUpdates[key] = value
	}
	if rawConfig, ok := auditUpdates["config"].(string); ok {
		auditUpdates["config"] = MaskSensitiveConfigJSON(rawConfig)
	}
	newValue, _ := json.Marshal(auditUpdates)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO config_audit_logs (operator, action, resource_type, resource_id, old_value, new_value, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, operator, models.AuditActionUpdate, models.AuditResourceChannelConfig, fmt.Sprintf("%d", id), string(oldValue), string(newValue), ip, userAgent)

	if err != nil {
		return fmt.Errorf("insert audit log failed: %w", err)
	}

	return tx.Commit()
}

// DeleteChannelConfig 删除渠道配置（软删除）
func (s *ConfigService) DeleteChannelConfig(ctx context.Context, id int64, operator, ip, userAgent string) error {
	return s.UpdateChannelConfig(ctx, id, map[string]interface{}{"status": "disabled"}, operator, ip, userAgent)
}

// ========== 审计日志查询 ==========

// ListAuditLogs 获取审计日志列表
func (s *ConfigService) ListAuditLogs(ctx context.Context, page, pageSize int, resourceType, resourceID, operator string, startDate, endDate time.Time) ([]models.ConfigAuditLog, int, error) {
	offset := (page - 1) * pageSize

	// 构建查询条件
	where := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if resourceType != "" {
		where += fmt.Sprintf(" AND resource_type = $%d", argIndex)
		args = append(args, resourceType)
		argIndex++
	}

	if resourceID != "" {
		where += fmt.Sprintf(" AND resource_id = $%d", argIndex)
		args = append(args, resourceID)
		argIndex++
	}

	if operator != "" {
		where += fmt.Sprintf(" AND operator = $%d", argIndex)
		args = append(args, operator)
		argIndex++
	}

	if !startDate.IsZero() {
		where += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, startDate)
		argIndex++
	}

	if !endDate.IsZero() {
		where += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, endDate)
		argIndex++
	}

	// 查询总数
	var total int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM config_audit_logs "+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count audit logs failed: %w", err)
	}

	// 查询数据
	args = append(args, pageSize, offset)
	query := fmt.Sprintf(`
		SELECT id, operator, action, resource_type, resource_id, old_value, new_value, ip_address, user_agent, created_at
		FROM config_audit_logs %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIndex, argIndex+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query audit logs failed: %w", err)
	}
	defer rows.Close()

	logs := []models.ConfigAuditLog{}
	for rows.Next() {
		var log models.ConfigAuditLog
		var oldValue, newValue sql.NullString

		err := rows.Scan(&log.ID, &log.Operator, &log.Action, &log.ResourceType, &log.ResourceID,
			&oldValue, &newValue, &log.IPAddress, &log.UserAgent, &log.CreatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("scan audit log failed: %w", err)
		}

		if oldValue.Valid {
			log.OldValue = oldValue.String
		}
		if newValue.Valid {
			log.NewValue = newValue.String
		}

		logs = append(logs, log)
	}

	return logs, total, nil
}
