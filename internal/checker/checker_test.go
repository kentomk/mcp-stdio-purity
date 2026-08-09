package checker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestValidateRecord(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		data   string
		reason string
		id     string
	}{
		{name: "notification", data: `{"jsonrpc":"2.0","method":"notifications/initialized"}`},
		{name: "request", data: `{"jsonrpc":"2.0","id":"server-1","method":"roots/list"}`},
		{name: "success response", data: `{"jsonrpc":"2.0","id":1,"result":{}}`, id: "1"},
		{name: "error response", data: `{"jsonrpc":"2.0","id":2,"error":{"code":-1,"message":"no"}}`, id: "2"},
		{name: "plain text", data: "SECRET_CANARY", reason: "invalid-json"},
		{name: "json scalar", data: `42`, reason: "invalid-json"},
		{name: "not rpc", data: `{"status":"ok"}`, reason: "not-jsonrpc-envelope"},
		{name: "mixed envelope", data: `{"jsonrpc":"2.0","id":1,"method":"x","result":{}}`, reason: "not-jsonrpc-envelope"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info := validateRecord(record{data: []byte(test.data), line: 1})
			if info.reason != test.reason || info.responseID != test.id {
				t.Fatalf("got reason=%q id=%q, want reason=%q id=%q", info.reason, info.responseID, test.reason, test.id)
			}
		})
	}
	for _, test := range []struct {
		name   string
		record record
		reason string
	}{
		{name: "invalid UTF-8", record: record{data: []byte{0xff}}, reason: "invalid-utf8"},
		{name: "empty line", record: record{data: []byte{}}, reason: "invalid-json"},
		{name: "multiple values", record: record{data: []byte(`{} {}`)}, reason: "invalid-json"},
		{name: "unterminated EOF", record: record{data: []byte(`{"jsonrpc":"2.0","method":"ping"}`), unterminated: true}, reason: "unterminated-line"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if info := validateRecord(test.record); info.reason != test.reason {
				t.Fatalf("reason=%q, want %q", info.reason, test.reason)
			}
		})
	}
}

func TestReadStreamCountsFragmentsAsOneRecord(t *testing.T) {
	data := append(bytes.Repeat([]byte{'x'}, 70*1024), '\n')
	out := make(chan streamEvent, 2)
	readStream(bytes.NewReader(data), 128*1024, 128*1024, out)
	first := <-out
	if first.record == nil || first.record.line != 1 {
		t.Fatalf("got first event=%+v, want record line 1", first)
	}
	if first.record.offset != 0 || len(first.record.data) != 70*1024 {
		t.Fatalf("got offset=%d bytes=%d, want offset=0 bytes=%d", first.record.offset, len(first.record.data), 70*1024)
	}
	if event := <-out; !event.eof {
		t.Fatalf("got final event=%+v, want EOF", event)
	}
}

func TestRunFixtures(t *testing.T) {
	tests := []struct {
		mode       string
		wantStatus string
		wantReason string
		wantCount  int
		wantPhase  string
		configure  func(*Config)
	}{
		{mode: "clean", wantStatus: "passed"},
		{mode: "capabilities", wantStatus: "passed"},
		{mode: "server-messages", wantStatus: "passed"},
		{mode: "startup-banner", wantStatus: "violations", wantReason: "invalid-json", wantPhase: "initialize"},
		{mode: "late-log", wantStatus: "violations", wantReason: "invalid-json", wantPhase: "probe"},
		{mode: "post-response-log", wantStatus: "violations", wantReason: "invalid-json", wantPhase: "cleanup"},
		{mode: "cleanup-child", wantStatus: "violations", wantReason: "invalid-json", wantPhase: "cleanup"},
		{mode: "many-invalid", wantStatus: "violations", wantReason: "invalid-json", wantCount: 20, wantPhase: "initialize"},
		{mode: "oversize", wantStatus: "error", configure: func(config *Config) { config.MaxLineBytes = 64 }},
		{mode: "oversize-unterminated", wantStatus: "error", configure: func(config *Config) { config.MaxLineBytes = 64 }},
		{mode: "total-limit", wantStatus: "error", configure: func(config *Config) { config.MaxLineBytes = 64; config.MaxStdoutBytes = 100 }},
		{mode: "hung", wantStatus: "error", configure: func(config *Config) { config.Timeout = 200 * time.Millisecond }},
		{mode: "cleanup-hold", wantStatus: "error", configure: func(config *Config) { config.CleanupGrace = 100 * time.Millisecond }},
		{mode: "initialize-error", wantStatus: "error"},
		{mode: "probe-error", wantStatus: "error"},
		{mode: "early-exit", wantStatus: "error"},
		{mode: "broken-pipe", wantStatus: "error"},
		{mode: "signal-termination", wantStatus: "error"},
	}
	for _, test := range tests {
		t.Run(test.mode, func(t *testing.T) {
			config := DefaultConfig([]string{os.Args[0], "-test.run=TestHelperProcess", "--", test.mode})
			config.Timeout = 3 * time.Second
			config.CleanupGrace = time.Second
			config.Stderr = &bytes.Buffer{}
			config.Env = append(os.Environ(), "MCP_STDIO_PURITY_HELPER=1")
			if test.configure != nil {
				test.configure(&config)
			}
			started := time.Now()
			report := Run(context.Background(), config)
			if time.Since(started) > 2*time.Second && (test.mode == "hung" || test.mode == "cleanup-hold") {
				t.Fatalf("bounded process cleanup took too long: %s report=%+v", time.Since(started), report)
			}
			if report.Status != test.wantStatus {
				t.Fatalf("status=%q lifecycle=%+v diagnostics=%+v", report.Status, report.Lifecycle, report.Diagnostics)
			}
			if test.wantReason == "" {
				if len(report.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %+v", report.Diagnostics)
				}
			} else {
				wantCount := test.wantCount
				if wantCount == 0 {
					wantCount = 1
				}
				if len(report.Diagnostics) != wantCount || report.Diagnostics[0].Reason != test.wantReason {
					t.Fatalf("diagnostics=%+v", report.Diagnostics)
				}
				if test.wantPhase != "" && report.Diagnostics[0].Phase != test.wantPhase {
					t.Fatalf("phase=%q, want %q", report.Diagnostics[0].Phase, test.wantPhase)
				}
			}
			encoded, err := json.Marshal(report)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(encoded, []byte("SECRET_CANARY")) || bytes.Contains(encoded, []byte("startup-banner")) || bytes.Contains(encoded, []byte("post-response-log")) {
				t.Fatalf("report leaked fixture content or command arguments: %s", encoded)
			}
		})
	}
}

