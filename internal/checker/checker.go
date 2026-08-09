package checker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	SchemaVersion         = 1
	RuleID                = "MSP001"
	defaultMaxDiagnostics = 20
)

type Config struct {
	Command        []string
	Timeout        time.Duration
	CleanupGrace   time.Duration
	MaxLineBytes   int
	MaxStdoutBytes int
	MaxDiagnostics int
	Stderr         io.Writer
	Env            []string
	ToolVersion    string
}

type Limits struct {
	TimeoutMillis      int64 `json:"timeoutMillis"`
	CleanupGraceMillis int64 `json:"cleanupGraceMillis"`
	MaxLineBytes       int   `json:"maxLineBytes"`
	MaxStdoutBytes     int   `json:"maxStdoutBytes"`
	MaxDiagnostics     int   `json:"maxDiagnostics"`
}

type Lifecycle struct {
	Initialized   bool   `json:"initialized"`
	ProbeComplete bool   `json:"probeComplete"`
	ProcessExited bool   `json:"processExited"`
	Error         string `json:"error,omitempty"`
}

type Diagnostic struct {
	RuleID      string `json:"ruleId"`
	Reason      string `json:"reason"`
	Phase       string `json:"phase"`
	Line        int    `json:"line"`
	ByteOffset  int64  `json:"byteOffset"`
	ByteCount   int    `json:"byteCount"`
	Message     string `json:"message"`
	Remediation string `json:"remediation"`
}

type Report struct {
	SchemaVersion        int          `json:"schemaVersion"`
	ToolVersion          string       `json:"toolVersion"`
	Command              string       `json:"command"`
	Status               string       `json:"status"`
	Lifecycle            Lifecycle    `json:"lifecycle"`
	Limits               Limits       `json:"limits"`
	Diagnostics          []Diagnostic `json:"diagnostics"`
	DiagnosticsTruncated bool         `json:"diagnosticsTruncated"`
}

type record struct {
	data         []byte
	line         int
	offset       int64
	unterminated bool
}

type streamEvent struct {
	record *record
	err    error
	eof    bool
}

type messageInfo struct {
	reason     string
	responseID string
	errorReply bool
}

