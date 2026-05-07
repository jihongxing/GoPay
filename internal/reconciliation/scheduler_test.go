package reconciliation

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockAlertNotifier struct {
	messages []string
}

func (m *mockAlertNotifier) SendAlert(ctx context.Context, message string) error {
	m.messages = append(m.messages, message)
	return nil
}

func TestScheduler_GetNextRunTime_Details(t *testing.T) {
	s := &Scheduler{}

	nextRun := s.getNextRunTime()

	// Should be at 2:00 AM
	assert.Equal(t, 2, nextRun.Hour())
	assert.Equal(t, 0, nextRun.Minute())
	assert.Equal(t, 0, nextRun.Second())

	// Should be in the future
	assert.True(t, nextRun.After(time.Now()))
}

func TestScheduler_GetNextRunTime_BeforeTwoAM(t *testing.T) {
	s := &Scheduler{}
	nextRun := s.getNextRunTime()

	now := time.Now()
	today2AM := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())

	if now.Before(today2AM) {
		// Should be today at 2 AM
		assert.Equal(t, now.Day(), nextRun.Day())
	} else {
		// Should be tomorrow at 2 AM
		tomorrow := now.AddDate(0, 0, 1)
		assert.Equal(t, tomorrow.Day(), nextRun.Day())
	}
}

func TestScheduler_Stop_NotRunning(t *testing.T) {
	s := &Scheduler{
		running: false,
		stopCh:  make(chan struct{}),
	}
	// Should not panic
	s.Stop()
}

func TestScheduler_StartAlreadyRunning_Error(t *testing.T) {
	s := &Scheduler{
		running: true,
		stopCh:  make(chan struct{}),
	}

	err := s.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

func TestScheduler_SendDifferenceAlert(t *testing.T) {
	alert := &mockAlertNotifier{}
	s := &Scheduler{alertNotifier: alert}

	result := &ReconcileResult{
		Channel:       "wechat",
		Date:          time.Now(),
		TotalOrders:   100,
		MatchedOrders: 95,
		MissingOrders: []string{"ORD_001"},
		ExtraOrders:   []string{"ORD_002", "ORD_003"},
		AmountMismatch: []AmountMismatch{
			{OrderNo: "ORD_004", InternalAmount: 100, ExternalAmount: 200},
		},
	}

	s.sendDifferenceAlert(context.Background(), result)

	assert.Len(t, alert.messages, 1)
	assert.Contains(t, alert.messages[0], "wechat")
	assert.Contains(t, alert.messages[0], "1 笔") // missing
	assert.Contains(t, alert.messages[0], "2 笔") // extra
}

func TestScheduler_SendDifferenceAlert_NilNotifier(t *testing.T) {
	s := &Scheduler{alertNotifier: nil}

	// Should not panic
	s.sendDifferenceAlert(context.Background(), &ReconcileResult{})
}

func TestDummyAlertNotifier(t *testing.T) {
	n := &DummyAlertNotifier{}
	err := n.SendAlert(context.Background(), "test message")
	assert.NoError(t, err)
}

func TestScheduler_RunNow_NoDatabase(t *testing.T) {
	alert := &mockAlertNotifier{}
	s := &Scheduler{
		service:       NewReconciliationService(),
		alertNotifier: alert,
	}

	// RunNow without a real DB will fail at reconciliation
	err := s.RunNow(context.Background(), time.Now().AddDate(0, 0, -1))
	// Should return error since no DB
	assert.Error(t, err)
}
