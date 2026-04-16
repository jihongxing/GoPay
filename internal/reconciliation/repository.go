package reconciliation

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// DBOrderRepository 基于数据库的订单仓储实现
type DBOrderRepository struct {
	db *sql.DB
}

// NewDBOrderRepository 创建数据库订单仓储
func NewDBOrderRepository(db *sql.DB) OrderRepository {
	return &DBOrderRepository{db: db}
}

// GetOrdersByDate 查询指定日期和渠道的订单
func (r *DBOrderRepository) GetOrdersByDate(ctx context.Context, date time.Time, channel string) ([]Order, error) {
	// 构建查询条件
	startTime := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endTime := startTime.Add(24 * time.Hour)

	query := `
		SELECT order_no, amount, status, paid_at
		FROM orders
		WHERE channel LIKE $1
		  AND paid_at >= $2
		  AND paid_at < $3
		  AND status = 'paid'
		ORDER BY paid_at
	`

	// 构建渠道匹配模式（支持 wechat_native, wechat_jsapi 等）
	channelPattern := channel + "%"

	rows, err := r.db.QueryContext(ctx, query, channelPattern, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("query orders failed: %w", err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var order Order
		var paidAt sql.NullTime

		err := rows.Scan(&order.OrderNo, &order.Amount, &order.Status, &paidAt)
		if err != nil {
			return nil, fmt.Errorf("scan order failed: %w", err)
		}

		if paidAt.Valid {
			order.PaidAt = paidAt.Time
		}

		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orders failed: %w", err)
	}

	return orders, nil
}
