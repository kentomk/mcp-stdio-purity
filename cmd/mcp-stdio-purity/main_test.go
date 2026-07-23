package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var fixtureBinary string

func TestMain(m *testing.M) {
	temp, err := os.MkdirTemp("", "mcp-stdio-purity-cli-test-")
	if err != nil {
		panic(err)
	}
	fixtureBinary = filepath.Join(temp, "fixture-server")
	command := exec.Command("go", "build", "-trimpath", "-o", fixtureBinary, "../../examples/fixture-server")
	command.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	if output, buildErr := command.CombinedOutput(); buildErr != nil {
		_ = os.RemoveAll(temp)
		panic(string(output))
	}
	status := m.Run()
	_ = os.RemoveAll(temp)
	os.Exit(status)
}

func TestCLIGoldenExitContracts(t *testing.T) {
	tests := []struct {
		name   string
		format string
		mode   string
		status int
		golden string
	}{
		{name: "passed text", format: "text", mode: "clean", status: 0, golden: "passed-text.golden"},
		{name: "violations text", format: "text", mode: "startup-banner", status: 1, golden: "violations-text.golden"},
		{name: "error text", format: "text", mode: "command-not-found", status: 2, golden: "error-text.golden"},
		{name: "passed json", format: "json", mode: "clean", status: 0, golden: "passed-json.golden"},
		{name: "violations json", format: "json", mode: "startup-banner", status: 1, golden: "violations-json.golden"},
		{name: "error json", format: "json", mode: "command-not-found", status: 2, golden: "error-json.golden"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command := fixtureBinary
			arguments := []string{"--mode", test.mode}
			if test.mode == "command-not-found" {
				command = filepath.Join(filepath.Dir(fixtureBinary), "command-not-found")
				arguments = nil
			}
			args := []string{"check", "--format", test.format, "--timeout", "3s", "--cleanup-grace", "100ms", "--", command}
			args = append(args, arguments...)
			var stdout, stderr bytes.Buffer
			if status := run(args, &stdout, &stderr); status != test.status {
				t.Fatalf("exit=%d, want %d; stderr=%s; stdout=%s", status, test.status, stderr.String(), stdout.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("unexpected stderr: %s", stderr.String())
			}
			actual := strings.ReplaceAll(stdout.String(), filepath.Base(command), "COMMAND")
			want, err := os.ReadFile(filepath.Join("testdata", test.golden))
			if err != nil {
				t.Fatal(err)
			}
			if actual != string(want) {
				t.Fatalf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", test.golden, actual, want)
			}
		})
	}
}

func TestHelpIsSuccessfulAndUsesStdout(t *testing.T) {
	for _, args := range [][]string{
		{"help"},
		{"--help"},
		{"-h"},
		{"check", "--help"},
		{"check", "-h"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if status := run(args, &stdout, &stderr); status != 0 {
				t.Fatalf("exit=%d, want 0; stderr=%q", status, stderr.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("unexpected stderr: %q", stderr.String())
			}
			for _, expected := range []string{
				"mcp-stdio-purity checks a real stdio MCP server",
				"mcp-stdio-purity check [flags] -- COMMAND [ARG...]",
				"--format text|json",
				"MSP001",
			} {
				if !strings.Contains(stdout.String(), expected) {
					t.Fatalf("help is missing %q:\n%s", expected, stdout.String())
				}
			}
		})
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("synthetic broken pipe") }

func TestJSONOutputBrokenPipe(t *testing.T) {
	var stderr bytes.Buffer
	status := run([]string{"check", "--format", "json", "--timeout", "3s", "--", fixtureBinary, "--mode", "clean"}, brokenWriter{}, &stderr)
	if status != 2 || !strings.Contains(stderr.String(), "encode report") {
		t.Fatalf("exit=%d stderr=%q", status, stderr.String())
	}
}

var _ io.Writer = brokenWriter{}
