package utils

import (
	"context"
	"sync"
)

// WorkerPool 使用信号量(Semaphore)模式控制并发数
// workers channel 作为信号量，控制同时执行的任务数量
type WorkerPool[T any] struct {
	workers   chan struct{} // 信号量：控制最大并发数
	workQueue chan T
	handler   func(T)
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
}

func NewWorkerPool[T any](ctx context.Context, size int, handler func(T)) *WorkerPool[T] {
	ctx, cancel := context.WithCancel(ctx)
	pool := &WorkerPool[T]{
		workers:   make(chan struct{}, size),
		workQueue: make(chan T, size*2),
		handler:   handler,
		ctx:       ctx,
		cancel:    cancel,
	}

	// Initialize workers
	for range size {
		pool.workers <- struct{}{}
	}

	// Start worker goroutines
	for range size {
		pool.wg.Add(1)
		go pool.worker()
	}

	return pool
}

func (p *WorkerPool[T]) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case work, ok := <-p.workQueue:
			if !ok {
				return
			}
			<-p.workers // Acquire worker slot
			p.handler(work)
			p.workers <- struct{}{} // Release worker slot
		}
	}
}

func (p *WorkerPool[T]) Submit(work T) bool {
	select {
	case p.workQueue <- work:
		return true
	case <-p.ctx.Done():
		return false
	default:
		return false
	}
}

func (p *WorkerPool[T]) SubmitBlock(work T) bool {
	select {
	case p.workQueue <- work:
		return true
	case <-p.ctx.Done():
		return false
	}
}

func (p *WorkerPool[T]) Close() {
	p.cancel()
	p.wg.Wait()

	p.closeOnce.Do(func() {
		close(p.workQueue)
	})
}
