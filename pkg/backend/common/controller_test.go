package common

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/highlight-apps/node-backend/logging"
)

func init() {
	stopTimeout = 10 * time.Millisecond
}

func TestNewProcessController(t *testing.T) {
	logger := logging.NewStdLogger()
	controller := NewProcessController(logger)

	if controller.logger != logger {
		t.Errorf("expected logger to be set")
	}
	if controller.logs == nil {
		t.Errorf("expected logs slice to be initialized")
	}
	if len(controller.logs) != 0 {
		t.Errorf("expected logs slice to be empty initially")
	}
}

func TestProcessController_SetOnStop(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())

	called := false
	onStopFunc := func() {
		called = true
	}

	b.SetOnStop(onStopFunc)

	if b.onStop == nil {
		t.Error("expected onStop function to be set")
	}

	if b.onStop != nil {
		b.onStop()
	}

	if !called {
		t.Error("expected onStop function to be called")
	}
}

func TestProcessController_SetOnStop_Nil(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())

	b.SetOnStop(nil)

	if b.onStop != nil {
		t.Error("expected onStop function to be nil")
	}
}

func TestProcessController_SetupCmd_CapturesStdoutAndStderr(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	ch := b.SubscribeLogs(context.Background())

	err := b.SetupCmd(exec.Command("sh", "-c", "echo out; echo err 1>&2"))
	if err != nil {
		t.Fatalf("SetupCmd failed: %v", err)
	}

	var got []string
	for len(got) < 2 {
		select {
		case line := <-ch:
			got = append(got, line)
		case <-time.After(1 * time.Second):
			t.Fatalf("timeout waiting for logs: got %v", got)
		}
	}

	foundOut, foundErr := false, false
	for _, line := range got {
		if line == "out" {
			foundOut = true
		}
		if line == "err" {
			foundErr = true
		}
	}
	if !foundOut || !foundErr {
		t.Errorf("expected both stdout and stderr, got %v", got)
	}

	buf := b.GetBuffer()
	if len(buf) != 2 {
		t.Errorf("expected buffer size 2, got %d", len(buf))
	}
}

func TestProcessController_SetupCmd_AlreadyRunning(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	defer func() {
		if b.IsRunning() {
			if err := b.Stop(); err != nil {
				t.Errorf("Cleanup Stop failed: %v", err)
			}
		}
	}()
	err := b.SetupCmd(exec.Command("sh", "-c", "sleep 1"))
	if err != nil {
		t.Fatalf("expected no error on first SetupCmd, got %v", err)
	}
	err2 := b.SetupCmd(exec.Command("echo"))
	if !errors.Is(err2, ErrProcessAlreadyRunning) {
		t.Errorf("expected ErrProcessAlreadyRunning, got %v", err2)
	}
}

func TestProcessController_SetupCmd_InvalidCommand(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	err := b.SetupCmd(exec.Command("nonexistent_cmd_xyz"))
	if err == nil {
		t.Error("expected error for invalid command")
	}
}

func TestProcessController_SetupCmd_ChannelClosedAfterProcessExit(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	ch, _ := b.subscribe()
	if err := b.SetupCmd(exec.Command("sh", "-c", "echo one; echo two")); err != nil {
		t.Fatalf("SetupCmd failed: %v", err)
	}
	got := make([]string, 0, 2)
	for {
		line, ok := <-ch
		if !ok {
			break
		}
		got = append(got, line)
	}
	if len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Errorf("expected [one two], got %v", got)
	}
}

func TestProcessController_SetupCmd_ErrorOnStdoutPipe(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	cmd := exec.Command("echo stdout")
	cmd.Stdout = os.Stdout
	err := b.SetupCmd(cmd)
	if err == nil || !strings.Contains(err.Error(), "failed to get stdout pipe") {
		t.Errorf("expected stdout pipe error, got %v", err)
	}
}

func TestProcessController_SetupCmd_ErrorOnStderrPipe(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	cmd := exec.Command("echo stderr")
	cmd.Stderr = os.Stderr
	err := b.SetupCmd(cmd)
	if err == nil || !strings.Contains(err.Error(), "failed to get stderr pipe") {
		t.Errorf("expected stderr pipe error, got %v", err)
	}
}

func TestProcessController_IsRunning_NotRunning(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	if b.IsRunning() {
		t.Error("expected not running")
	}
}

func TestProcessController_IsRunning_Running(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	defer func() {
		if b.IsRunning() {
			if err := b.Stop(); err != nil {
				t.Errorf("Cleanup Stop failed: %v", err)
			}
		}
	}()
	err := b.SetupCmd(exec.Command("sh", "-c", "sleep 1"))
	if err != nil {
		t.Fatalf("SetupCmd failed: %v", err)
	}
	if !b.IsRunning() {
		t.Error("expected running after SetupCmd")
	}
}

func TestProcessController_Restart_NotRunning(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	err := b.Restart(func() error {
		t.Fatal("doStart should not be called when not running")
		return nil
	})
	if !errors.Is(err, ErrProcessNotRunning) {
		t.Errorf("expected ErrProcessNotRunning, got %v", err)
	}
}

