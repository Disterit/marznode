package common

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/highlight-apps/node-backend/logging"
)

var logsLimit = 100
var stopTimeout = 3 * time.Second
var stopTimer = time.NewTimer

type ProcessController interface {
	SetupCmd(cmd *exec.Cmd) error
	IsRunning() bool
	Restart(doStart func() error) error
	Reload(signal syscall.Signal) error
	Stop() error
	GetBuffer() []string
	SubscribeLogs(ctx context.Context) <-chan string
	SetOnStop(fn func())
}

type BaseProcessController struct {
	logger      logging.Logger
	mu          sync.Mutex
	logs        []string
	subscribers []chan string
	cmdMu       sync.Mutex
	Cmd         *exec.Cmd
	restartMu   sync.Mutex
	restarting  bool
	onStop      func()
}

var _ ProcessController = (*BaseProcessController)(nil)

func NewProcessController(logger logging.Logger) *BaseProcessController {
	return &BaseProcessController{
		logger: logger,
		logs:   make([]string, 0, logsLimit),
	}
}

func (c *BaseProcessController) SetOnStop(fn func()) {
	c.onStop = fn
}

func (c *BaseProcessController) SetupCmd(cmd *exec.Cmd) error {
	if c.IsRunning() {
		c.logger.Error("process already running")
		return ErrProcessAlreadyRunning
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	go c.captureProcessLogs(stdout, stderr)

	c.cmdMu.Lock()
	c.Cmd = cmd
	c.cmdMu.Unlock()

	c.logger.Info("process started")
	return nil
}

func (c *BaseProcessController) IsRunning() bool {
	c.cmdMu.Lock()
	defer c.cmdMu.Unlock()
	return c.Cmd != nil && c.Cmd.Process != nil
}

func (c *BaseProcessController) Restart(doStart func() error) error {
	c.restartMu.Lock()
	if c.restarting {
		c.restartMu.Unlock()
		c.logger.Error("process already restarting")
		return ErrProcessAlreadyRestarting
	}
	c.restarting = true
	c.restartMu.Unlock()
	defer func() {
		c.restartMu.Lock()
		c.restarting = false
		c.restartMu.Unlock()
	}()

	if err := c.Stop(); err != nil {
		return err
	}
	c.logger.Info("process restarted")
	return doStart()
}

func (c *BaseProcessController) Reload(signal syscall.Signal) error {
	c.cmdMu.Lock()
	defer c.cmdMu.Unlock()
	if c.Cmd == nil || c.Cmd.Process == nil {
		return fmt.Errorf("process not running")
	}
	if err := c.Cmd.Process.Signal(signal); err != nil {
		return fmt.Errorf("failed to reload process: %w", err)
	}
	return nil
}

func (c *BaseProcessController) Stop() error {
	c.cmdMu.Lock()
	cmdCopy := c.Cmd
	c.cmdMu.Unlock()
	if cmdCopy == nil || cmdCopy.Process == nil {
		c.logger.Error("process not running")
		return ErrProcessNotRunning
	}
	if err := cmdCopy.Process.Signal(syscall.SIGTERM); err != nil {
		c.logger.Error("failed to send SIGTERM to process:", err)
	}

	t := stopTimer(stopTimeout)
	defer t.Stop()
	<-t.C

	if err := cmdCopy.Process.Kill(); err != nil {
		c.logger.Debug("failed to kill process after timeout:", err)
	}
	c.cmdMu.Lock()
	if c.Cmd == cmdCopy {
		c.Cmd = nil
	}
	c.cmdMu.Unlock()
	c.logger.Info("process stopped")
	return nil
}

func (c *BaseProcessController) GetBuffer() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	bufCopy := make([]string, len(c.logs))
	copy(bufCopy, c.logs)
	return bufCopy
}

func (c *BaseProcessController) SubscribeLogs(ctx context.Context) <-chan string {
	ch, cancel := c.subscribe()
	go func() {
		<-ctx.Done()
		cancel()
	}()
	return ch
}

func (c *BaseProcessController) subscribe() (<-chan string, func()) {
	ch := make(chan string, 100)
	c.mu.Lock()
	c.subscribers = append(c.subscribers, ch)
	c.mu.Unlock()
	return ch, func() { c.unsubscribe(ch) }
}

func (c *BaseProcessController) unsubscribe(ch <-chan string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, sub := range c.subscribers {
		if sub == ch {
			c.subscribers = append(c.subscribers[:i], c.subscribers[i+1:]...)
			close(sub)
			break
		}
	}
}

func (c *BaseProcessController) captureProcessLogs(stdout, stderr io.Reader) {
	processLine := func(line string) {
		c.logger.Info(line)
		c.mu.Lock()
		c.logs = append(c.logs, line)
		if len(c.logs) > logsLimit {
			c.logs = c.logs[1:]
		}
		subs := make([]chan string, len(c.subscribers))
		copy(subs, c.subscribers)
		c.mu.Unlock()
		for _, ch := range subs {
			select {
			case ch <- line:
			default:
			}
		}
	}
	scannerOut := bufio.NewScanner(stdout)
	scannerErr := bufio.NewScanner(stderr)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for scannerOut.Scan() {
			processLine(scannerOut.Text())
		}
		if err := scannerOut.Err(); err != nil {
			c.logger.Error("error scanning stdout:", err)
		}
	}()
	go func() {
		defer wg.Done()
		for scannerErr.Scan() {
			processLine(scannerErr.Text())
		}
		if err := scannerErr.Err(); err != nil {
			c.logger.Error("error scanning stderr:", err)
		}
	}()
	wg.Wait()
	c.cmdMu.Lock()
	cmdCopy := c.Cmd
	c.cmdMu.Unlock()
	if cmdCopy != nil {
		cmdCopy.Wait()
		c.cmdMu.Lock()
		if c.Cmd == cmdCopy {
			c.Cmd = nil
		}
		c.cmdMu.Unlock()
	}
	c.logger.Warn("process stopped/died")
	if c.onStop != nil {
		c.onStop()
	}
	c.mu.Lock()
	for _, ch := range c.subscribers {
		close(ch)
	}
	c.subscribers = nil
	c.mu.Unlock()
}
