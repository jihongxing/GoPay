package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gopay/internal/models"
	"gopay/internal/service"
	"gopay/pkg/channel"
	"gopay/pkg/logger"

	"github.com/gin-gonic/gin"
)

var (
	channelManager webhookChannelManager
	notifyService  *service.NotifyService
)

type webhookChannelManager interface {
	GetProvider(appID, channelName string) (channel.PaymentChannel, error)
}

type webhookChannelProviderLister interface {
	ListProvidersByChannelPrefix(prefix string) ([]channel.PaymentChannel, error)
}

// InitWebhookServices 初始化 Webhook 相关服务
func InitWebhookServices(cm webhookChannelManager, ns *service.NotifyService) {
	channelManager = cm
	notifyService = ns
}

// WechatWebhook 处理微信支付回调
func WechatWebhook(c *gin.Context) {
	logger.Info("Received wechat webhook")

	// 读取原始请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error("Failed to read webhook body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "invalid request"})
		return
	}

	// 获取请求头
	headers := make(map[string]string)
	for key := range c.Request.Header {
		headers[key] = c.Request.Header.Get(key)
	}

	// 构建 Webhook 请求
	webhookReq := &channel.WebhookRequest{
		RawBody: body,
		Headers: headers,
	}

	// 从 webhook body 中解析 out_trade_no
	outTradeNo, err := parseOutTradeNoFromWechatWebhook(body)
	var webhookResp *channel.WebhookResponse
	if err != nil {
		// 微信退款回调场景：out_trade_no 可能不在最外层，先尝试通过候选 Provider 验签解密
		webhookResp, err = tryHandleWechatRefundWebhookByFallback(c.Request.Context(), webhookReq)
		if err != nil {
			logger.Error("Failed to parse out_trade_no from webhook: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "invalid webhook data"})
			return
		}
		outTradeNo = webhookResp.PlatformTradeNo
	}

	// 通过 out_trade_no 查询订单获取 app_id
	order, err := findOrderByOutTradeNo(c.Request.Context(), outTradeNo)
	if err != nil {
		logger.Error("Failed to find order by out_trade_no: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "order not found"})
		return
	}

	appID := order.AppID
	channelName := order.Channel

	// 获取正确的支付渠道 Provider（用于签名验证）
	provider, err := channelManager.GetProvider(appID, channelName)
	if err != nil {
		logger.Error("Failed to get payment provider: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": "FAIL", "message": "系统错误"})
		return
	}

	// 调用 Provider 处理 Webhook（包含签名验证）
	if webhookResp == nil {
		webhookResp, err = provider.HandleWebhook(c.Request.Context(), webhookReq)
		if err != nil {
			logger.Error("Failed to handle webhook: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": "FAIL", "message": "处理失败"})
			return
		}
	}

	if !webhookResp.Success {
		logger.Error("Webhook verification failed")
		c.Data(http.StatusOK, "application/json", webhookResp.ResponseBody)
		return
	}

	logger.Info("Webhook verified: platformTradeNo=%s, status=%s", webhookResp.PlatformTradeNo, webhookResp.Status)

	// 更新订单状态（使用行锁）
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
			logger.Error("Failed to update order status: %v", err)
			c.Data(http.StatusOK, "application/json", webhookResp.ResponseBody)
			return
		}

		// 铁律一：事务提交后，异步通知业务系统
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
			logger.Error("Failed to update refund status: %v", err)
			c.Data(http.StatusOK, "application/json", webhookResp.ResponseBody)
			return
		}

		// 退款成功后，异步通知业务系统
		notifyService.NotifyRefundAsync(order, webhookResp)
	}

	logger.Info("Webhook processed successfully: orderNo=%s", order.OrderNo)

	// 返回成功响应给微信
	c.Data(http.StatusOK, "application/json", webhookResp.ResponseBody)
}

// parseOutTradeNoFromWechatWebhook 从微信 webhook body 中解析 out_trade_no
func parseOutTradeNoFromWechatWebhook(body []byte) (string, error) {
	// 简化实现：实际需要解析微信的加密数据
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	if outTradeNo, ok := data["out_trade_no"].(string); ok {
		return outTradeNo, nil
	}

	return "", fmt.Errorf("out_trade_no not found in webhook body")
}

// findOrderByOutTradeNo 通过 out_trade_no 查找订单
func findOrderByOutTradeNo(ctx context.Context, outTradeNo string) (*models.Order, error) {
	// 注意：这里假设 out_trade_no 在系统中是全局唯一的
	// 如果不是，需要在 webhook body 中包含 app_id
	return orderService.QueryOrderByOutTradeNoGlobal(ctx, outTradeNo)
}

