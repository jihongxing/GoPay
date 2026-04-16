package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gopay/internal/models"
	"gopay/pkg/errors"
	"gopay/pkg/logger"
)

// NotifyService 异步通知服务
type NotifyService struct {
	db           *sql.DB
	orderService *OrderService
	alertManager AlertManager
	httpClient   *http.Client
	workerPool   chan struct{} // 限制并发数量
}

// AlertManager 告警接口
type AlertManager interface {
	AlertNotifyFailed(order *models.Order)
}

// NewNotifyService 创建通知服务
func NewNotifyService(db *sql.DB, orderService *OrderService) *NotifyService {
	return &NotifyService{
		db:           db,
		orderService: orderService,
		httpClient: &http.Client{
			Timeout: 3 * time.Second, // 铁律二：3秒超时
		},
		workerPool: make(chan struct{}, 100), // 最多 100 个并发通知
	}
}

// SetAlertManager 设置告警管理器
func (s *NotifyService) SetAlertManager(am AlertManager) {
	s.alertManager = am
}

// NotifyRequest 通知请求
type NotifyRequest struct {
	OrderNo        string `json:"order_no"`
	OutTradeNo     string `json:"out_trade_no"`
	Amount         int64  `json:"amount"`
	Status         string `json:"status"`
	PaidAt         string `json:"paid_at"`
	Channel        string `json:"channel"`
	ChannelOrderNo string `json:"channel_order_no"`
}

// NotifyAsync 异步通知业务系统（在事务外调用）
func (s *NotifyService) NotifyAsync(order *models.Order) {
	// 获取 worker 令牌（限制并发）
	select {
	case s.workerPool <- struct{}{}:
		// 获取到令牌，在新的 goroutine 中执行通知
		go func() {
			defer func() { <-s.workerPool }() // 释放令牌

			ctx := context.Background()

			// 获取应用信息
			app, err := s.getApp(order.AppID)
			if err != nil {
				logger.Error("Failed to get app for notify: orderNo=%s, error=%v", order.OrderNo, err)
				return
			}

			// 执行通知（最多5次重试）
			s.notifyWithRetry(ctx, order, app.CallbackURL)
		}()
	default:
		// 工作池已满，记录日志但不阻塞
		logger.Error("Worker pool is full, cannot notify order: %s", order.OrderNo)
	}
}

// notifyWithRetry 带重试的通知（铁律二：最多5次重试，指数退避）
func (s *NotifyService) notifyWithRetry(ctx context.Context, order *models.Order, callbackURL string) {
	maxRetries := 5

	for i := 0; i < maxRetries; i++ {
		logger.Info("Notifying business system: orderNo=%s, attempt=%d/%d", order.OrderNo, i+1, maxRetries)

		// 执行通知
		success, statusCode, respBody, duration, err := s.doNotify(ctx, order, callbackURL)

		// 记录通知日志
		s.saveNotifyLog(&models.NotifyLog{
			OrderNo:        order.OrderNo,
			CallbackURL:    callbackURL,
			RequestBody:    s.buildNotifyRequestBody(order),
			ResponseStatus: statusCode,
			ResponseBody:   respBody,
			Success:        success,
			ErrorMsg:       s.getErrorMsg(err),
			DurationMs:     int(duration.Milliseconds()),
		})

		// 增加重试次数
		s.orderService.IncrementRetryCount(ctx, order.OrderNo)

		if success {
			// 通知成功，更新订单状态
			s.orderService.UpdateNotifyStatus(ctx, order.OrderNo, models.NotifyStatusNotified)
			logger.Info("Business system notified successfully: orderNo=%s", order.OrderNo)
			return
		}

		// 通知失败，判断是否继续重试
		if i < maxRetries-1 {
			// 指数退避：1s, 2s, 4s, 8s, 16s
			backoff := time.Duration(1<<uint(i)) * time.Second
			logger.Info("Notify failed, retry after %v: orderNo=%s, error=%v", backoff, order.OrderNo, err)
			time.Sleep(backoff)
		}
	}

	// 所有重试都失败，标记为通知失败
	s.orderService.UpdateNotifyStatus(ctx, order.OrderNo, models.NotifyStatusFailedNotify)
	logger.Error("All notify attempts failed: orderNo=%s", order.OrderNo)

	// TODO: 发送告警通知运维人员
	s.alertOps(order)
}

// doNotify 执行单次通知
func (s *NotifyService) doNotify(ctx context.Context, order *models.Order, callbackURL string) (success bool, statusCode int, respBody string, duration time.Duration, err error) {
	startTime := time.Now()

	// 构建请求体
	reqBody := s.buildNotifyRequest(order)
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return false, 0, "", time.Since(startTime), fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "POST", callbackURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return false, 0, "", time.Since(startTime), fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GoPay/1.0")

	// 发送请求（3秒超时）
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, 0, "", time.Since(startTime), fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, resp.StatusCode, "", time.Since(startTime), fmt.Errorf("failed to read response: %w", err)
	}

	duration = time.Since(startTime)
	respBody = string(respBodyBytes)
	statusCode = resp.StatusCode

	// 判断是否成功（HTTP 200 表示成功）
	success = (statusCode == 200)

	return success, statusCode, respBody, duration, nil
}

