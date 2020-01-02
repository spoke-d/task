package task

import (
	"testing"
	"time"
)

func TestEvery(t *testing.T) {
	fn := Every(time.Second)
	interval, err := fn()
	if err != nil {
		t.Errorf("expected err to be nil")
	}
	if expected, actual := time.Second, interval; expected != actual {
		t.Errorf("expected: %d, actual: %d", expected, actual)
	}
}

func TestEveryWithOption(t *testing.T) {
	fn := Every(time.Second, SkipFirst)
	interval, err := fn()

	if expected, actual := ErrSkip, err; expected != actual {
		t.Errorf("expected: %v, actual: %v", expected, actual)
	}
	if expected, actual := time.Second, interval; expected != actual {
		t.Errorf("expected: %d, actual: %d", expected, actual)
	}
}
