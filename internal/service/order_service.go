package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"gopay/internal/models"
	"gopay/pkg/channel"
	"gopay/pkg/errors"
	"gopay/pkg/logger"
)

// ChannelManagerInterface 渠道管理器接口
type ChannelManagerInterface interface {
	GetProvider(appID, channelType string) (channel.PaymentChannel, error)
}

// OrderService 订单服务
type OrderService struct {
	db             *sql.DB
	channelManager ChannelManagerInterface
	publicBaseURL  string
}

// NewOrderService 创建订单服务
func NewOrderService(db *sql.DB, channelManager ChannelManagerInterface) *OrderService {
	return &OrderService{
		db:             db,
		channelManager: channelManager,
		publicBaseURL:  "http://localhost:8080",
	}
}

// SetPublicBaseURL 设置对外访问地址
func (s *OrderService) SetPublicBaseURL(baseURL string) {
	if baseURL == "" {
		return
	}
	s.publicBaseURL = baseURL
}

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	AppID      string            `json:"app_id" binding:"required"`
	OutTradeNo string            `json:"out_trade_no" binding:"required"`
	Amount     int64             `json:"amount" binding:"required,gt=0"`
	Subject    string            `json:"subject" binding:"required"`
	Body       string            `json:"body"`
	Channel    string            `json:"channel" binding:"required"`
	NotifyURL  string            `json:"notify_url"`
	ExtraData  map[string]string `json:"extra_data"`
}

// CreateOrderResponse 创建订单响应
type CreateOrderResponse struct {
	OrderNo   string            `json:"order_no"`
	PayURL    string            `json:"pay_url"`
	QRCode    string            `json:"qr_code"`
	PrepayID  string            `json:"prepay_id"`
	ExtraData map[string]string `json:"extra_data"`
}

// CreateOrder 创建支付订单
func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	logger.Info("Creating order: appID=%s, outTradeNo=%s, amount=%d, channel=%s",
		req.AppID, req.OutTradeNo, req.Amount, req.Channel)

	// 1. 验证金额
	if req.Amount <= 0 {
		return nil, errors.NewInvalidAmountError(req.Amount)
	}

	// 2. 验证应用是否存在
	app, err := s.getApp(req.AppID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewAppNotFoundError(req.AppID)
		}
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	// 3. 检查订单是否已存在
	existingOrder, err := s.getOrderByOutTradeNo(app.ID, req.OutTradeNo)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing order: %w", err)
	}
	if existingOrder != nil {
		// 订单已存在
		logger.Info("Order already exists: orderNo=%s", existingOrder.OrderNo)
		return nil, errors.NewOrderExistsError(req.OutTradeNo)
	}

	// 4. 生成订单号
	orderNo := s.generateOrderNo()

	// 5. 获取支付渠道 Provider
	provider, err := s.channelManager.GetProvider(req.AppID, req.Channel)
	if err != nil {
		// 渠道管理器会返回精确的错误类型
		return nil, err
	}

	// 6. 调用支付渠道创建订单
	channelReq := &channel.CreateOrderRequest{
		OrderID:     orderNo,
		BizOrderNo:  req.OutTradeNo,
		Amount:      req.Amount,
		Subject:     req.Subject,
		Description: req.Body,
		NotifyURL:   s.buildWebhookURL(req.Channel), // GoPay 的 Webhook 地址
		ExtraData:   req.ExtraData,
	}

	channelResp, err := provider.CreateOrder(ctx, channelReq)
	if err != nil {
		return nil, errors.NewPaymentFailedError("支付渠道创建订单失败", err)
	}

	// 8. 保存订单到数据库
	order := &models.Order{
		OrderNo:        orderNo,
		AppID:          req.AppID,
		OutTradeNo:     req.OutTradeNo,
		Channel:        req.Channel,
		Amount:         req.Amount,
		Currency:       "CNY",
		Subject:        req.Subject,
		Body:           req.Body,
		Status:         models.OrderStatusPending,
		NotifyStatus:   models.NotifyStatusPending,
		RetryCount:     0,
		ChannelOrderNo: channelResp.PlatformTradeNo,
		PayURL:         channelResp.PayURL,
		ExpiresAt:      time.Now().Add(2 * time.Hour), // 订单2小时后过期
	}

	if err := s.saveOrder(order); err != nil {
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	logger.Info("Order created successfully: orderNo=%s, channelOrderNo=%s",
		orderNo, channelResp.PlatformTradeNo)

	return &CreateOrderResponse{
		OrderNo:   orderNo,
		PayURL:    channelResp.PayURL,
		QRCode:    channelResp.QRCode,
		PrepayID:  channelResp.PrepayID,
		ExtraData: channelResp.ExtraData,
	}, nil
}