func TestRunCommandNotFound(t *testing.T) {
	config := DefaultConfig([]string{"/mcp-stdio-purity-test/command-does-not-exist"})
	config.Stderr = &bytes.Buffer{}
	report := Run(context.Background(), config)
	if report.Status != "error" || report.Lifecycle.Error != "spawn command" || report.Lifecycle.ProcessExited {
		t.Fatalf("unexpected command-not-found report: %+v", report)
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("MCP_STDIO_PURITY_HELPER") != "1" {
		return
	}
	separator := 0
	for index, arg := range os.Args {
		if arg == "--" {
			separator = index
			break
		}
	}
	mode := "clean"
	if separator > 0 && separator+1 < len(os.Args) {
		mode = os.Args[separator+1]
	}
	if mode == "startup-banner" {
		fmt.Println("SECRET_CANARY startup")
	}
	if mode == "many-invalid" {
		for range 25 {
			fmt.Println("SECRET_CANARY repeated")
		}
	}
	if mode == "oversize" {
		fmt.Println(strings.Repeat("X", 128))
	}
	if mode == "oversize-unterminated" {
		fmt.Print(strings.Repeat("X", 128))
	}
	if mode == "total-limit" {
		for range 4 {
			fmt.Println(`{"jsonrpc":"2.0","method":"x"}`)
		}
	}
	if mode == "early-exit" {
		os.Exit(0)
	}
	if mode == "child-output" {
		time.Sleep(50 * time.Millisecond)
		fmt.Println("SECRET_CANARY descendant cleanup")
		os.Exit(0)
	}
	if mode == "child-hold" {
		time.Sleep(5 * time.Second)
		os.Exit(0)
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var request struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if json.Unmarshal(scanner.Bytes(), &request) != nil {
			continue
		}
		if string(request.ID) == "1" {
			if mode == "hung" {
				continue
			}
			if mode == "initialize-error" {
				fmt.Println(`{"jsonrpc":"2.0","id":1,"error":{"code":-1,"message":"synthetic"}}`)
				continue
			}
			capabilities := `{}`
			if mode == "capabilities" {
				capabilities = `{"tools":{},"resources":{},"prompts":{}}`
			}
			if mode == "broken-pipe" {
				_ = os.Stdin.Close()
			}
			fmt.Printf(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-11-25","capabilities":%s,"serverInfo":{"name":"fixture","version":"0"}}}`+"\n", capabilities)
			if mode == "broken-pipe" {
				time.Sleep(100 * time.Millisecond)
				os.Exit(0)
			}
			if mode == "signal-termination" {
				process, _ := os.FindProcess(os.Getpid())
				_ = process.Signal(os.Interrupt)
				time.Sleep(time.Second)
				os.Exit(1)
			}
			continue
		}
		if request.Method == "notifications/initialized" {
			if mode == "late-log" {
				fmt.Println("SECRET_CANARY late")
			}
			if mode == "server-messages" {
				fmt.Println(`{"jsonrpc":"2.0","method":"notifications/progress","params":{}}`)
				fmt.Println(`{"jsonrpc":"2.0","id":"server-1","method":"roots/list","params":{}}`)
				fmt.Println(`{"jsonrpc":"2.0","id":"server-result","error":{"code":-1,"message":"synthetic"}}`)
			}
			continue
		}
		if len(request.ID) > 0 {
			if mode == "probe-error" {
				fmt.Printf(`{"jsonrpc":"2.0","id":%s,"error":{"code":-1,"message":"synthetic"}}`+"\n", request.ID)
				continue
			}
			if mode == "capabilities" {
				expected := map[string]string{"2": "tools/list", "3": "resources/list", "4": "prompts/list"}
				if expected[string(request.ID)] != request.Method {
					fmt.Printf(`{"jsonrpc":"2.0","id":%s,"error":{"code":-1,"message":"wrong probe"}}`+"\n", request.ID)
					continue
				}
			}
			fmt.Printf(`{"jsonrpc":"2.0","id":%s,"result":{}}`+"\n", request.ID)
			if mode == "post-response-log" {
				fmt.Println("SECRET_CANARY after response")
			}
			if mode == "cleanup-child" {
				startHelperDescendant("child-output")
			}
			if mode == "cleanup-hold" {
				startHelperDescendant("child-hold")
			}
		}
	}
	os.Exit(0)
}

func startHelperDescendant(mode string) {
	command := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", mode)
	command.Env = append(os.Environ(), "MCP_STDIO_PURITY_HELPER=1")
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "start synthetic descendant")
	}
}
