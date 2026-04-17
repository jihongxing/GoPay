package handler

import (
	"net/http"

	"gopay/pkg/logger"

	"github.com/gin-gonic/gin"
)

// StripeWebhook 处理 Stripe 支付回调
func StripeWebhook(c *gin.Context) {
	logger.Info("Received stripe webhook")

	// TODO: 实现 Stripe webhook 处理逻辑
	// 1. 读取请求体
	// 2. 验证 Stripe-Signature header
	// 3. 解析事件类型 (payment_intent.succeeded, charge.refunded 等)
	// 4. 更新订单状态
	// 5. 异步通知业务系统

	c.JSON(http.StatusOK, gin.H{"received": true})
}