// QueryOrder 查询订单
func (s *OrderService) QueryOrder(ctx context.Context, orderNo string) (*models.Order, error) {
	logger.Info("Querying order: orderNo=%s", orderNo)

	order, err := s.getOrderByOrderNo(orderNo)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewOrderNotFoundError(orderNo)
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return order, nil
}

// UpdateOrderStatus 更新订单状态（使用行锁）
func (s *OrderService) UpdateOrderStatus(ctx context.Context, orderNo string, status string, paidAt *time.Time, paidAmount int64) error {
	logger.Info("Updating order status: orderNo=%s, status=%s", orderNo, status)

	// 开始事务
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 使用 FOR UPDATE 锁定订单行
	var currentStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT status FROM orders WHERE order_no = $1 FOR UPDATE
	`, orderNo).Scan(&currentStatus)

	if err != nil {
		if err == sql.ErrNoRows {
			return errors.NewOrderNotFoundError(orderNo)
		}
		return fmt.Errorf("failed to lock order: %w", err)
	}

	// 检查状态是否可以更新（防止重复更新）
	if currentStatus == models.OrderStatusPaid {
		if status == models.OrderStatusRefunded {
			// 支持 paid -> refunded
		} else {
			logger.Info("Order already paid, skip update: orderNo=%s", orderNo)
			return errors.NewOrderPaidError(orderNo)
		}
	}

	if currentStatus == models.OrderStatusRefunded {
		if status == models.OrderStatusRefunded {
			return nil
		}
		logger.Info("Order already paid, skip update: orderNo=%s", orderNo)
		return errors.NewOrderPaidError(orderNo)
	}

	if currentStatus == models.OrderStatusClosed {
		logger.Info("Order already closed, skip update: orderNo=%s", orderNo)
		return errors.NewOrderClosedError(orderNo)
	}

	// 更新订单状态
	_, err = tx.ExecContext(ctx, `
		UPDATE orders
		SET status = $1, paid_at = $2, updated_at = NOW()
		WHERE order_no = $3
	`, status, paidAt, orderNo)

	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// 提交事务（铁律一：先提交事务，再异步通知）
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info("Order status updated successfully: orderNo=%s, status=%s", orderNo, status)

	return nil
}

// UpdateNotifyStatus 更新通知状态
func (s *OrderService) UpdateNotifyStatus(ctx context.Context, orderNo string, notifyStatus string) error {
	logger.Info("Updating notify status: orderNo=%s, notifyStatus=%s", orderNo, notifyStatus)

	_, err := s.db.ExecContext(ctx, `
		UPDATE orders
		SET notify_status = $1, notified_at = NOW(), updated_at = NOW()
		WHERE order_no = $2
	`, notifyStatus, orderNo)

	if err != nil {
		return fmt.Errorf("failed to update notify status: %w", err)
	}

	return nil
}

// IncrementRetryCount 增加重试次数
func (s *OrderService) IncrementRetryCount(ctx context.Context, orderNo string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE orders
		SET retry_count = retry_count + 1, updated_at = NOW()
		WHERE order_no = $1
	`, orderNo)

	if err != nil {
		return fmt.Errorf("failed to increment retry count: %w", err)
	}

	return nil
}

