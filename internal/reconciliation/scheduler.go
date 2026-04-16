package reconciliation

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// Scheduler 对账定时调度器
type Scheduler struct {
	service       *ReconciliationService
	alertNotifier AlertNotifier
	running       bool
	stopCh        chan struct{}
}

// AlertNotifier 告警通知接口
type AlertNotifier interface {
	SendAlert(ctx context.Context, message string) error
}

// NewScheduler 创建调度器
func NewScheduler(db *sql.DB, alertNotifier AlertNotifier) *Scheduler {
	// 创建带数据库连接的对账服务
	service := &ReconciliationService{
		wechatReconciler: &WechatReconciler{
			billDownloader: NewWechatBillDownloader(),
			orderRepo:      NewDBOrderRepository(db),
		},
		alipayReconciler: &AlipayReconciler{
			billDownloader: NewAlipayBillDownloader(),
			orderRepo:      NewDBOrderRepository(db),
		},
		reportGenerator: NewReportGenerator(),
	}

	return &Scheduler{
		service:       service,
		alertNotifier: alertNotifier,
		stopCh:        make(chan struct{}),
	}
}

// Start 启动定时任务
func (s *Scheduler) Start(ctx context.Context) error {
	if s.running {
		return fmt.Errorf("scheduler already running")
	}

	s.running = true
	log.Println("对账调度器已启动")

	// 计算下次执行时间（每天凌晨 2 点）
	nextRun := s.getNextRunTime()
	log.Printf("下次对账时间: %s", nextRun.Format("2006-01-02 15:04:05"))

	// 启动定时器
	timer := time.NewTimer(time.Until(nextRun))
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("对账调度器已停止（上下文取消）")
			s.running = false
			return ctx.Err()

		case <-s.stopCh:
			log.Println("对账调度器已停止")
			s.running = false
			return nil

		case <-timer.C:
			// 执行对账任务
			log.Println("开始执行 T+1 对账任务...")
			s.runReconciliation(ctx)

			// 计算下次执行时间
			nextRun = s.getNextRunTime()
			log.Printf("下次对账时间: %s", nextRun.Format("2006-01-02 15:04:05"))
			timer.Reset(time.Until(nextRun))
		}
	}
}

// Stop 停止定时任务
func (s *Scheduler) Stop() {
	if s.running {
		close(s.stopCh)
	}
}

// RunNow 立即执行一次对账（用于手动触发）
func (s *Scheduler) RunNow(ctx context.Context, date time.Time) error {
	log.Printf("手动触发对账任务，日期: %s", date.Format("2006-01-02"))
	return s.reconcileDate(ctx, date)
}

// runReconciliation 执行对账任务
func (s *Scheduler) runReconciliation(ctx context.Context) {
	// 对前一天的数据进行对账
	yesterday := time.Now().AddDate(0, 0, -1)

	if err := s.reconcileDate(ctx, yesterday); err != nil {
		log.Printf("对账任务执行失败: %v", err)
		// 发送告警
		if s.alertNotifier != nil {
			alertMsg := fmt.Sprintf("对账任务执行失败\n日期: %s\n错误: %v",
				yesterday.Format("2006-01-02"), err)
			s.alertNotifier.SendAlert(ctx, alertMsg)
		}
	}
}

// reconcileDate 对指定日期执行对账
func (s *Scheduler) reconcileDate(ctx context.Context, date time.Time) error {
	startTime := time.Now()
	log.Printf("开始对账，日期: %s", date.Format("2006-01-02"))

	// 执行所有渠道的对账
	results, err := s.service.ReconcileAll(ctx, date)
	if err != nil {
		return fmt.Errorf("reconcile all channels failed: %w", err)
	}

	// 处理对账结果
	hasError := false
	for _, result := range results {
		// 生成报告
		reportPath, err := s.service.GenerateReport(ctx, result)
		if err != nil {
			log.Printf("生成对账报告失败 [%s]: %v", result.Channel, err)
			hasError = true
			continue
		}

		log.Printf("对账报告已生成 [%s]: %s", result.Channel, reportPath)

		// 检查是否有差异
		hasDifference := len(result.MissingOrders) > 0 ||
			len(result.ExtraOrders) > 0 ||
			len(result.AmountMismatch) > 0

		if hasDifference {
			// 发送告警
			s.sendDifferenceAlert(ctx, result)
			hasError = true
		} else {
			log.Printf("对账成功 [%s]: 总订单 %d, 匹配 %d",
				result.Channel, result.TotalOrders, result.MatchedOrders)
		}
	}

	duration := time.Since(startTime)
	log.Printf("对账任务完成，耗时: %v", duration)

	if hasError {
		return fmt.Errorf("对账发现差异，请查看报告")
	}

	return nil
}

// sendDifferenceAlert 发送差异告警
func (s *Scheduler) sendDifferenceAlert(ctx context.Context, result *ReconcileResult) {
	if s.alertNotifier == nil {
		return
	}

	alertMsg := fmt.Sprintf(
		"⚠️ 对账异常告警\n\n"+
			"渠道: %s\n"+
			"日期: %s\n"+
			"总订单数: %d\n"+
			"匹配订单数: %d\n"+
			"长款（外部有但内部无）: %d 笔\n"+
			"短款（内部有但外部无）: %d 笔\n"+
			"金额不匹配: %d 笔\n\n"+
			"请及时处理！",
		result.Channel,
		result.Date.Format("2006-01-02"),
		result.TotalOrders,
		result.MatchedOrders,
		len(result.MissingOrders),
		len(result.ExtraOrders),
		len(result.AmountMismatch),
	)

	if err := s.alertNotifier.SendAlert(ctx, alertMsg); err != nil {
		log.Printf("发送告警失败: %v", err)
	}
}

// getNextRunTime 计算下次执行时间（每天凌晨 2 点）
func (s *Scheduler) getNextRunTime() time.Time {
	now := time.Now()

	// 今天凌晨 2 点
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())

	// 如果已经过了今天凌晨 2 点，则设置为明天凌晨 2 点
	if now.After(nextRun) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	return nextRun
}

// DummyAlertNotifier 空告警通知器（用于测试）
type DummyAlertNotifier struct{}

func (n *DummyAlertNotifier) SendAlert(ctx context.Context, message string) error {
	log.Printf("告警通知: %s", message)
	return nil
}
