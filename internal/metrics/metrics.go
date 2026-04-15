package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// 订单相关指标
	OrdersTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gopay_orders_total",
			Help: "Total number of orders created",
		},
		[]string{"channel", "status"},
	)

	OrdersAmount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gopay_orders_amount_total",
			Help: "Total amount of orders (in cents)",
		},
		[]string{"channel"},
	)

	OrdersDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gopay_orders_duration_seconds",
			Help:    "Order creation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"channel"},
	)

	// 支付相关指标
	PaymentsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gopay_payments_total",
			Help: "Total number of successful payments",
		},
		[]string{"channel"},
	)

	PaymentsAmount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gopay_payments_amount_total",
			Help: "Total amount of successful payments (in cents)",
		},
		[]string{"channel"},
	)

	// 回调相关指标
	WebhooksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gopay_webhooks_total",
			Help: "Total number of webhooks received",
		},
		[]string{"channel", "status"},
	)

	WebhooksDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gopay_webhooks_duration_seconds",
			Help:    "Webhook processing duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"channel"},
	)

	WebhooksRetries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gopay_webhooks_retries_total",
			Help: "Total number of webhook retries",
		},
		[]string{"channel"},
	)

	// 对账相关指标
	ReconciliationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gopay_reconciliation_total",
			Help: "Total number of reconciliations",
		},
		[]string{"channel", "status"},
	)

	ReconciliationMismatches = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gopay_reconciliation_mismatches",
			Help: "Number of mismatches in reconciliation",
		},
		[]string{"channel", "type"}, // type: missing, extra, amount
	)

	// HTTP 请求指标
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gopay_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gopay_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// 数据库相关指标
	DBQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gopay_db_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation", "table"},
	)

	DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gopay_db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "table"},
	)

	// 系统指标
	ActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gopay_active_connections",
			Help: "Number of active connections",
		},
	)

	ProviderCacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gopay_provider_cache_hits_total",
			Help: "Total number of provider cache hits",
		},
		[]string{"app_id", "channel"},
	)

	ProviderCacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gopay_provider_cache_misses_total",
			Help: "Total number of provider cache misses",
		},
		[]string{"app_id", "channel"},
	)

	// 错误指标
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gopay_errors_total",
			Help: "Total number of errors",
		},
		[]string{"type", "component"},
	)
)

// RecordOrder 记录订单创建
func RecordOrder(channel, status string, amount int64, duration float64) {
	OrdersTotal.WithLabelValues(channel, status).Inc()
	OrdersAmount.WithLabelValues(channel).Add(float64(amount))
	OrdersDuration.WithLabelValues(channel).Observe(duration)
}

// RecordPayment 记录支付成功
func RecordPayment(channel string, amount int64) {
	PaymentsTotal.WithLabelValues(channel).Inc()
	PaymentsAmount.WithLabelValues(channel).Add(float64(amount))
}

// RecordWebhook 记录回调处理
func RecordWebhook(channel, status string, duration float64) {
	WebhooksTotal.WithLabelValues(channel, status).Inc()
	WebhooksDuration.WithLabelValues(channel).Observe(duration)
}

// RecordWebhookRetry 记录回调重试
func RecordWebhookRetry(channel string) {
	WebhooksRetries.WithLabelValues(channel).Inc()
}

// RecordReconciliation 记录对账
func RecordReconciliation(channel, status string, missing, extra, amountMismatch int) {
	ReconciliationTotal.WithLabelValues(channel, status).Inc()
	ReconciliationMismatches.WithLabelValues(channel, "missing").Set(float64(missing))
	ReconciliationMismatches.WithLabelValues(channel, "extra").Set(float64(extra))
	ReconciliationMismatches.WithLabelValues(channel, "amount").Set(float64(amountMismatch))
}

// RecordHTTPRequest 记录 HTTP 请求
func RecordHTTPRequest(method, path, status string, duration float64) {
	HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
}

// RecordDBQuery 记录数据库查询
func RecordDBQuery(operation, table string, duration float64) {
	DBQueriesTotal.WithLabelValues(operation, table).Inc()
	DBQueryDuration.WithLabelValues(operation, table).Observe(duration)
}

// RecordError 记录错误
func RecordError(errorType, component string) {
	ErrorsTotal.WithLabelValues(errorType, component).Inc()
}

// RecordProviderCacheHit 记录 Provider 缓存命中
func RecordProviderCacheHit(appID, channel string) {
	ProviderCacheHits.WithLabelValues(appID, channel).Inc()
}

// RecordProviderCacheMiss 记录 Provider 缓存未命中
func RecordProviderCacheMiss(appID, channel string) {
	ProviderCacheMisses.WithLabelValues(appID, channel).Inc()
}
