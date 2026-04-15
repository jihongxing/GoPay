package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/yourusername/gopay/examples/go-client/client"
)

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// 解析命令行参数
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	// 创建客户端
	gopayURL := getEnv("GOPAY_URL", "http://localhost:8080")
	appID := getEnv("APP_ID", "")
	if appID == "" {
		log.Fatal("APP_ID is required")
	}

	c := client.NewClient(gopayURL, appID)

	// 执行命令
	command := args[0]
	switch command {
	case "create":
		createOrder(c)
	case "query":
		if len(args) < 2 {
			log.Fatal("Usage: go run main.go query <order_no>")
		}
		queryOrder(c, args[1])
	case "callback":
		startCallbackServer()
	default:
		log.Fatalf("Unknown command: %s", command)
	}
}

// createOrder 创建支付订单
func createOrder(c *client.Client) {
	req := &client.CreateOrderRequest{
		OutTradeNo: fmt.Sprintf("GO_ORDER_%d", os.Getpid()),
		Amount:     100, // 1元 = 100分
		Subject:    "测试商品",
		Body:       "这是一个测试订单",
		Channel:    getEnv("CHANNEL", "wechat_native"),
		NotifyURL:  getEnv("NOTIFY_URL", "http://localhost:8081/callback"),
	}

	log.Printf("创建订单: %+v", req)

	resp, err := c.CreateOrder(req)
	if err != nil {
		log.Fatalf("创建订单失败: %v", err)
	}

	log.Printf("✅ 订单创建成功!")
	log.Printf("订单号: %s", resp.OrderNo)
	log.Printf("商户订单号: %s", req.OutTradeNo)

	if resp.PayURL != "" {
		log.Printf("支付链接: %s", resp.PayURL)
	}

	if resp.QRCode != "" {
		log.Printf("二维码: %s", resp.QRCode)
		log.Printf("请使用微信扫描二维码完成支付")
	}

	if resp.PrepayID != "" {
		log.Printf("预支付ID: %s", resp.PrepayID)
	}
}

// queryOrder 查询订单状态
func queryOrder(c *client.Client, orderNo string) {
	log.Printf("查询订单: %s", orderNo)

	order, err := c.QueryOrder(orderNo)
	if err != nil {
		log.Fatalf("查询订单失败: %v", err)
	}

	log.Printf("✅ 订单查询成功!")
	log.Printf("订单号: %s", order.OrderNo)
	log.Printf("商户订单号: %s", order.OutTradeNo)
	log.Printf("订单状态: %s", order.Status)
	log.Printf("支付金额: %d 分 (%.2f 元)", order.Amount, float64(order.Amount)/100)
	log.Printf("支付渠道: %s", order.Channel)
	log.Printf("创建时间: %s", order.CreatedAt)

	if order.PaidAt != nil {
		log.Printf("支付时间: %s", *order.PaidAt)
	}
}

// startCallbackServer 启动回调服务器
func startCallbackServer() {
	port := getEnv("CALLBACK_PORT", "8081")

	http.HandleFunc("/callback", handleCallback)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	log.Printf("🚀 回调服务器启动在端口 %s", port)
	log.Printf("回调地址: http://localhost:%s/callback", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// handleCallback 处理支付回调
func handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析回调数据
	var callback client.CallbackData
	if err := json.NewDecoder(r.Body).Decode(&callback); err != nil {
		log.Printf("❌ 解析回调数据失败: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	log.Printf("📨 收到支付回调:")
	log.Printf("  订单号: %s", callback.OrderNo)
	log.Printf("  商户订单号: %s", callback.OutTradeNo)
	log.Printf("  订单状态: %s", callback.Status)
	log.Printf("  支付金额: %d 分", callback.Amount)
	log.Printf("  支付渠道: %s", callback.Channel)

	// 这里处理你的业务逻辑
	// 例如：更新订单状态、发货、发送通知等
	if callback.Status == "paid" {
		log.Printf("✅ 订单支付成功，可以进行后续业务处理")
		// TODO: 实现你的业务逻辑
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"code":    "SUCCESS",
		"message": "OK",
	})
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// printUsage 打印使用说明
func printUsage() {
	fmt.Println("GoPay Go 客户端示例")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  go run main.go create              # 创建支付订单")
	fmt.Println("  go run main.go query <order_no>    # 查询订单状态")
	fmt.Println("  go run main.go callback            # 启动回调服务器")
	fmt.Println()
	fmt.Println("环境变量:")
	fmt.Println("  GOPAY_URL      GoPay 服务地址 (默认: http://localhost:8080)")
	fmt.Println("  APP_ID         应用ID (必填)")
	fmt.Println("  CHANNEL        支付渠道 (默认: wechat_native)")
	fmt.Println("  NOTIFY_URL     回调地址 (默认: http://localhost:8081/callback)")
	fmt.Println("  CALLBACK_PORT  回调服务器端口 (默认: 8081)")
}
