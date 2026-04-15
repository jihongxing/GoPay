package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"gopay/pkg/logger"
)

// AuditLog 审计日志
type AuditLog struct {
	ID         int64     `json:"id" db:"id"`
	UserID     string    `json:"user_id" db:"user_id"`
	Action     string    `json:"action" db:"action"`
	Resource   string    `json:"resource" db:"resource"`
	ResourceID string    `json:"resource_id" db:"resource_id"`
	IP         string    `json:"ip" db:"ip"`
	UserAgent  string    `json:"user_agent" db:"user_agent"`
	Details    string    `json:"details" db:"details"` // JSON 格式
	Status     string    `json:"status" db:"status"`   // success, failed
	ErrorMsg   string    `json:"error_msg" db:"error_msg"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// AuditLogger 审计日志记录器
type AuditLogger struct {
	db *sql.DB
}

// NewAuditLogger 创建审计日志记录器
func NewAuditLogger(db *sql.DB) *AuditLogger {
	return &AuditLogger{
		db: db,
	}
}

// LogOperation 记录操作
func (al *AuditLogger) LogOperation(ctx context.Context, userID, action, resource, resourceID, ip, userAgent string, details map[string]interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		logger.Error("Failed to marshal audit details: %v", err)
		detailsJSON = []byte("{}")
	}

	_, err = al.db.ExecContext(ctx, `
		INSERT INTO audit_logs (
			user_id, action, resource, resource_id, ip, user_agent, details, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`, userID, action, resource, resourceID, ip, userAgent, string(detailsJSON), "success")

	if err != nil {
		logger.Error("Failed to save audit log: %v", err)
		return err
	}

	logger.Info("Audit log saved: user=%s, action=%s, resource=%s, resourceID=%s", userID, action, resource, resourceID)
	return nil
}

// LogFailedOperation 记录失败的操作
func (al *AuditLogger) LogFailedOperation(ctx context.Context, userID, action, resource, resourceID, ip, userAgent, errorMsg string, details map[string]interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		logger.Error("Failed to marshal audit details: %v", err)
		detailsJSON = []byte("{}")
	}

	_, err = al.db.ExecContext(ctx, `
		INSERT INTO audit_logs (
			user_id, action, resource, resource_id, ip, user_agent, details, status, error_msg, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`, userID, action, resource, resourceID, ip, userAgent, string(detailsJSON), "failed", errorMsg)

	if err != nil {
		logger.Error("Failed to save failed audit log: %v", err)
		return err
	}

	logger.Info("Failed audit log saved: user=%s, action=%s, resource=%s, error=%s", userID, action, resource, errorMsg)
	return nil
}

// LogSensitiveOperation 记录敏感操作
func (al *AuditLogger) LogSensitiveOperation(ctx context.Context, userID, action, resource, resourceID, ip string, details map[string]interface{}) error {
	// 敏感操作需要额外记录和告警
	logger.Error("SENSITIVE OPERATION: user=%s, action=%s, resource=%s, resourceID=%s, ip=%s",
		userID, action, resource, resourceID, ip)

	// 记录到审计日志
	err := al.LogOperation(ctx, userID, action, resource, resourceID, ip, "", details)
	if err != nil {
		return err
	}

	// 发送告警（可选）
	// alertManager.AlertSensitiveOperation(userID, action, resource, resourceID, ip)

	return nil
}

// QueryAuditLogs 查询审计日志
func (al *AuditLogger) QueryAuditLogs(ctx context.Context, userID, action, resource string, startTime, endTime time.Time, limit int) ([]*AuditLog, error) {
	query := `
		SELECT id, user_id, action, resource, resource_id, ip, user_agent, details, status, error_msg, created_at
		FROM audit_logs
		WHERE created_at BETWEEN $1 AND $2
	`
	args := []interface{}{startTime, endTime}
	argIndex := 3

	if userID != "" {
		query += " AND user_id = $" + string(rune(argIndex))
		args = append(args, userID)
		argIndex++
	}

	if action != "" {
		query += " AND action = $" + string(rune(argIndex))
		args = append(args, action)
		argIndex++
	}

	if resource != "" {
		query += " AND resource = $" + string(rune(argIndex))
		args = append(args, resource)
		argIndex++
	}

	query += " ORDER BY created_at DESC LIMIT $" + string(rune(argIndex))
	args = append(args, limit)

	rows, err := al.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*AuditLog
	for rows.Next() {
		log := &AuditLog{}
		err := rows.Scan(
			&log.ID, &log.UserID, &log.Action, &log.Resource, &log.ResourceID,
			&log.IP, &log.UserAgent, &log.Details, &log.Status, &log.ErrorMsg, &log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// InitAuditLogTable 初始化审计日志表
func InitAuditLogTable(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS audit_logs (
		id SERIAL PRIMARY KEY,
		user_id VARCHAR(100) NOT NULL,
		action VARCHAR(100) NOT NULL,
		resource VARCHAR(100) NOT NULL,
		resource_id VARCHAR(100),
		ip VARCHAR(50),
		user_agent VARCHAR(500),
		details TEXT,
		status VARCHAR(20) NOT NULL,
		error_msg TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
	`

	_, err := db.Exec(schema)
	return err
}
