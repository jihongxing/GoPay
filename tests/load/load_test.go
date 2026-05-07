package load

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func requireLoadTestServer(t *testing.T, baseURL string) {
	t.Helper()

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		t.Skipf("跳过压力测试，本地服务不可用: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("跳过压力测试，健康检查返回非 200: %d", resp.StatusCode)
	}
}

// LoadTestConfig 压力测试配置
type LoadTestConfig struct {
	BaseURL     string
	AppID       string
	AppSecret   string
	Concurrency int           // 并发数
	Duration    time.Duration // 测试持续时间
	RampUpTime  time.Duration // 预热时间
	TargetQPS   int           // 目标 QPS
	RequestType string        // 请求类型: checkout, query
}

// LoadTestResult 压力测试结果
type LoadTestResult struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalDuration   time.Duration
	AvgResponseTime time.Duration
	MinResponseTime time.Duration
	MaxResponseTime time.Duration
	P50ResponseTime time.Duration
	P95ResponseTime time.Duration
	P99ResponseTime time.Duration
	ActualQPS       float64
	ErrorRate       float64
	ResponseTimes   []time.Duration
}

// CheckoutRequest 下单请求
type CheckoutRequest struct {
	AppID      string `json:"app_id"`
	OutTradeNo string `json:"out_trade_no"`
	Amount     int64  `json:"amount"`
	Subject    string `json:"subject"`
	Channel    string `json:"channel"`
}

