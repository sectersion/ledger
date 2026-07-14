// Package queue is the in-memory FIFO task queue: a channel-backed worker
// pool with a concurrency cap, plus per-task phase state for the DAG.
// Queued-but-unspawned tasks are the one piece of state that isn't durable
// (nothing lost on crash, since they have no worktree/journal yet).
package queue

import (
	"context"
	"sync"
)

// Task is one unit of work. Run does the actual work; ID and Phase are
// bookkeeping for DAG state.
type Task struct {
	ID    string
	Phase string
	Run   func(ctx context.Context) error
}

// Queue is a FIFO task queue with a fixed concurrency cap.
type Queue struct {
	concurrency int

	mu     sync.Mutex
	phases map[string]string
}

// New creates a queue that runs at most concurrency tasks at once.
func New(concurrency int) *Queue {
	return &Queue{concurrency: concurrency, phases: map[string]string{}}
}

// Phase returns the current phase of task id, or "" if unknown.
func (q *Queue) Phase(id string) string {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.phases[id]
}

func (q *Queue) setPhase(id, phase string) {
	q.mu.Lock()
	q.phases[id] = phase
	q.mu.Unlock()
}

// Run drains tasks in FIFO order, running at most q.concurrency at a time,
// and returns each task's error keyed by task ID. It returns once every
// task submitted (tasks channel closed) has finished.
func (q *Queue) Run(ctx context.Context, tasks <-chan Task) map[string]error {
	sem := make(chan struct{}, q.concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	errs := map[string]error{}

	for t := range tasks {
		t := t
		q.setPhase(t.ID, t.Phase)
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			err := t.Run(ctx)
			mu.Lock()
			errs[t.ID] = err
			mu.Unlock()
		}()
	}
	wg.Wait()
	return errs
}
