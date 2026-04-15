package handler

import (
	"github.com/gin-gonic/gin"
	"gopay/pkg/logger"
	"gopay/pkg/response"
)

// ListFailedOrders 查询失败的订单
func ListFailedOrders(c *gin.Context) {
	logger.Info("List failed orders request")

	// 查询通知失败的订单
	orders, err := orderService.ListPendingNotifyOrders(c.Request.Context(), 100)
	if err != nil {
		logger.Error("Failed to list failed orders: %v", err)
		response.InternalError(c, "查询失败订单失败", err.Error())
		return
	}

	response.Success(c, "查询成功", gin.H{
		"total":  len(orders),
		"orders": orders,
	})
}

// RetryNotify 手动重试通知
func RetryNotify(c *gin.Context) {
	orderNo := c.Param("order_no")
	if orderNo == "" {
		response.BadRequest(c, "订单号不能为空")
		return
	}

	logger.Info("Retry notify request: order_no=%s", orderNo)

	// 调用通知服务重试
	err := notifyService.RetryNotify(c.Request.Context(), orderNo)
	if err != nil {
		logger.Error("Failed to retry notify: %v", err)
		response.HandleError(c, err)
		return
	}

	response.Success(c, "重试已启动", gin.H{"order_no": orderNo})
}
