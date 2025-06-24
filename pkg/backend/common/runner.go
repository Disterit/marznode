package common

import (
	"context"
	"sync"

	"github.com/highlight-apps/node-backend/logging"
)

type Runner interface {
	Version() (string, error)
	Start(config string) error
	IsRunning() bool
	Restart(config string) error
	Reload() error
	Stop() error
	SubscribeLogs(ctx context.Context) <-chan string
	GetBuffer() []string
}

type BaseRunner struct {
	Logger         logging.Logger
	Controller     ProcessController
	ExecutablePath string

	stopEventMu   sync.Mutex
	stopEvent     chan struct{}
	stopEventOnce sync.Once
}

var _ Runner = (*BaseRunner)(nil)

func NewBaseRunner(executablePath string, logger logging.Logger, controller ProcessController) *BaseRunner {
	return &BaseRunner{
		Logger:         logger,
		Controller:     controller,
		ExecutablePath: executablePath,
		stopEvent:      make(chan struct{}),
	}
}

func (b *BaseRunner) Version() (string, error) {
	panic("not implemented")
}

func (b *BaseRunner) Start(config string) error {
	panic("not implemented")
}

func (b *BaseRunner) IsRunning() bool {
	return b.Controller.IsRunning()
}

func (b *BaseRunner) Restart(config string) error {
	panic("not implemented")
}

func (b *BaseRunner) Reload() error {
	panic("not implemented")
}

func (b *BaseRunner) Stop() error {
	return b.Controller.Stop()
}

func (b *BaseRunner) SubscribeLogs(ctx context.Context) <-chan string {
	return b.Controller.SubscribeLogs(ctx)
}

func (b *BaseRunner) GetBuffer() []string {
	return b.Controller.GetBuffer()
}

func (b *BaseRunner) TriggerStopEvent() {
	b.stopEventOnce.Do(func() {
		b.stopEventMu.Lock()
		defer b.stopEventMu.Unlock()
		close(b.stopEvent)
	})
}

func (b *BaseRunner) ResetStopEvent() {
	b.stopEventMu.Lock()
	defer b.stopEventMu.Unlock()
	b.stopEvent = make(chan struct{})
	b.stopEventOnce = sync.Once{}
}

func (b *BaseRunner) StopEvent() <-chan struct{} {
	b.stopEventMu.Lock()
	defer b.stopEventMu.Unlock()
	return b.stopEvent
}

func (b *BaseRunner) SetupOnStopHandler() {
	b.Controller.SetOnStop(func() {
		b.TriggerStopEvent()
	})
}

func (b *BaseRunner) RestartWithCallback(doStart func() error) error {
	return b.Controller.Restart(doStart)
}