func TestProcessController_Restart_Success(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	err := b.SetupCmd(exec.Command("sh", "-c", "sleep 1"))
	if err != nil {
		t.Fatalf("SetupCmd failed: %v", err)
	}
	called := false
	doStart := func() error {
		called = true
		return nil
	}
	if err := b.Restart(doStart); err != nil {
		t.Fatalf("Restart failed: %v", err)
	}
	if !called {
		t.Error("expected doStart to be called")
	}
}

func TestProcessController_Restart_AlreadyRestarting(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	err := b.SetupCmd(exec.Command("sh", "-c", "sleep 1"))
	if err != nil {
		t.Fatalf("SetupCmd failed: %v", err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	doStart := func() error {
		wg.Done()
		time.Sleep(50 * time.Millisecond)
		return nil
	}
	go func() {
		if err := b.Restart(doStart); err != nil {
			t.Errorf("first Restart failed: %v", err)
		}
	}()
	wg.Wait()
	err2 := b.Restart(doStart)
	if !errors.Is(err2, ErrProcessAlreadyRestarting) {
		t.Errorf("expected ErrProcessAlreadyRestarting, got %v", err2)
	}
}

func TestProcessController_Reload_NotRunning(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	if err := b.Reload(syscall.SIGHUP); err == nil {
		t.Error("expected error on Reload when not running")
	}
}

func TestProcessController_Reload_Success(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	err := b.SetupCmd(exec.Command("sh", "-c", "sleep 1"))
	if err != nil {
		t.Fatalf("SetupCmd failed: %v", err)
	}
	defer func() {
		if b.IsRunning() {
			if err := b.Stop(); err != nil {
				t.Errorf("Cleanup Stop failed: %v", err)
			}
		}
	}()
	if err := b.Reload(syscall.SIGHUP); err != nil {
		t.Errorf("Reload failed: %v", err)
	}
}

func TestProcessController_Reload_ErrorOnSignal(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	b.Cmd = &exec.Cmd{Process: &os.Process{Pid: -1}}
	err := b.Reload(syscall.SIGHUP)
	if err == nil {
		t.Error("expected error on Reload when Signal fails")
	}
}

func TestProcessController_Stop_NotRunning(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	err := b.Stop()
	if !errors.Is(err, ErrProcessNotRunning) {
		t.Errorf("expected ErrProcessNotRunning, got %v", err)
	}
}

func TestProcessController_Stop_Success(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	err := b.SetupCmd(exec.Command("sh", "-c", "sleep 2"))
	if err != nil {
		t.Fatalf("SetupCmd failed: %v", err)
	}
	if !b.IsRunning() {
		t.Fatal("expected process to be running")
	}
	if err := b.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if b.IsRunning() {
		t.Error("expected process to be stopped")
	}
}

func TestProcessController_Stop_KillAfterTimeout(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	cmd := exec.Command("sh", "-c", "trap '' TERM; sleep 1")
	if err := b.SetupCmd(cmd); err != nil {
		t.Fatalf("SetupCmd failed: %v", err)
	}
	if !b.IsRunning() {
		t.Fatal("expected process to be running")
	}
	if err := b.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if b.IsRunning() {
		t.Error("expected process to be stopped after timeout kill")
	}
}

func TestProcessController_Stop_ErrorOnSignalAndKill(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	b.Cmd = &exec.Cmd{Process: &os.Process{Pid: -1}}
	if err := b.Stop(); err != nil {
		t.Errorf("expected no error on Stop with bad process, got %v", err)
	}
}

func TestProcessController_GetBuffer_CropsOldEntries(t *testing.T) {
	old := logsLimit
	logsLimit = 3
	defer func() { logsLimit = old }()

	b := NewProcessController(logging.NewStdLogger())
	b.mu.Lock()
	for i := range 5 {
		b.logs = append(b.logs, fmt.Sprintf("l%d", i))
		if len(b.logs) > logsLimit {
			b.logs = b.logs[1:]
		}
	}
	b.mu.Unlock()

	buf := b.GetBuffer()
	if len(buf) != 3 {
		t.Errorf("expected buffer size 3, got %d", len(buf))
	}
	want := []string{"l2", "l3", "l4"}
	for i, v := range want {
		if buf[i] != v {
			t.Errorf("at %d: expected %s, got %s", i, v, buf[i])
		}
	}
}

func TestProcessController_SubscribeLogs_UnsubscribeClosesChannel(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	ctx, cancel := context.WithCancel(context.Background())
	ch := b.SubscribeLogs(ctx)
	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel closed after cancel")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for channel close")
	}
}

// Private methods tests

type errorReader struct{}

func (errorReader) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("read error")
}

type scannerErrorReader struct {
	data     string
	readOnce bool
}

func (r *scannerErrorReader) Read(p []byte) (int, error) {
	if !r.readOnce {
		r.readOnce = true
		copy(p, r.data)
		return len(r.data), nil
	}
	return 0, fmt.Errorf("scanner error")
}

type stringReaderWithLines struct {
	lines []string
	pos   int
}