// buildNotifyRequest 构建通知请求
func (s *NotifyService) buildNotifyRequest(order *models.Order) *NotifyRequest {
	paidAtStr := ""
	if order.PaidAt != nil {
		paidAtStr = order.PaidAt.Format(time.RFC3339)
	}

	return &NotifyRequest{
		OrderNo:        order.OrderNo,
		OutTradeNo:     order.OutTradeNo,
		Amount:         order.Amount,
		Status:         order.Status,
		PaidAt:         paidAtStr,
		Channel:        order.Channel,
		ChannelOrderNo: order.ChannelOrderNo,
	}
}

// buildNotifyRequestBody 构建通知请求体（用于日志）
func (s *NotifyService) buildNotifyRequestBody(order *models.Order) string {
	req := s.buildNotifyRequest(order)
	bodyBytes, _ := json.Marshal(req)
	return string(bodyBytes)
}

// saveNotifyLog 保存通知日志
func (s *NotifyService) saveNotifyLog(log *models.NotifyLog) error {
	_, err := s.db.Exec(`
		INSERT INTO notify_logs (
			order_no, callback_url, request_body, response_status,
			response_body, success, error_msg, duration_ms
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		log.OrderNo, log.CallbackURL, log.RequestBody, log.ResponseStatus,
		log.ResponseBody, log.Success, log.ErrorMsg, log.DurationMs,
	)

	if err != nil {
		logger.Error("Failed to save notify log: orderNo=%s, error=%v", log.OrderNo, err)
	}

	return err
}

// getApp 获取应用信息
func (s *NotifyService) getApp(appID string) (*models.App, error) {
	app := &models.App{}
	err := s.db.QueryRow(`
		SELECT id, app_id, app_name, app_secret, callback_url, status, created_at, updated_at
		FROM apps
		WHERE app_id = $1 AND status = 'active'
	`, appID).Scan(
		&app.ID, &app.AppID, &app.AppName, &app.AppSecret,
		&app.CallbackURL, &app.Status, &app.CreatedAt, &app.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return app, nil
}

// getErrorMsg 获取错误信息
func (s *NotifyService) getErrorMsg(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// alertOps 告警运维人员
func (s *NotifyService) alertOps(order *models.Order) {
	// 记录告警日志
	logger.Error("ALERT: Notify failed after max retries, orderNo=%s, outTradeNo=%s, amount=%d",
		order.OrderNo, order.OutTradeNo, order.Amount)

	if s.alertManager != nil {
		s.alertManager.AlertNotifyFailed(order)
	}
}

// RetryNotify 手动重试通知（内部管理接口使用）
func (s *NotifyService) RetryNotify(ctx context.Context, orderNo string) error {
	logger.Info("Manual retry notify: orderNo=%s", orderNo)

	// 查询订单
	order, err := s.orderService.QueryOrder(ctx, orderNo)
	if err != nil {
		return err // QueryOrder 已经返回精确的错误类型
	}

	// 检查订单状态
	if order.Status != models.OrderStatusPaid {
		return errors.NewInvalidRequestError(
			"订单状态不正确，无法重试通知",
			map[string]string{
				"order_no":       orderNo,
				"current_status": order.Status,
				"required":       models.OrderStatusPaid,
			},
		)
	}

	// 获取应用信息
	app, err := s.getApp(order.AppID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.NewAppNotFoundError(order.AppID)
		}
		return fmt.Errorf("failed to get app: %w", err)
	}

	// 重置重试次数
	_, err = s.db.ExecContext(ctx, `
		UPDATE orders SET retry_count = 0, notify_status = $1, updated_at = NOW()
		WHERE order_no = $2
	`, models.NotifyStatusPending, orderNo)

	if err != nil {
		return fmt.Errorf("failed to reset retry count: %w", err)
	}

	// 异步执行通知
	go s.notifyWithRetry(ctx, order, app.CallbackURL)

	logger.Info("Manual retry notify started: orderNo=%s", orderNo)

	return nil
}

// ProcessPendingNotifies 处理待通知的订单（定时任务调用）
func (s *NotifyService) ProcessPendingNotifies(ctx context.Context) error {
	logger.Info("Processing pending notifies...")

	// 查询待通知的订单
	orders, err := s.orderService.ListPendingNotifyOrders(ctx, 100)
	if err != nil {
		return fmt.Errorf("failed to list pending notify orders: %w", err)
	}

	if len(orders) == 0 {
		logger.Info("No pending notify orders")
		return nil
	}

	logger.Info("Found %d pending notify orders", len(orders))

	// 异步处理每个订单
	for _, order := range orders {
		s.NotifyAsync(order)
	}

	return nil
}
