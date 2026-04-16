package handler

import (
	"github.com/gin-gonic/gin"
	"gopay/internal/service"
	"gopay/pkg/logger"
	"gopay/pkg/response"
)

// RefundOrder 发起退款
func RefundOrder(c *gin.Context) {
	if refundService == nil {
		response.InternalError(c, "退款服务未初始化", "")
		return
	}

	var req struct {
		Amount int64  `json:"amount"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误", err.Error())
		return
	}

	orderNo := c.Param("order_no")
	if orderNo == "" {
		response.BadRequest(c, "订单号不能为空")
		return
	}

	logger.Info("Refund request: order_no=%s, amount=%d", orderNo, req.Amount)

	result, err := refundService.Refund(c.Request.Context(), &service.RefundRequest{
		OrderNo: orderNo,
		Amount:  req.Amount,
		Reason:  req.Reason,
	})
	if err != nil {
		logger.Error("Failed to refund order: %v", err)
		response.HandleError(c, err)
		return
	}

	response.Success(c, "退款已提交", result)
}

// QueryRefund 查询退款状态
func QueryRefund(c *gin.Context) {
	if refundService == nil {
		response.InternalError(c, "退款服务未初始化", "")
		return
	}

	orderNo := c.Param("order_no")
	refundNo := c.Param("refund_no")
	if orderNo == "" || refundNo == "" {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := refundService.QueryRefund(c.Request.Context(), orderNo, refundNo)
	if err != nil {
		logger.Error("Failed to query refund: %v", err)
		response.HandleError(c, err)
		return
	}

	response.Success(c, "查询成功", result)
}