// AlipayWebhook 支付宝回调
func AlipayWebhook(c *gin.Context) {
	logger.Info("Received alipay webhook")

	// 读取原始请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error("Failed to read webhook body: %v", err)
		c.String(http.StatusOK, "failure")
		return
	}

	// 获取请求头
	headers := make(map[string]string)
	for key := range c.Request.Header {
		headers[key] = c.Request.Header.Get(key)
	}

	// 构建 Webhook 请求
	webhookReq := &channel.WebhookRequest{
		RawBody: body,
		Headers: headers,
	}

	// 从 webhook body 中解析 out_trade_no
	outTradeNo, err := parseOutTradeNoFromAlipayWebhook(body)
	if err != nil {
		logger.Error("Failed to parse out_trade_no from webhook: %v", err)
		c.String(http.StatusOK, "failure")
		return
	}

	// 通过 out_trade_no 查询订单获取 app_id
	order, err := findOrderByOutTradeNo(c.Request.Context(), outTradeNo)
	if err != nil {
		logger.Error("Failed to find order by out_trade_no: %v", err)
		c.String(http.StatusOK, "failure")
		return
	}

	appID := order.AppID
	channelName := order.Channel

	// 获取正确的支付渠道 Provider（用于签名验证）
	provider, err := channelManager.GetProvider(appID, channelName)
	if err != nil {
		logger.Error("Failed to get payment provider: %v", err)
		c.String(http.StatusOK, "failure")
		return
	}

	// 调用 Provider 处理 Webhook（包含签名验证）
	webhookResp, err := provider.HandleWebhook(c.Request.Context(), webhookReq)
	if err != nil {
		logger.Error("Failed to handle webhook: %v", err)
		c.String(http.StatusOK, "failure")
		return
	}

	if !webhookResp.Success {
		logger.Error("Webhook verification failed")
		c.Data(http.StatusOK, "text/plain", webhookResp.ResponseBody)
		return
	}

	logger.Info("Webhook verified: platformTradeNo=%s, status=%s", webhookResp.PlatformTradeNo, webhookResp.Status)

	// 更新订单状态（使用行锁）
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
			logger.Error("Failed to update order status: %v", err)
			c.Data(http.StatusOK, "text/plain", webhookResp.ResponseBody)
			return
		}

		// 异步通知业务系统
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
			logger.Error("Failed to update refund status: %v", err)
			c.Data(http.StatusOK, "text/plain", webhookResp.ResponseBody)
			return
		}

		// 退款成功后，异步通知业务系统
		notifyService.NotifyRefundAsync(order, webhookResp)
	}

	logger.Info("Webhook processed successfully: orderNo=%s", order.OrderNo)

	// 返回成功响应给支付宝
	c.Data(http.StatusOK, "text/plain", webhookResp.ResponseBody)
}

// parseOutTradeNoFromAlipayWebhook 从支付宝 webhook body 中解析 out_trade_no
func parseOutTradeNoFromAlipayWebhook(body []byte) (string, error) {
	// 支付宝回调是 form 格式，需要解析
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	if outTradeNo, ok := data["out_trade_no"].(string); ok {
		return outTradeNo, nil
	}

	return "", fmt.Errorf("out_trade_no not found in webhook body")
}

func tryHandleWechatRefundWebhookByFallback(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error) {
	if !isWechatRefundEvent(req.RawBody) {
		return nil, fmt.Errorf("out_trade_no not found in webhook body")
	}

	lister, ok := channelManager.(webhookChannelProviderLister)
	if !ok {
		return nil, fmt.Errorf("refund fallback is not supported")
	}

	providers, err := lister.ListProvidersByChannelPrefix("wechat_")
	if err != nil {
		return nil, err
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf("wechat provider not found")
	}

	var lastErr error
	for _, provider := range providers {
		resp, err := provider.HandleWebhook(ctx, req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp == nil || !resp.Success {
			continue
		}
		if resp.PlatformTradeNo == "" {
			lastErr = fmt.Errorf("platform trade no is empty in refund webhook response")
			continue
		}
		return resp, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no provider matched refund webhook")
}

func isWechatRefundEvent(body []byte) bool {
	var envelope struct {
		EventType string `json:"event_type"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return false
	}
	return strings.Contains(strings.ToUpper(envelope.EventType), "REFUND")
}
