package xray

import (
	"testing"
)

func TestGetXrayVersion_Success(t *testing.T) {
	orig := runCombinedOutput
	defer func() { runCombinedOutput = orig }()
	runCombinedOutput = func(name string, args ...string) ([]byte, error) {
		return []byte("Xray 1.2.3"), nil
	}

	version, err := getXrayVersion("fake-path")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if version != "1.2.3" {
		t.Errorf("expected version '1.2.3', got %q", version)
	}
}

func TestGetXrayVersion_ExecError(t *testing.T) {
	_, err := getXrayVersion("/nonexistent/path")
	if err == nil {
		t.Fatal("expected exec error, got nil")
	}
}

func TestGetXrayVersion_UnmatchedOutput(t *testing.T) {
	orig := runCombinedOutput
	defer func() { runCombinedOutput = orig }()
	runCombinedOutput = func(name string, args ...string) ([]byte, error) {
		return []byte("invalid output"), nil
	}

	version, err := getXrayVersion("fake-path")
	if err == nil {
		t.Fatalf("expected error for unmatched output, got nil")
	}
	if version != "" {
		t.Errorf("expected empty version on error, got %q", version)
	}
}