func newStringReaderWithLines(lines []string) *stringReaderWithLines {
	return &stringReaderWithLines{
		lines: lines,
	}
}

func (r *stringReaderWithLines) Read(p []byte) (int, error) {
	if r.pos >= len(r.lines) {
		return 0, io.EOF
	}

	line := r.lines[r.pos]
	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}

	n := copy(p, line)
	r.pos++
	return n, nil
}

func TestProcessController_captureProcessLogs_ErrorReading(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	ch, _ := b.subscribe()
	b.captureProcessLogs(errorReader{}, errorReader{})
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after error in readers")
	}
}

func TestProcessController_captureProcessLogs_ScannerError(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())
	ctx := t.Context()

	ch := b.SubscribeLogs(ctx)

	smallBufCh := make(chan string, 1)
	b.mu.Lock()
	b.subscribers = append(b.subscribers, smallBufCh)
	b.mu.Unlock()

	stdout := &scannerErrorReader{data: "test stdout data\n"}
	stderr := &scannerErrorReader{data: "test stderr data\n"}

	cmd := exec.Command("sleep", "0.1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	b.cmdMu.Lock()
	b.Cmd = cmd
	b.cmdMu.Unlock()

	go b.captureProcessLogs(stdout, stderr)

	receivedLines := 0
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				if receivedLines < 2 {
					t.Errorf("expected to receive at least 2 lines, got %d", receivedLines)
				}

				b.cmdMu.Lock()
				if b.Cmd != nil {
					t.Error("expected Cmd to be nil after scanner error")
				}
				b.cmdMu.Unlock()

				b.mu.Lock()
				if len(b.subscribers) != 0 {
					t.Errorf("expected subscribers to be empty, got %d", len(b.subscribers))
				}
				b.mu.Unlock()

				return
			}
			receivedLines++
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for logs or channel close")
		}
	}
}

func TestProcessController_captureProcessLogs_WaitCmdExit(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())

	ch, _ := b.subscribe()

	cmd := exec.Command("echo", "test")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	b.cmdMu.Lock()
	b.Cmd = cmd
	b.cmdMu.Unlock()

	stdout := strings.NewReader("")
	stderr := strings.NewReader("")

	go b.captureProcessLogs(stdout, stderr)

	_, ok := <-ch
	for ok {
		_, ok = <-ch
	}

	b.cmdMu.Lock()
	defer b.cmdMu.Unlock()
	if b.Cmd != nil {
		t.Error("expected Cmd to be nil after process exit")
	}
}

func TestProcessController_captureProcessLogs_LogsTruncation(t *testing.T) {
	oldLimit := logsLimit
	logsLimit = 2
	defer func() { logsLimit = oldLimit }()

	b := NewProcessController(logging.NewStdLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := b.SubscribeLogs(ctx)

	lines := []string{"line1", "line2", "line3"}
	stdout := newStringReaderWithLines(lines)
	stderr := strings.NewReader("")

	cmd := exec.Command("echo", "test")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	b.cmdMu.Lock()
	b.Cmd = cmd
	b.cmdMu.Unlock()

	go b.captureProcessLogs(stdout, stderr)

	var received []string
	for range lines {
		select {
		case line := <-ch:
			received = append(received, line)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for logs")
		}
	}

	if len(received) != len(lines) {
		t.Errorf("expected %d lines, got %d", len(lines), len(received))
	}

	buf := b.GetBuffer()
	if len(buf) != logsLimit {
		t.Errorf("expected buffer size %d, got %d", logsLimit, len(buf))
	}

	expected := []string{"line2", "line3"}
	for i, line := range expected {
		if i < len(buf) && buf[i] != line {
			t.Errorf("expected buffer[%d] = %q, got %q", i, line, buf[i])
		}
	}

	select {
	case <-time.After(time.Second):
		if b.IsRunning() {
			if err := b.Stop(); err != nil {
				t.Errorf("Cleanup Stop failed: %v", err)
			}
		}
	}
}

func TestProcessController_OnStopCallback_CalledOnProcessExit(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())

	called := false
	b.SetOnStop(func() {
		called = true
	})

	ch, _ := b.subscribe()

	cmd := exec.Command("echo", "test")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	b.cmdMu.Lock()
	b.Cmd = cmd
	b.cmdMu.Unlock()

	stdout := strings.NewReader("test output\n")
	stderr := strings.NewReader("")

	go b.captureProcessLogs(stdout, stderr)

	_, ok := <-ch
	for ok {
		_, ok = <-ch
	}

	if !called {
		t.Error("expected onStop callback to be called when process exits")
	}
}

func TestProcessController_OnStopCallback_NotCalledWhenNil(t *testing.T) {
	b := NewProcessController(logging.NewStdLogger())

	ch, _ := b.subscribe()

	cmd := exec.Command("echo", "test")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	b.cmdMu.Lock()
	b.Cmd = cmd
	b.cmdMu.Unlock()

	stdout := strings.NewReader("test output\n")
	stderr := strings.NewReader("")

	go b.captureProcessLogs(stdout, stderr)

	_, ok := <-ch
	for ok {
		_, ok = <-ch
	}
}
