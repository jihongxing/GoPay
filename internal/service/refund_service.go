package service

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"gopay/internal/models"
	"gopay/pkg/channel"
	"gopay/pkg/errors"
)

type refundProvider interface {
	Refund(ctx context.Context, req *channel.RefundRequest) (*channel.RefundResponse, error)
	QueryRefund(ctx context.Context, req *channel.RefundRequest) (*channel.RefundResponse, error)
}

// RefundService 退款服务
type RefundService struct {
	db             *sql.DB
	orderService   *OrderService
	channelManager ChannelManagerInterface
}

// NewRefundService 创建退款服务
func NewRefundService(db *sql.DB, orderService *OrderService, channelManager ChannelManagerInterface) *RefundService {
	return &RefundService{
		db:             db,
		orderService:   orderService,
		channelManager: channelManager,
	}
}

// RefundRequest 退款请求
type RefundRequest struct {
	OrderNo string `json:"order_no" binding:"required"`
	Amount  int64  `json:"amount"`
	Reason  string `json:"reason"`
}

// RefundResponse 退款响应
type RefundResponse struct {
	RefundNo         string     `json:"refund_no"`
	OrderNo          string     `json:"order_no"`
	PlatformTradeNo  string     `json:"platform_trade_no"`
	PlatformRefundNo string     `json:"platform_refund_no"`
	Status           string     `json:"status"`
	Amount           int64      `json:"amount"`
	RefundedAt       *time.Time `json:"refunded_at,omitempty"`
}

// Refund 发起退款
func (s *RefundService) Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	if s.db == nil || s.orderService == nil || s.channelManager == nil {
		return nil, fmt.Errorf("refund service is not initialized")
	}
	if req == nil || req.OrderNo == "" {
		return nil, errors.NewInvalidRequestError("订单号不能为空", nil)
	}

	order, err := s.orderService.QueryOrder(ctx, req.OrderNo)
	if err != nil {
		return nil, err
	}
	if order.Status != models.OrderStatusPaid {
		return nil, errors.NewInvalidRequestError("只能退款已支付订单", map[string]string{
			"order_no": req.OrderNo,
			"status":   order.Status,
		})
	}

	refundAmount := req.Amount
	if refundAmount <= 0 {
		refundAmount = order.Amount
	}
	if refundAmount > order.Amount {
		return nil, errors.NewInvalidAmountError(refundAmount)
	}

	provider, err := s.channelManager.GetProvider(order.AppID, order.Channel)
	if err != nil {
		return nil, err
	}
	refundCapable, ok := provider.(refundProvider)
	if !ok {
		return nil, fmt.Errorf("channel %s does not support refund", order.Channel)
	}

	refundNo := s.generateRefundNo()
	refundResp, err := refundCapable.Refund(ctx, &channel.RefundRequest{
		OrderID:    order.OrderNo,
		BizOrderNo: order.OutTradeNo,
		PlatformNo: order.ChannelOrderNo,
		RefundNo:   refundNo,
		Amount:     refundAmount,
		Reason:     req.Reason,
	})
	if err != nil {
		return nil, err
	}

	if err := s.saveRefundTransaction(ctx, order, refundResp); err != nil {
		return nil, err
	}
	if err := s.updateRefundStatus(ctx, order.OrderNo, refundResp.Status); err != nil {
		return nil, err
	}

	return &RefundResponse{
		RefundNo:         refundResp.RefundNo,
		OrderNo:          order.OrderNo,
		PlatformTradeNo:  refundResp.PlatformTradeNo,
		PlatformRefundNo: refundResp.PlatformRefundNo,
		Status:           string(refundResp.Status),
		Amount:           refundResp.Amount,
		RefundedAt:       refundResp.RefundedAt,
	}, nil
}

// QueryRefund 查询退款状态
func (s *RefundService) QueryRefund(ctx context.Context, orderNo, refundNo string) (*RefundResponse, error) {
	if s.db == nil || s.orderService == nil || s.channelManager == nil {
		return nil, fmt.Errorf("refund service is not initialized")
	}
	if orderNo == "" || refundNo == "" {
		return nil, errors.NewInvalidRequestError("订单号和退款单号不能为空", nil)
	}

	order, err := s.orderService.QueryOrder(ctx, orderNo)
	if err != nil {
		return nil, err
	}

	provider, err := s.channelManager.GetProvider(order.AppID, order.Channel)
	if err != nil {
		return nil, err
	}
	refundCapable, ok := provider.(refundProvider)
	if !ok {
		return nil, fmt.Errorf("channel %s does not support refund", order.Channel)
	}

	resp, err := refundCapable.QueryRefund(ctx, &channel.RefundRequest{
		OrderID:    order.OrderNo,
		BizOrderNo: order.OutTradeNo,
		PlatformNo: order.ChannelOrderNo,
		RefundNo:   refundNo,
	})
	if err != nil {
		return nil, err
	}

	return &RefundResponse{
		RefundNo:         resp.RefundNo,
		OrderNo:          order.OrderNo,
		PlatformTradeNo:  resp.PlatformTradeNo,
		PlatformRefundNo: resp.PlatformRefundNo,
		Status:           string(resp.Status),
		Amount:           resp.Amount,
		RefundedAt:       resp.RefundedAt,
	}, nil
}

func (s *RefundService) saveRefundTransaction(ctx context.Context, order *models.Order, refundResp *channel.RefundResponse) error {
	if refundResp == nil {
		return fmt.Errorf("refund response is required")
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO transactions (
			transaction_no, order_no, channel, channel_order_no,
			type, amount, status, raw_request, raw_response
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, s.generateTransactionNo(), order.OrderNo, order.Channel, order.ChannelOrderNo,
		models.TransactionTypeRefund, refundResp.Amount, string(refundResp.Status), "", "")
	if err != nil {
		return fmt.Errorf("save refund transaction failed: %w", err)
	}

	return nil
}

func (s *RefundService) updateRefundStatus(ctx context.Context, orderNo string, status channel.RefundStatus) error {
	if status != channel.RefundStatusSuccess {
		return nil
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE orders
		SET status = $1, updated_at = NOW()
		WHERE order_no = $2
	`, models.OrderStatusRefunded, orderNo)
	if err != nil {
		return fmt.Errorf("update refund status failed: %w", err)
	}

	return nil
}

func (s *RefundService) generateRefundNo() string {
	return fmt.Sprintf("RFD_%s_%06d", time.Now().Format("20060102150405"), rand.Intn(1000000))
}

func (s *RefundService) generateTransactionNo() string {
	return fmt.Sprintf("TRX_%s_%06d", time.Now().Format("20060102150405"), rand.Intn(1000000))
}
