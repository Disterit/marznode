package hysteria2

import (
	"bufio"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
)

type Hysteria struct {
	executablePath string
	process        *exec.Cmd
	logChan        chan []byte
	mu             sync.Mutex
	running        bool
}

func NewHysteria(executablePath string) *Hysteria {
	return &Hysteria{
		executablePath: executablePath,
		logChan:        make(chan []byte, 100),
	}
}

func (h *Hysteria) Start(configYAML []byte) error {
	tempFile, err := os.CreateTemp("", "hysteria2-*.yaml")
	if err != nil {
		return err
	}
	defer tempFile.Close()

	_, err = tempFile.Write(configYAML)
	if err != nil {
		return err
	}

	h.process = exec.Command(h.executablePath, "server", "-c", tempFile.Name())

	stdout, err := h.process.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := h.process.StderrPipe()
	if err != nil {
		return err
	}

	if err := h.process.Start(); err != nil {
		return err
	}

	h.running = true
	go h.captureLogs(stdout)
	go h.captureLogs(stderr)

	log.Println("Starting hysteria server")
	return nil
}

func (h *Hysteria) captureLogs(pipe io.ReadCloser) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)

		select {
		case h.logChan <- line:
		default:

		}
	}
	log.Println("Hysteria process stopped")
	h.running = false
	close(h.logChan)
}

func (h *Hysteria) GetLogChannel() <-chan []byte {
	return h.logChan
}

func (h *Hysteria) Stop() {
	if h.running && h.process != nil {
		_ = h.process.Process.Kill()
		h.running = false
		close(h.logChan)
	}
}
