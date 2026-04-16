package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"gopay/internal/reconciliation"
)

func main() {
	// 命令行参数
	var (
		dbDSN      = flag.String("db", "", "数据库连接字符串")
		runOnce    = flag.Bool("once", false, "只执行一次对账（不启动定时任务）")
		targetDate = flag.String("date", "", "对账日期（格式: 2006-01-02，默认为昨天）")
	)
	flag.Parse()

	// 检查数据库连接
	if *dbDSN == "" {
		log.Fatal("请提供数据库连接字符串: -db=\"postgres://user:pass@localhost/gopay\"")
	}

	// 连接数据库
	db, err := sql.Open("postgres", *dbDSN)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	// 测试数据库连接
	if err := db.Ping(); err != nil {
		log.Fatalf("数据库连接测试失败: %v", err)
	}
	log.Println("数据库连接成功")

	// 创建告警通知器
	alertNotifier := &reconciliation.DummyAlertNotifier{}

	// 创建调度器
	scheduler := reconciliation.NewScheduler(db, alertNotifier)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 如果是一次性执行
	if *runOnce {
		// 解析日期
		var date time.Time
		if *targetDate != "" {
			date, err = time.Parse("2006-01-02", *targetDate)
			if err != nil {
				log.Fatalf("日期格式错误: %v", err)
			}
		} else {
			// 默认为昨天
			date = time.Now().AddDate(0, 0, -1)
		}

		log.Printf("执行一次性对账任务，日期: %s", date.Format("2006-01-02"))
		if err := scheduler.RunNow(ctx, date); err != nil {
			log.Fatalf("对账任务执行失败: %v", err)
		}
		log.Println("对账任务执行完成")
		return
	}

	// 启动定时任务
	log.Println("启动对账定时任务...")

	// 监听系统信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 在 goroutine 中启动调度器
	errCh := make(chan error, 1)
	go func() {
		errCh <- scheduler.Start(ctx)
	}()

	// 等待退出信号或错误
	select {
	case sig := <-sigCh:
		log.Printf("收到信号: %v，正在停止...", sig)
		scheduler.Stop()
		cancel()
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			log.Printf("调度器异常退出: %v", err)
		}
	}

	log.Println("对账服务已停止")
}
