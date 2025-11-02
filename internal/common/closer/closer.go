// Package closer Для graceful shutdown
package closerutil

import (
	"context"
	"fmt"
	reflectutil "github.com/s-turchinskiy/metrics/internal/common/reflect"
	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	"log"
	"strings"
	"sync"
	"time"
)

type Closer struct {
	mu      sync.Mutex
	funcs   []Func
	timeout time.Duration
}

func New(timeout time.Duration) *Closer {
	return &Closer{timeout: timeout}
}
func (c *Closer) Add(f Func) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.funcs = append(c.funcs, f)
}

func (c *Closer) close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var (
		msgs     = make([]string, 0, len(c.funcs))
		complete = make(chan struct{}, 1)
	)

	go func() {
		for _, f := range c.funcs {
			logger.Log.Debugw("stopping " + reflectutil.GetFunctionName(f))
			if err := f(ctx); err != nil {
				msgs = append(msgs, fmt.Sprintf("[!] %v", err))
			}
		}

		complete <- struct{}{}
	}()

	select {
	case <-complete:
		break
	case <-ctx.Done():
		return fmt.Errorf("shutdown cancelled: %v", ctx.Err())
	}

	if len(msgs) > 0 {
		return fmt.Errorf(
			"shutdown finished with error(s): \n%s",
			strings.Join(msgs, "\n"),
		)
	}

	return nil
}

func (c *Closer) Shutdown() error {

	log.Println("shutting down server gracefully")
	logger.Log.Info("shutting down server gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	if err := c.close(shutdownCtx); err != nil {
		return fmt.Errorf("closer: %v", err)
	}

	return nil

}

func (c *Closer) ProcessingErrorsChannel(errorsCh chan error) {
	err := <-errorsCh

	if err == nil {
		return
	}

	logger.Log.Infow("error, server stopped", "error", err.Error())
	errShutdown := c.Shutdown()
	if errShutdown != nil {
		logger.Log.Fatalw("fatal error", "error", err.Error(), "error shutdown", errShutdown.Error())
	} else {

		logger.Log.Fatalw("fatal error", "error", err.Error())
	}
}

type Func func(ctx context.Context) error
