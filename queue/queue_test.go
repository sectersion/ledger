package queue

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunRespectsConcurrencyCap(t *testing.T) {
	const cap, n = 3, 20
	q := New(cap)

	var current, max int64
	tasks := make(chan Task, n)
	for i := 0; i < n; i++ {
		i := i
		tasks <- Task{
			ID:    fmt.Sprintf("task-%d", i),
			Phase: "running",
			Run: func(ctx context.Context) error {
				c := atomic.AddInt64(&current, 1)
				for {
					m := atomic.LoadInt64(&max)
					if c <= m || atomic.CompareAndSwapInt64(&max, m, c) {
						break
					}
				}
				time.Sleep(5 * time.Millisecond)
				atomic.AddInt64(&current, -1)
				return nil
			},
		}
	}
	close(tasks)

	errs := q.Run(context.Background(), tasks)
	if len(errs) != n {
		t.Fatalf("got %d results, want %d", len(errs), n)
	}
	if max > cap {
		t.Fatalf("max concurrent = %d, want <= %d", max, cap)
	}
	if q.Phase("task-0") != "running" {
		t.Fatalf("phase = %q, want running", q.Phase("task-0"))
	}
}
