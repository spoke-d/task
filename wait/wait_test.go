package wait

import (
	"context"
	"testing"
	"time"
)

func TestWait(t *testing.T) {
	t.Parallel()

	t.Run("run", func(t *testing.T) {
		var called bool
		err := Wait(func(ctx context.Context) {
			called = true
		}, time.Millisecond*50)

		if err != nil {
			t.Error(err)
		}
		if expected, actual := true, called; expected != actual {
			t.Errorf("expected: %v, actual: %v", expected, actual)
		}
	})

	t.Run("run long process", func(t *testing.T) {
		var called bool
		err := Wait(func(ctx context.Context) {
			time.Sleep(time.Millisecond * 100)
			select {
			case <-ctx.Done():
				called = true
			case <-time.After(time.Millisecond * 100):
				t.Errorf("failed to wait for done")
			}
		}, time.Millisecond*50)

		if expected, actual := "timed out waiting for completion", err.Error(); expected != actual {
			t.Errorf("expected: %v, actual: %v", expected, actual)
		}
		time.Sleep(time.Millisecond * 200)
		if expected, actual := true, called; expected != actual {
			t.Errorf("expected: %v, actual: %v", expected, actual)
		}
	})
}
