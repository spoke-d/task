package conjoint

import (
	"context"
	"sync"
	"time"
)

type Conjoint struct {
	left, right context.Context
	mutext      sync.Mutex    // protects timer and err
	done        chan struct{} // chan closed on cancelFunc() call, or parent done
	once        sync.Once     // protects cancel func
	err         error         // err set on cancel or timeout
}

// Context takes two Contexts and combines them into a pair, conjoining their
// behavior:
//
//  - If either parent context is canceled, the Conjoint is canceled. The err is
//  set to whatever the err of the parent that was canceled.
//  - If either parent has a deadline, the Conjoint uses that same deadline. If
//  both have a deadline, it uses the sooner/lesser one.
//  - Values from both parents are unioned together. When a key is present in
//  both parent trees, the left (first) context supercedes the right (second).
func Context(left, right context.Context) (context.Context, context.CancelFunc) {
	c := &Conjoint{
		left:  left,
		right: right,
		done:  make(chan struct{}),
	}

	if c.left.Done() == nil && c.right.Done() == nil {
		// Both parents are un-cancelable, so it's more technically correct to
		// return a no-op func here.
		return c, func() {}
	}

	if c.left.Err() != nil {
		c.cancel(c.left.Err())
		return c, func() {}
	}
	if c.right.Err() != nil {
		c.cancel(c.right.Err())
		return c, func() {}
	}

	go func() {
		select {
		case <-c.left.Done():
			c.cancel(c.left.Err())
		case <-c.right.Done():
			c.cancel(c.right.Err())
		case <-c.done:
			// Ensure the goroutine dies when canceled
		}
	}()

	return c, func() { c.cancel(context.Canceled) }
}

func (c *Conjoint) cancel(err error) {
	c.once.Do(func() {
		if err == nil {
			panic("Conjoint: internal error: missing cancel error")
		}

		c.mutext.Lock()
		if c.err == nil {
			c.err = err
			close(c.done)
		}
		c.mutext.Unlock()
	})
}

func (c *Conjoint) Deadline() (time.Time, bool) {
	leftDeadline, hok := c.left.Deadline()
	rightDeadline, tok := c.right.Deadline()
	if !hok && !tok {
		return time.Time{}, false
	}

	if hok && !tok {
		return leftDeadline, true
	}
	if !hok && tok {
		return rightDeadline, true
	}

	if leftDeadline.Before(rightDeadline) {
		return leftDeadline, true
	}
	return rightDeadline, true
}

func (c *Conjoint) Done() <-chan struct{} {
	return c.done
}

func (c *Conjoint) Err() error {
	c.mutext.Lock()
	defer c.mutext.Unlock()

	return c.err
}

func (c *Conjoint) Value(key interface{}) interface{} {
	v := c.left.Value(key)
	if v != nil {
		return v
	}
	return c.right.Value(key)
}
