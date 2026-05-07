package handler

import (
	"io"
	"net/http"
	"time"

	"gopay/internal/models"
	"gopay/pkg/channel"
	"gopay/pkg/logger"

	"github.com/gin-gonic/gin"
)

// StripeWebhook 处理 Stripe 支付回调
func StripeWebhook(c *gin.Context) {
	logger.Info("Received stripe webhook")

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error("Failed to read stripe webhook body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	headers := make(map[string]string)
	for key := range c.Request.Header {
		headers[key] = c.Request.Header.Get(key)
	}

	webhookReq := &channel.WebhookRequest{
		RawBody: body,
		Headers: headers,
	}

	// Stripe webhook 需要通过 metadata 中的 order_id 找到订单
	// 先尝试用任意一个 stripe provider 来验签和解析
	lister, ok := channelManager.(webhookChannelProviderLister)
	if !ok {
		logger.Error("Stripe webhook: channel manager does not support listing providers")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	providers, err := lister.ListProvidersByChannelPrefix("stripe")
	if err != nil || len(providers) == 0 {
		logger.Error("Stripe webhook: no stripe provider configured: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stripe not configured"})
		return
	}

	var webhookResp *channel.WebhookResponse
	var lastErr error
	for _, provider := range providers {
		webhookResp, lastErr = provider.HandleWebhook(c.Request.Context(), webhookReq)
		if lastErr == nil && webhookResp != nil && webhookResp.Success {
			break
		}
	}

	if lastErr != nil || webhookResp == nil || !webhookResp.Success {
		logger.Error("Failed to handle stripe webhook: %v", lastErr)
		c.JSON(http.StatusBadRequest, gin.H{"error": "webhook processing failed"})
		return
	}

	// 如果没有 order_id，说明是不需要处理的事件类型
	if webhookResp.OrderID == "" {
		c.JSON(http.StatusOK, gin.H{"received": true})
		return
	}

	// 查询订单
	order, err := orderService.QueryOrder(c.Request.Context(), webhookResp.OrderID)
	if err != nil {
		logger.Error("Stripe webhook: order not found: %s, err: %v", webhookResp.OrderID, err)
		c.JSON(http.StatusOK, gin.H{"received": true})
		return
	}

	// 更新订单状态
	if webhookResp.Status == channel.OrderStatusPaid {
		paidAt := webhookResp.PaidAt
		if paidAt.IsZero() {
			paidAt = time.Now()
		}
		err = orderService.UpdateOrderStatus(
			c.Request.Context(),
			order.OrderNo,
			models.OrderStatusPaid,
			&paidAt,
			webhookResp.PaidAmount,
		)
		if err != nil {
			logger.Error("Failed to update stripe order status: %v", err)
			c.JSON(http.StatusOK, gin.H{"received": true})
			return
		}
		notifyService.NotifyAsync(order)
	}

	if webhookResp.Status == channel.OrderStatusRefund {
		refundAt := webhookResp.PaidAt
		if refundAt.IsZero() {
			refundAt = time.Now()
		}
		err = orderService.UpdateOrderStatus(
			c.Request.Context(),
			order.OrderNo,
			models.OrderStatusRefunded,
			&refundAt,
			webhookResp.PaidAmount,
		)
		if err != nil {
			logger.Error("Failed to update stripe refund status: %v", err)
			c.JSON(http.StatusOK, gin.H{"received": true})
			return
		}
		notifyService.NotifyRefundAsync(order, webhookResp)
	}

	logger.Info("Stripe webhook processed: orderNo=%s", order.OrderNo)
	c.JSON(http.StatusOK, gin.H{"received": true})
}