// signRequest 对请求进行签名
func signRequest(appSecret, body, timestamp, nonce string) string {
	message := body + "\n" + timestamp + "\n" + nonce
	h := hmac.New(sha256.New, []byte(appSecret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// sendCheckoutRequest 发送下单请求
func sendCheckoutRequest(config *LoadTestConfig) (time.Duration, error) {
	req := CheckoutRequest{
		AppID:      config.AppID,
		OutTradeNo: fmt.Sprintf("LOAD_TEST_%s", uuid.New().String()),
		Amount:     100,
		Subject:    "压力测试商品",
		Channel:    "wechat_native",
	}

	body, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce := uuid.New().String()
	signature := signRequest(config.AppSecret, string(body), timestamp, nonce)

	httpReq, err := http.NewRequest("POST", config.BaseURL+"/api/v1/checkout", bytes.NewBuffer(body))
	if err != nil {
		return 0, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-App-ID", config.AppID)
	httpReq.Header.Set("X-Timestamp", timestamp)
	httpReq.Header.Set("X-Nonce", nonce)
	httpReq.Header.Set("X-Signature", signature)

	start := time.Now()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	duration := time.Since(start)

	if err != nil {
		return duration, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return duration, fmt.Errorf("status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return duration, nil
}

// sendQueryRequest 发送查询请求
func sendQueryRequest(config *LoadTestConfig, orderNo string) (time.Duration, error) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce := uuid.New().String()
	signature := signRequest(config.AppSecret, "", timestamp, nonce)

	httpReq, err := http.NewRequest("GET", config.BaseURL+"/api/v1/orders/"+orderNo, nil)
	if err != nil {
		return 0, err
	}

	httpReq.Header.Set("X-App-ID", config.AppID)
	httpReq.Header.Set("X-Timestamp", timestamp)
	httpReq.Header.Set("X-Nonce", nonce)
	httpReq.Header.Set("X-Signature", signature)

	start := time.Now()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	duration := time.Since(start)

	if err != nil {
		return duration, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return duration, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return duration, nil
}

// RunLoadTest 执行压力测试
func RunLoadTest(config *LoadTestConfig) *LoadTestResult {
	result := &LoadTestResult{
		MinResponseTime: time.Hour,
		ResponseTimes:   make([]time.Duration, 0),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	startTime := time.Now()
	endTime := startTime.Add(config.Duration)

	// 计算每个 goroutine 的请求间隔
	requestInterval := time.Duration(0)
	if config.TargetQPS > 0 {
		requestInterval = time.Second / time.Duration(config.TargetQPS/config.Concurrency)
	}

	// 启动并发 worker
	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// 预热期：逐步增加负载
			if config.RampUpTime > 0 {
				rampUpDelay := config.RampUpTime / time.Duration(config.Concurrency)
				time.Sleep(time.Duration(workerID) * rampUpDelay)
			}

			ticker := time.NewTicker(requestInterval)
			defer ticker.Stop()

			for time.Now().Before(endTime) {
				var duration time.Duration
				var err error

				// 根据请求类型发送不同的请求
				if config.RequestType == "query" {
					duration, err = sendQueryRequest(config, "ORD_TEST_001")
				} else {
					duration, err = sendCheckoutRequest(config)
				}

				atomic.AddInt64(&result.TotalRequests, 1)

				mu.Lock()
				result.ResponseTimes = append(result.ResponseTimes, duration)
				if duration < result.MinResponseTime {
					result.MinResponseTime = duration
				}
				if duration > result.MaxResponseTime {
					result.MaxResponseTime = duration
				}
				mu.Unlock()

				if err != nil {
					atomic.AddInt64(&result.FailedRequests, 1)
				} else {
					atomic.AddInt64(&result.SuccessRequests, 1)
				}

				// 控制 QPS
				if requestInterval > 0 {
					<-ticker.C
				}
			}
		}(i)
	}

	wg.Wait()

	// 计算统计数据
	result.TotalDuration = time.Since(startTime)
	result.ActualQPS = float64(result.TotalRequests) / result.TotalDuration.Seconds()
	result.ErrorRate = float64(result.FailedRequests) / float64(result.TotalRequests) * 100

	// 计算平均响应时间
	var totalTime time.Duration
	for _, t := range result.ResponseTimes {
		totalTime += t
	}
	if len(result.ResponseTimes) > 0 {
		result.AvgResponseTime = totalTime / time.Duration(len(result.ResponseTimes))
	}

	// 计算百分位数
	if len(result.ResponseTimes) > 0 {
		// 简单排序
		sortedTimes := make([]time.Duration, len(result.ResponseTimes))
		copy(sortedTimes, result.ResponseTimes)

		// 冒泡排序（对于大数据集应该使用更高效的排序算法）
		for i := 0; i < len(sortedTimes); i++ {
			for j := i + 1; j < len(sortedTimes); j++ {
				if sortedTimes[i] > sortedTimes[j] {
					sortedTimes[i], sortedTimes[j] = sortedTimes[j], sortedTimes[i]
				}
			}
		}

		result.P50ResponseTime = sortedTimes[len(sortedTimes)*50/100]
		result.P95ResponseTime = sortedTimes[len(sortedTimes)*95/100]
		result.P99ResponseTime = sortedTimes[len(sortedTimes)*99/100]
	}

	return result
}

// PrintResult 打印测试结果
func PrintResult(result *LoadTestResult) {
	fmt.Println("\n========== 压力测试结果 ==========")
	fmt.Printf("总请求数:       %d\n", result.TotalRequests)
	fmt.Printf("成功请求数:     %d\n", result.SuccessRequests)
	fmt.Printf("失败请求数:     %d\n", result.FailedRequests)
	fmt.Printf("测试持续时间:   %v\n", result.TotalDuration)
	fmt.Printf("实际 QPS:       %.2f\n", result.ActualQPS)
	fmt.Printf("错误率:         %.2f%%\n", result.ErrorRate)
	fmt.Println("\n响应时间统计:")
	fmt.Printf("  平均:         %v\n", result.AvgResponseTime)
	fmt.Printf("  最小:         %v\n", result.MinResponseTime)
	fmt.Printf("  最大:         %v\n", result.MaxResponseTime)
	fmt.Printf("  P50:          %v\n", result.P50ResponseTime)
	fmt.Printf("  P95:          %v\n", result.P95ResponseTime)
	fmt.Printf("  P99:          %v\n", result.P99ResponseTime)
	fmt.Println("==================================")
}

// TestLoadCheckout 测试下单接口压力
func TestLoadCheckout(t *testing.T) {
	config := &LoadTestConfig{
		BaseURL:     "http://localhost:8080",
		AppID:       "test_app_001",
		AppSecret:   "test_secret_123456",
		Concurrency: 100,
		Duration:    30 * time.Second,
		RampUpTime:  5 * time.Second,
		TargetQPS:   1000,
		RequestType: "checkout",
	}
	requireLoadTestServer(t, config.BaseURL)

	t.Logf("开始压力测试: 目标 QPS=%d, 并发数=%d, 持续时间=%v",
		config.TargetQPS, config.Concurrency, config.Duration)

	result := RunLoadTest(config)
	PrintResult(result)

	// 验证性能指标
	if result.ActualQPS < float64(config.TargetQPS)*0.8 {
		t.Errorf("实际 QPS (%.2f) 低于目标的 80%% (%d)", result.ActualQPS, config.TargetQPS)
	}

	if result.ErrorRate > 1.0 {
		t.Errorf("错误率 (%.2f%%) 超过 1%%", result.ErrorRate)
	}

	if result.P95ResponseTime > 100*time.Millisecond {
		t.Errorf("P95 响应时间 (%v) 超过 100ms", result.P95ResponseTime)
	}
}

// TestLoadQuery 测试查询接口压力
func TestLoadQuery(t *testing.T) {
	config := &LoadTestConfig{
		BaseURL:     "http://localhost:8080",
		AppID:       "test_app_001",
		AppSecret:   "test_secret_123456",
		Concurrency: 200,
		Duration:    30 * time.Second,
		RampUpTime:  5 * time.Second,
		TargetQPS:   5000,
		RequestType: "query",
	}
	requireLoadTestServer(t, config.BaseURL)

	t.Logf("开始压力测试: 目标 QPS=%d, 并发数=%d, 持续时间=%v",
		config.TargetQPS, config.Concurrency, config.Duration)

	result := RunLoadTest(config)
	PrintResult(result)

	// 查询接口应该更快
	if result.P95ResponseTime > 50*time.Millisecond {
		t.Errorf("P95 响应时间 (%v) 超过 50ms", result.P95ResponseTime)
	}
}

// TestLoad10kQPS 测试 10k+ QPS 目标
func TestLoad10kQPS(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过长时间压力测试")
	}

	config := &LoadTestConfig{
		BaseURL:     "http://localhost:8080",
		AppID:       "test_app_001",
		AppSecret:   "test_secret_123456",
		Concurrency: 500,
		Duration:    60 * time.Second,
		RampUpTime:  10 * time.Second,
		TargetQPS:   10000,
		RequestType: "query",
	}
	requireLoadTestServer(t, config.BaseURL)

	t.Logf("开始 10k+ QPS 压力测试: 目标 QPS=%d, 并发数=%d, 持续时间=%v",
		config.TargetQPS, config.Concurrency, config.Duration)

	result := RunLoadTest(config)
	PrintResult(result)

	// 验证是否达到 10k+ QPS
	if result.ActualQPS < 10000 {
		t.Logf("警告: 实际 QPS (%.2f) 未达到 10k 目标", result.ActualQPS)
	} else {
		t.Logf("成功: 实际 QPS (%.2f) 达到 10k+ 目标", result.ActualQPS)
	}

	// 验证响应时间
	if result.P95ResponseTime > 100*time.Millisecond {
		t.Logf("警告: P95 响应时间 (%v) 超过 100ms", result.P95ResponseTime)
	}

	// 验证错误率
	if result.ErrorRate > 0.1 {
		t.Errorf("错误率 (%.2f%%) 超过 0.1%%", result.ErrorRate)
	}
}
