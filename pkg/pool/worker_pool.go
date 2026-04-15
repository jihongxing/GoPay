package service

import (
	"context"
	"sync"
)

// WorkerPool goroutine 工作池
type WorkerPool struct {
	workers   chan struct{}
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	taskQueue chan func()
}

// NewWorkerPool 创建工作池
func NewWorkerPool(maxWorkers int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &WorkerPool{
		workers:   make(chan struct{}, maxWorkers),
		ctx:       ctx,
		cancel:    cancel,
		taskQueue: make(chan func(), maxWorkers*2), // 任务队列容量为 worker 数量的 2 倍
	}

	// 启动 worker goroutines
	for i := 0; i < maxWorkers; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	return pool
}

// worker 工作协程
func (p *WorkerPool) worker() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case task := <-p.taskQueue:
			if task != nil {
				task()
			}
		}
	}
}

// Submit 提交任务
func (p *WorkerPool) Submit(task func()) error {
	select {
	case <-p.ctx.Done():
		return context.Canceled
	case p.taskQueue <- task:
		return nil
	default:
		// 队列已满，阻塞等待
		select {
		case <-p.ctx.Done():
			return context.Canceled
		case p.taskQueue <- task:
			return nil
		}
	}
}

// Shutdown 关闭工作池
func (p *WorkerPool) Shutdown() {
	p.cancel()
	close(p.taskQueue)
	p.wg.Wait()
}
