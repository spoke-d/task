package task

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/spoke-d/task/conjoint"
	"github.com/spoke-d/task/guard"
)

// Group of tasks sharing the same lifecycle.
//
// All tasks in a group will be started and stopped at the same time.
type Group struct {
	tasks   []Task
	guard   *guard.Guard
	cancel  context.CancelFunc
	abort   chan struct{}
	running sync.Map
	clock   Clock
}

// NewGroup creates a Group with sane defaults.
func NewGroup() *Group {
	return &Group{
		guard:   guard.New(),
		running: sync.Map{},
		clock:   wallClock{},
	}
}

// Add a new task to the group, returning its index.
func (g *Group) Add(f Func, schedule Schedule) *Task {
	task := Task{
		clock:    g.clock,
		f:        f,
		schedule: schedule,
		reset:    make(chan struct{}, 16), // Buffered to not block senders
	}
	g.tasks = append(g.tasks, task)
	return &task
}

// Len returns the number of tasks with in the group.
func (g *Group) Len() int {
	return len(g.tasks)
}

// Start all the tasks in the group.
func (g *Group) Start(ctx context.Context) error {
	ctx, g.cancel = context.WithCancel(ctx)
	g.abort = make(chan struct{})

	if err := g.guard.Unlock(); err != nil {
		return err
	}

	for i, task := range g.tasks {
		g.running.Store(i, struct{}{})
		go func(i int, task Task) {
			g.guard.Visit(func(fnContext context.Context) error {
				c, _ := conjoint.Context(ctx, fnContext)
				task.loop(c)
				g.running.Delete(i)
				return nil
			}, g.abort)
		}(i, task)
	}
	return nil
}

// Stop all tasks in the group.
//
// This works by sending a cancellation signal to all tasks of the
// group and waiting for them to terminate.
//
// If a task is idle (i.e. not executing its task function) it will terminate
// immediately.
//
// If a task is busy executing its task function, the cancellation signal will
// propagate through the context passed to it, and the task will block waiting
// for the function to terminate.
//
// In case the given timeout expires before all tasks complete, this method
// exits immediately and returns an error, otherwise it returns nil.
func (g *Group) Stop(timeout time.Duration) error {
	if g.cancel == nil {
		return nil
	}

	close(g.abort)
	g.cancel()

	abort := make(chan struct{})

	lockErrors := make(chan error, 1)
	go func() { lockErrors <- g.guard.Lock(abort) }()

	// Wait for graceful termination, but abort if the context expires.
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		close(abort)

		results := 0
		g.running.Range(func(key, value interface{}) bool {
			results++
			return true
		})

		// Timeout if the guard didn't stop in time.
		if results != 0 {
			return errors.Errorf("tasks %d are still running", results)
		}
		return errors.Errorf("timed out attempting to stop")
	case err := <-lockErrors:
		close(lockErrors)
		return err
	}
}

// Kill asks the group to stop and returns immediately.
func (g *Group) Kill() error {
	if err := g.guard.Kill(); err != nil {
		return err
	}
	return g.guard.Wait()
}
