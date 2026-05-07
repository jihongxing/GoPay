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
	return r.getOrders(ctx, date, channel, "")
}

// GetOrdersByDateAndApp 查询指定日期、渠道和应用的订单
func (r *DBOrderRepository) GetOrdersByDateAndApp(ctx context.Context, date time.Time, channel, appID string) ([]Order, error) {
	return r.getOrders(ctx, date, channel, appID)
}

func (r *DBOrderRepository) getOrders(ctx context.Context, date time.Time, channel, appID string) ([]Order, error) {
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
	`
	args := []any{channel + "%", startTime, endTime}
	if appID != "" {
		query += " AND app_id = $4"
		args = append(args, appID)
	}
	query += "\n\t\tORDER BY paid_at\n\t"

	rows, err := r.db.QueryContext(ctx, query, args...)
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
