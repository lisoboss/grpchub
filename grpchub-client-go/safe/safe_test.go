package safe

import (
	"sync"
	"testing"
	"time"
)

type testSafeClose struct {
	SafeClose

	c chan int
}

func (s *testSafeClose) send() int {
	s.RLock()
	defer s.RUnlock()

	select {
	case <-s.Closed():
		return 0
	case s.c <- 1:
		return 1
	}
}

func (s *testSafeClose) recv() int {
	s.RLock()
	defer s.RUnlock()

	select {
	case <-s.Closed():
		return 0
	case r := <-s.c:
		return r
	}
}

func (s *testSafeClose) loopS() {
	for {
		if r := s.recv(); r == 0 {
			break
		}
	}
}

func (s *testSafeClose) loopR() {
	for {
		if r := s.send(); r == 0 {
			break
		}
	}
}

func newTestSafeClose() *testSafeClose {
	c := make(chan any, 16)
	return &testSafeClose{
		SafeClose: NewSafeClose(func() {
			close(c)
		}),
	}
}

func TestSafeClose_Stress(t *testing.T) {
	const goroutines = 1000
	var wg sync.WaitGroup

	sc := NewSafeClose()

	// 启动大量 goroutine 调用 RLock/RUnlock
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sc.RLock()
			time.Sleep(time.Millisecond) // 模拟操作
			sc.RUnlock()
		}()
	}

	// 启动并发添加关闭回调
	for i := 0; i < goroutines/10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sc.AddCloseCallbaks(func() {
				t.Logf("Callback %d executed", i)
			})
		}(i)
	}

	// 并发尝试关闭（只能成功一次）
	for i := 0; i < goroutines/10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sc.Close()
		}()
	}

	// 检查 Close 通知是否触发
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-sc.Closed()
		t.Log("Closed channel received")
	}()

	wg.Wait()
}

func BenchmarkSafeClose(b *testing.B) {

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			t := newTestSafeClose()
			go t.loopR()
			go t.loopS()
			t.Close()
		}
	})
}
