package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

type appScopedProvider interface {
	AppID() string
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

	order, webhookResp, err := resolveWechatWebhook(c.Request.Context(), webhookReq)
	if err != nil {
		logger.Error("Failed to resolve wechat webhook: %v", err)
		switch {
		case strings.Contains(err.Error(), "order not found"):
			c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "order not found"})
		case strings.Contains(err.Error(), "provider lookup failed"):
			c.JSON(http.StatusInternalServerError, gin.H{"code": "FAIL", "message": "系统错误"})
		case strings.Contains(err.Error(), "handle webhook failed"):
			c.JSON(http.StatusInternalServerError, gin.H{"code": "FAIL", "message": "处理失败"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "invalid webhook data"})
		}
		return
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

	outTradeNo, err := parseOutTradeNoFromAlipayWebhook(body)
	if err != nil {
		logger.Error("Failed to parse alipay out_trade_no: %v", err)
		c.String(http.StatusOK, "failure")
		return
	}

	appID, err := parseAppIDFromAlipayWebhook(body)
	var order *models.Order
	if err == nil && appID != "" {
		// 优先使用 app_id + out_trade_no 精确查询，兼容跨应用重复单号
		order, err = findOrderByAppAndOutTradeNo(c.Request.Context(), appID, outTradeNo)
	} else {
		// 兼容旧测试和历史回调格式
		order, err = findOrderByOutTradeNo(c.Request.Context(), outTradeNo)
	}
	if err != nil {
		logger.Error("Failed to find alipay order: %v", err)
		c.String(http.StatusOK, "failure")
		return
	}

	appID = order.AppID
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
	values, err := parseAlipayWebhookValues(body)
	if err != nil {
		return "", err
	}
	outTradeNo := values.Get("out_trade_no")
	if outTradeNo == "" {
		return "", fmt.Errorf("out_trade_no not found in webhook body")
	}
	return outTradeNo, nil
}

func parseAppIDFromAlipayWebhook(body []byte) (string, error) {
	values, err := parseAlipayWebhookValues(body)
	if err != nil {
		return "", err
	}
	appID := values.Get("app_id")
	if appID == "" {
		return "", fmt.Errorf("app_id not found in webhook body")
	}
	return appID, nil
}

// findOrderByOutTradeNo 兼容旧测试和旧调用路径
func findOrderByOutTradeNo(ctx context.Context, outTradeNo string) (*models.Order, error) {
	if orderService == nil {
		return nil, fmt.Errorf("order service is not initialized")
	}
	return orderService.QueryOrderByOutTradeNoGlobal(ctx, outTradeNo)
}

func findOrderByAppAndOutTradeNo(ctx context.Context, appID, outTradeNo string) (*models.Order, error) {
	if orderService == nil {
		return nil, fmt.Errorf("order service is not initialized")
	}
	return orderService.QueryOrderByOutTradeNo(ctx, appID, outTradeNo)
}

func resolveWechatWebhook(ctx context.Context, req *channel.WebhookRequest) (*models.Order, *channel.WebhookResponse, error) {
	outTradeNo, err := parseOutTradeNoFromWechatWebhook(req.RawBody)
	if err == nil {
		order, err := findOrderByOutTradeNo(ctx, outTradeNo)
		if err != nil {
			return nil, nil, err
		}
		provider, err := channelManager.GetProvider(order.AppID, order.Channel)
		if err != nil {
			return nil, nil, fmt.Errorf("provider lookup failed: %w", err)
		}
		resp, err := provider.HandleWebhook(ctx, req)
		if err != nil {
			return nil, nil, fmt.Errorf("handle webhook failed: %w", err)
		}
		return order, resp, nil
	}

	lister, ok := channelManager.(webhookChannelProviderLister)
	if ok {
		providers, err := lister.ListProvidersByChannelPrefix("wechat_")
		if err != nil {
			return nil, nil, err
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
				lastErr = fmt.Errorf("platform trade no is empty in wechat webhook response")
				continue
			}

			if scoped, ok := provider.(appScopedProvider); ok && scoped.AppID() != "" {
				order, err := findOrderByAppAndOutTradeNo(ctx, scoped.AppID(), resp.PlatformTradeNo)
				if err == nil {
					return order, resp, nil
				}
				lastErr = err
				continue
			}

			order, err := findOrderByOutTradeNo(ctx, resp.PlatformTradeNo)
			if err == nil {
				return order, resp, nil
			}
			lastErr = err
		}

		if lastErr != nil {
			return nil, nil, lastErr
		}
	}

	return nil, nil, err
}

func parseAlipayWebhookValues(body []byte) (url.Values, error) {
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, err
	}
	if values.Get("out_trade_no") != "" || values.Get("app_id") != "" || values.Get("trade_no") != "" {
		return values, nil
	}

	var data map[string]string
	if err := json.Unmarshal(body, &data); err == nil && len(data) > 0 {
		fallback := url.Values{}
		for key, value := range data {
			fallback.Set(key, value)
		}
		return fallback, nil
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("empty alipay webhook body")
	}
	return values, nil
}
