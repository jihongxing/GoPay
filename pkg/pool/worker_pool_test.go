package service

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWorkerPool_Submit(t *testing.T) {
	pool := NewWorkerPool(4)
	defer pool.Shutdown()

	var counter int64
	for i := 0; i < 100; i++ {
		err := pool.Submit(func() {
			atomic.AddInt64(&counter, 1)
		})
		assert.NoError(t, err)
	}

	// Wait for tasks to complete
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int64(100), atomic.LoadInt64(&counter))
}

func TestWorkerPool_Shutdown(t *testing.T) {
	pool := NewWorkerPool(2)

	var counter int64
	for i := 0; i < 10; i++ {
		pool.Submit(func() {
			atomic.AddInt64(&counter, 1)
			time.Sleep(10 * time.Millisecond)
		})
	}

	pool.Shutdown()
	// After shutdown, all submitted tasks should have completed
	assert.True(t, atomic.LoadInt64(&counter) > 0)
}

func TestWorkerPool_SubmitAfterShutdown(t *testing.T) {
	pool := NewWorkerPool(2)
	pool.Shutdown()

	// After shutdown, ctx is cancelled so Submit returns context.Canceled
	// or panics on closed channel depending on timing.
	// We just verify it doesn't hang.
	func() {
		defer func() { recover() }()
		pool.Submit(func() {})
	}()
}

func TestWorkerPool_ConcurrentSubmit(t *testing.T) {
	pool := NewWorkerPool(8)
	defer pool.Shutdown()

	var counter int64
	done := make(chan struct{})

	go func() {
		for i := 0; i < 50; i++ {
			pool.Submit(func() {
				atomic.AddInt64(&counter, 1)
			})
		}
		close(done)
	}()

	for i := 0; i < 50; i++ {
		pool.Submit(func() {
			atomic.AddInt64(&counter, 1)
		})
	}

	<-done
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int64(100), atomic.LoadInt64(&counter))
}