func Run(parent context.Context, cfg Config) Report {
	report := baseReport(cfg)
	if err := validateConfig(cfg); err != nil {
		report.Status = "error"
		report.Lifecycle.Error = err.Error()
		return report
	}

	ctx, cancel := context.WithTimeout(parent, cfg.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, cfg.Command[0], cfg.Command[1:]...)
	configureCommand(cmd)
	if cfg.Env != nil {
		cmd.Env = cfg.Env
	}
	if cfg.Stderr != nil {
		cmd.Stderr = cfg.Stderr
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return operationalError(report, "create stdin pipe")
	}
	stdout, stdoutWriter, err := os.Pipe()
	if err != nil {
		return operationalError(report, "create stdout pipe")
	}
	cmd.Stdout = stdoutWriter
	if err := cmd.Start(); err != nil {
		_ = stdout.Close()
		_ = stdoutWriter.Close()
		return operationalError(report, "spawn command")
	}
	_ = stdoutWriter.Close()

	stream := make(chan streamEvent, 1)
	go readStream(stdout, cfg.MaxLineBytes, cfg.MaxStdoutBytes, stream)
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()

	phase := "initialize"
	if err := writeMessage(stdin, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"mcp-stdio-purity","version":"0"}}}`); err != nil {
		cancel()
		report.Lifecycle.Error = "write initialize request"
	}

	streamDone := false
	processDone := false
	stdinClosed := false
	pendingProbes := map[string]bool{}
	ctxDone := ctx.Done()
	var cleanupTimer *time.Timer
	var cleanupDone <-chan time.Time
	for !streamDone || !processDone {
		select {
		case event := <-stream:
			if event.record != nil {
				info := validateRecord(*event.record)
				if info.reason != "" {
					if len(report.Diagnostics) < cfg.MaxDiagnostics {
						report.Diagnostics = append(report.Diagnostics, newDiagnostic(info.reason, phase, *event.record))
					} else {
						report.DiagnosticsTruncated = true
					}
				}
				if info.responseID == "1" && !report.Lifecycle.Initialized {
					if info.errorReply {
						report.Lifecycle.Error = "server rejected initialize request"
						cancel()
						continue
					}
					report.Lifecycle.Initialized = true
					phase = "probe"
					if err := writeMessage(stdin, `{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`); err == nil {
						pendingProbes, err = writeProbes(stdin, probeMethods(event.record.data))
					}
					if err != nil && report.Lifecycle.Error == "" {
						report.Lifecycle.Error = "write lifecycle probe"
						cancel()
					}
				}
				if pendingProbes[info.responseID] {
					delete(pendingProbes, info.responseID)
					if info.errorReply && report.Lifecycle.Error == "" {
						report.Lifecycle.Error = "server rejected lifecycle probe"
						cancel()
					}
					if len(pendingProbes) == 0 && !report.Lifecycle.ProbeComplete {
						report.Lifecycle.ProbeComplete = true
						phase = "cleanup"
						if err := stdin.Close(); err != nil && report.Lifecycle.Error == "" {
							report.Lifecycle.Error = "close server stdin"
						}
						stdinClosed = true
						if !streamDone && cleanupTimer == nil {
							cleanupTimer = time.NewTimer(cfg.CleanupGrace)
							cleanupDone = cleanupTimer.C
						}
					}
				}
			}
			if event.err != nil && report.Lifecycle.Error == "" {
				report.Lifecycle.Error = event.err.Error()
				cancel()
			}
			if event.eof {
				streamDone = true
				stream = nil
				if cleanupTimer != nil {
					cleanupTimer.Stop()
					cleanupDone = nil
				}
			}
		case err := <-waited:
			processDone = true
			waited = nil
			report.Lifecycle.ProcessExited = true
			if err != nil && report.Lifecycle.Error == "" {
				report.Lifecycle.Error = "server process failed"
			}
		case <-cleanupDone:
			cleanupDone = nil
			_ = terminateProcessTree(cmd)
			if report.Lifecycle.Error == "" && !streamDone {
				report.Lifecycle.Error = "cleanup grace exceeded"
			}
			_ = stdout.Close()
		case <-ctxDone:
			ctxDone = nil
			if report.Lifecycle.Error == "" {
				report.Lifecycle.Error = "server lifecycle timeout"
			}
			_ = terminateProcessTree(cmd)
			_ = stdout.Close()
			if !stdinClosed {
				_ = stdin.Close()
				stdinClosed = true
			}
		}
	}
	if !stdinClosed {
		_ = stdin.Close()
	}

	if len(report.Diagnostics) > 0 {
		report.Status = "violations"
	} else if report.Lifecycle.Error != "" || !report.Lifecycle.Initialized || !report.Lifecycle.ProbeComplete || !report.Lifecycle.ProcessExited {
		report.Status = "error"
		if report.Lifecycle.Error == "" {
			report.Lifecycle.Error = "incomplete MCP lifecycle"
		}
	} else {
		report.Status = "passed"
	}
	return report
}

func ExitCode(report Report) int {
	switch report.Status {
	case "passed":
		return 0
	case "violations":
		return 1
	default:
		return 2
	}
}

func baseReport(cfg Config) Report {
	command := "<redacted>"
	if len(cfg.Command) > 0 {
		command = filepath.Base(cfg.Command[0])
	}
	version := cfg.ToolVersion
	if version == "" {
		version = "dev"
	}
	return Report{
		SchemaVersion: SchemaVersion,
		ToolVersion:   version,
		Command:       command,
		Status:        "error",
		Limits: Limits{
			TimeoutMillis:      cfg.Timeout.Milliseconds(),
			CleanupGraceMillis: cfg.CleanupGrace.Milliseconds(),
			MaxLineBytes:       cfg.MaxLineBytes,
			MaxStdoutBytes:     cfg.MaxStdoutBytes,
			MaxDiagnostics:     cfg.MaxDiagnostics,
		},
		Diagnostics: []Diagnostic{},
	}
}

func validateConfig(cfg Config) error {
	if len(cfg.Command) == 0 || strings.TrimSpace(cfg.Command[0]) == "" {
		return errors.New("missing command after --")
	}
	if cfg.Timeout <= 0 || cfg.CleanupGrace <= 0 || cfg.MaxLineBytes <= 0 || cfg.MaxStdoutBytes <= 0 || cfg.MaxLineBytes > cfg.MaxStdoutBytes || cfg.MaxDiagnostics <= 0 {
		return errors.New("invalid timeout or output limits")
	}
	return nil
}

func operationalError(report Report, message string) Report {
	report.Status = "error"
	report.Lifecycle.Error = message
	return report
}

func writeMessage(w io.Writer, message string) error {
	_, err := io.WriteString(w, message+"\n")
	return err
}

func readStream(r io.Reader, maxLine, maxTotal int, out chan<- streamEvent) {
	defer close(out)
	reader := bufio.NewReaderSize(r, 64*1024)
	line := 0
	var offset int64
	total := 0
	capacity := 64 * 1024
	if maxLine < capacity {
		capacity = maxLine
	}
	pending := make([]byte, 0, capacity)
	for {
		fragment, err := reader.ReadSlice('\n')
		if len(fragment) > 0 {
			if len(pending)+len(fragment) > maxLine {
				out <- streamEvent{err: errors.New("stdout line limit exceeded"), eof: true}
				return
			}
			if total+len(fragment) > maxTotal {
				out <- streamEvent{err: errors.New("stdout total limit exceeded"), eof: true}
				return
			}
			pending = append(pending, fragment...)
			total += len(fragment)
		}
		if err == nil {
			content := append([]byte(nil), pending[:len(pending)-1]...)
			line++
			out <- streamEvent{record: &record{data: content, line: line, offset: offset}}
			offset += int64(len(pending))
			pending = pending[:0]
			continue
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if err == io.EOF {
			if len(pending) > 0 {
				content := append([]byte(nil), pending...)
				line++
				out <- streamEvent{record: &record{data: content, line: line, offset: offset, unterminated: true}}
				offset += int64(len(pending))
			}
			out <- streamEvent{eof: true}
			return
		}
		out <- streamEvent{err: errors.New("read server stdout"), eof: true}
		return
	}
}

func validateRecord(rec record) messageInfo {
	if rec.unterminated {
		return messageInfo{reason: "unterminated-line"}
	}
	if !utf8.Valid(rec.data) {
		return messageInfo{reason: "invalid-utf8"}
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(rec.data, &obj); err != nil || obj == nil {
		return messageInfo{reason: "invalid-json"}
	}
	var version string
	if err := json.Unmarshal(obj["jsonrpc"], &version); err != nil || version != "2.0" {
		return messageInfo{reason: "not-jsonrpc-envelope"}
	}
	methodRaw, hasMethod := obj["method"]
	idRaw, hasID := obj["id"]
	_, hasResult := obj["result"]
	_, hasError := obj["error"]
	if hasMethod {
		var method string
		if json.Unmarshal(methodRaw, &method) != nil || method == "" || hasResult || hasError {
			return messageInfo{reason: "not-jsonrpc-envelope"}
		}
		if hasID && !validID(idRaw) {
			return messageInfo{reason: "not-jsonrpc-envelope"}
		}
		return messageInfo{}
	}
	if !hasID || !validID(idRaw) || hasResult == hasError {
		return messageInfo{reason: "not-jsonrpc-envelope"}
	}
	return messageInfo{responseID: string(bytes.TrimSpace(idRaw)), errorReply: hasError}
}

func probeMethods(data []byte) []string {
	var response struct {
		Result struct {
			Capabilities map[string]json.RawMessage `json:"capabilities"`
		} `json:"result"`
	}
	if json.Unmarshal(data, &response) != nil {
		return []string{"ping"}
	}
	methods := make([]string, 0, 3)
	for _, capability := range []string{"tools", "resources", "prompts"} {
		if _, ok := response.Result.Capabilities[capability]; ok {
			methods = append(methods, capability+"/list")
		}
	}
	if len(methods) == 0 {
		return []string{"ping"}
	}
	return methods
}

func writeProbes(w io.Writer, methods []string) (map[string]bool, error) {
	pending := make(map[string]bool, len(methods))
	for index, method := range methods {
		id := index + 2
		message, err := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"method":  method,
			"params":  map[string]any{},
		})
		if err != nil {
			return nil, err
		}
		if err := writeMessage(w, string(message)); err != nil {
			return nil, err
		}
		pending[fmt.Sprint(id)] = true
	}
	return pending, nil
}

func validID(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return false
	}
	var text string
	if json.Unmarshal(trimmed, &text) == nil {
		return true
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var number json.Number
	return decoder.Decode(&number) == nil
}

func newDiagnostic(reason, phase string, rec record) Diagnostic {
	return Diagnostic{
		RuleID:      RuleID,
		Reason:      reason,
		Phase:       phase,
		Line:        rec.line,
		ByteOffset:  rec.offset,
		ByteCount:   len(rec.data),
		Message:     "stdout record is not a valid JSON-RPC 2.0 message",
		Remediation: "route logs and descendant output to stderr, then rerun the preflight",
	}
}

func FormatText(report Report) string {
	var b strings.Builder
	for _, diagnostic := range report.Diagnostics {
		fmt.Fprintf(&b, "%s %s phase=%s line=%d offset=%d bytes=%d\n", diagnostic.RuleID, diagnostic.Reason, diagnostic.Phase, diagnostic.Line, diagnostic.ByteOffset, diagnostic.ByteCount)
	}
	switch report.Status {
	case "passed":
		b.WriteString("PASS stdout purity violations=0\n")
	case "violations":
		fmt.Fprintf(&b, "FAIL stdout purity violations=%d\n", len(report.Diagnostics))
		if report.DiagnosticsTruncated {
			b.WriteString("WARNING additional stdout purity violations omitted by max-diagnostics\n")
		}
	default:
		fmt.Fprintf(&b, "ERROR %s\n", report.Lifecycle.Error)
	}
	return b.String()
}

func DefaultConfig(command []string) Config {
	return Config{
		Command:        command,
		Timeout:        10 * time.Second,
		CleanupGrace:   250 * time.Millisecond,
		MaxLineBytes:   1024 * 1024,
		MaxStdoutBytes: 16 * 1024 * 1024,
		MaxDiagnostics: defaultMaxDiagnostics,
		Stderr:         os.Stderr,
	}
}