// ListPendingNotifyOrders 查询待通知的订单
func (s *OrderService) ListPendingNotifyOrders(ctx context.Context, limit int) ([]*models.Order, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, order_no, app_id, out_trade_no, channel, amount, currency,
		       subject, body, status, notify_status, retry_count,
		       channel_order_no, pay_url, paid_at, notified_at,
		       expires_at, created_at, updated_at
		FROM orders
		WHERE status = $1 AND notify_status = $2 AND retry_count < 5
		ORDER BY created_at ASC
		LIMIT $3
	`, models.OrderStatusPaid, models.NotifyStatusPending, limit)

	if err != nil {
		return nil, fmt.Errorf("failed to query pending notify orders: %w", err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		order := &models.Order{}
		err := rows.Scan(
			&order.ID, &order.OrderNo, &order.AppID, &order.OutTradeNo,
			&order.Channel, &order.Amount, &order.Currency,
			&order.Subject, &order.Body, &order.Status, &order.NotifyStatus,
			&order.RetryCount, &order.ChannelOrderNo, &order.PayURL,
			&order.PaidAt, &order.NotifiedAt, &order.ExpiresAt,
			&order.CreatedAt, &order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// getApp 获取应用信息
func (s *OrderService) getApp(appID string) (*models.App, error) {
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

// getOrderByOrderNo 根据订单号查询订单
func (s *OrderService) getOrderByOrderNo(orderNo string) (*models.Order, error) {
	order := &models.Order{}
	err := s.db.QueryRow(`
		SELECT id, order_no, app_id, out_trade_no, channel, amount, currency,
		       subject, body, status, notify_status, retry_count,
		       channel_order_no, pay_url, paid_at, notified_at,
		       expires_at, created_at, updated_at
		FROM orders
		WHERE order_no = $1
	`, orderNo).Scan(
		&order.ID, &order.OrderNo, &order.AppID, &order.OutTradeNo,
		&order.Channel, &order.Amount, &order.Currency,
		&order.Subject, &order.Body, &order.Status, &order.NotifyStatus,
		&order.RetryCount, &order.ChannelOrderNo, &order.PayURL,
		&order.PaidAt, &order.NotifiedAt, &order.ExpiresAt,
		&order.CreatedAt, &order.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return order, nil
}

// getOrderByOutTradeNo 根据业务订单号查询订单
func (s *OrderService) getOrderByOutTradeNo(appID int64, outTradeNo string) (*models.Order, error) {
	order := &models.Order{}
	err := s.db.QueryRow(`
		SELECT id, order_no, app_id, out_trade_no, channel, amount, currency,
		       subject, body, status, notify_status, retry_count,
		       channel_order_no, pay_url, paid_at, notified_at,
		       expires_at, created_at, updated_at
		FROM orders
		WHERE app_id = (SELECT app_id FROM apps WHERE id = $1) AND out_trade_no = $2
	`, appID, outTradeNo).Scan(
		&order.ID, &order.OrderNo, &order.AppID, &order.OutTradeNo,
		&order.Channel, &order.Amount, &order.Currency,
		&order.Subject, &order.Body, &order.Status, &order.NotifyStatus,
		&order.RetryCount, &order.ChannelOrderNo, &order.PayURL,
		&order.PaidAt, &order.NotifiedAt, &order.ExpiresAt,
		&order.CreatedAt, &order.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return order, nil
}

// saveOrder 保存订单
func (s *OrderService) saveOrder(order *models.Order) error {
	_, err := s.db.Exec(`
		INSERT INTO orders (
			order_no, app_id, out_trade_no, channel, amount, currency,
			subject, body, status, notify_status, retry_count,
			channel_order_no, pay_url, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`,
		order.OrderNo, order.AppID, order.OutTradeNo, order.Channel,
		order.Amount, order.Currency, order.Subject, order.Body,
		order.Status, order.NotifyStatus, order.RetryCount,
		order.ChannelOrderNo, order.PayURL, order.ExpiresAt,
	)

	return err
}

// QueryOrderByOutTradeNoGlobal 根据业务订单号全局查询订单（不需要 app_id）
func (s *OrderService) QueryOrderByOutTradeNoGlobal(ctx context.Context, outTradeNo string) (*models.Order, error) {
	logger.Info("Querying order globally by out_trade_no: %s", outTradeNo)

	order := &models.Order{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, order_no, app_id, out_trade_no, channel, amount, currency,
		       subject, body, status, notify_status, retry_count,
		       channel_order_no, pay_url, paid_at, notified_at,
		       expires_at, created_at, updated_at
		FROM orders
		WHERE out_trade_no = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, outTradeNo).Scan(
		&order.ID, &order.OrderNo, &order.AppID, &order.OutTradeNo,
		&order.Channel, &order.Amount, &order.Currency,
		&order.Subject, &order.Body, &order.Status, &order.NotifyStatus,
		&order.RetryCount, &order.ChannelOrderNo, &order.PayURL,
		&order.PaidAt, &order.NotifiedAt, &order.ExpiresAt,
		&order.CreatedAt, &order.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewOrderNotFoundError(outTradeNo)
		}
		return nil, fmt.Errorf("failed to query order globally: %w", err)
	}

	return order, nil
}

// QueryOrderByOutTradeNo 根据业务订单号查询订单
func (s *OrderService) QueryOrderByOutTradeNo(ctx context.Context, appID, outTradeNo string) (*models.Order, error) {
	logger.Info("Querying order by out_trade_no: appID=%s, outTradeNo=%s", appID, outTradeNo)

	order := &models.Order{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, order_no, app_id, out_trade_no, channel, amount, currency,
		       subject, body, status, notify_status, retry_count,
		       channel_order_no, pay_url, paid_at, notified_at,
		       expires_at, created_at, updated_at
		FROM orders
		WHERE app_id = $1 AND out_trade_no = $2
	`, appID, outTradeNo).Scan(
		&order.ID, &order.OrderNo, &order.AppID, &order.OutTradeNo,
		&order.Channel, &order.Amount, &order.Currency,
		&order.Subject, &order.Body, &order.Status, &order.NotifyStatus,
		&order.RetryCount, &order.ChannelOrderNo, &order.PayURL,
		&order.PaidAt, &order.NotifiedAt, &order.ExpiresAt,
		&order.CreatedAt, &order.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewOrderNotFoundError(outTradeNo)
		}
		return nil, fmt.Errorf("failed to query order by out_trade_no: %w", err)
	}

	return order, nil
}

