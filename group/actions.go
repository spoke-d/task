package group

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// Block creates a blocking executional group that waits until some other group
// action finishing, causing the closing of the group action.
func Block(g *Group) {
	cancel := make(chan struct{})
	g.Add(func(ctx context.Context) error {
		<-cancel
		return nil
	}, func(error) {
		close(cancel)
	})
}

// Interrupt creates a blocking executional group that becomes free if a
// interupt or terminate os signal is received.
func Interrupt(g *Group) {
	cancel := make(chan struct{})
	g.Add(func(ctx context.Context) error {
		return interrupt(cancel)
	}, func(error) {
		close(cancel)
	})
}

func interrupt(cancel <-chan struct{}) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-c:
		return fmt.Errorf("received signal %s", sig)
	case <-cancel:
		return errors.New("canceled")
	}
}