// ListFailedOrders 查询失败的订单
func (s *OrderService) ListFailedOrders(ctx context.Context, limit int) ([]*models.Order, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, order_no, app_id, out_trade_no, channel, amount, currency,
		       subject, body, status, notify_status, retry_count,
		       channel_order_no, pay_url, paid_at, notified_at,
		       expires_at, created_at, updated_at
		FROM orders
		WHERE notify_status = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, models.NotifyStatusFailedNotify, limit)

	if err != nil {
		return nil, fmt.Errorf("failed to query failed orders: %w", err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		order := &models.Order{}
		err := rows.Scan(
			&order.ID, &order.OrderNo, &order.AppID, &order.OutTradeNo,
			&order.Channel, &order.Amount, &order.Currency,
			&order.Subject, &order.Body, &order.Status, &order.NotifyStatus,
			&order.RetryCount, &order.ChannelOrderNo, &order.PayURL,
			&order.PaidAt, &order.NotifiedAt, &order.ExpiresAt,
			&order.CreatedAt, &order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// generateOrderNo 生成订单号
func (s *OrderService) generateOrderNo() string {
	// 使用时间戳 + 随机字节确保唯一性
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomStr := hex.EncodeToString(randomBytes)

	return fmt.Sprintf("ORD_%s_%s",
		time.Now().Format("20060102150405"),
		randomStr)
}

// buildWebhookURL 构建 Webhook 回调地址
func (s *OrderService) buildWebhookURL(channel string) string {
	baseURL := s.publicBaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return fmt.Sprintf("%s/api/v1/webhook/%s", strings.TrimRight(baseURL, "/"), channel)
}
