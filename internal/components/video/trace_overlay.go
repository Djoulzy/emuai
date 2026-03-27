package video

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

const defaultTraceOverlayMaxLines = 256

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

type TraceOverlay struct {
	mu       sync.RWMutex
	maxLines int
	partial  string
	lines    []string
	status   TraceStatus
}

type TraceStatus struct {
	Cycle     uint64
	Registers string
	Flags     string
}

func NewTraceOverlay(maxLines int) *TraceOverlay {
	if maxLines <= 0 {
		maxLines = defaultTraceOverlayMaxLines
	}

	return &TraceOverlay{maxLines: maxLines}
}

func (o *TraceOverlay) Write(p []byte) (int, error) {
	if o == nil {
		return len(p), nil
	}

	sanitized := ansiEscapePattern.ReplaceAllString(string(p), "")
	sanitized = strings.ReplaceAll(sanitized, "\r\n", "\n")
	sanitized = strings.ReplaceAll(sanitized, "\r", "\n")

	o.mu.Lock()
	defer o.mu.Unlock()

	combined := o.partial + sanitized
	parts := strings.Split(combined, "\n")
	last := len(parts) - 1
	for idx, line := range parts {
		if idx == last {
			o.partial = line
			continue
		}
		o.appendLineLocked(line)
	}

	if strings.HasSuffix(combined, "\n") {
		o.partial = ""
	}

	return len(p), nil
}

func (o *TraceOverlay) Text(maxLines int) string {
	if o == nil {
		return ""
	}

	o.mu.RLock()
	defer o.mu.RUnlock()

	lines := o.visibleLinesLocked(maxLines)
	return strings.Join(lines, "\n")
}

func (o *TraceOverlay) SetClock(cycle uint64) {
	if o == nil {
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	o.status.Cycle = cycle
}

func (o *TraceOverlay) Status() TraceStatus {
	if o == nil {
		return TraceStatus{}
	}

	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.status
}

func (o *TraceOverlay) appendLineLocked(line string) {
	if line == "" {
		return
	}

	o.lines = append(o.lines, line)
	if overflow := len(o.lines) - o.maxLines; overflow > 0 {
		o.lines = append([]string(nil), o.lines[overflow:]...)
	}
}

func (o *TraceOverlay) SetCPUState(a byte, x byte, y byte, sp byte, p byte) {
	if o == nil {
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	o.status.Registers = fmt.Sprintf("A:%02X X:%02X Y:%02X SP:%02X P:%02X", a, x, y, sp, p)
	o.status.Flags = flagsFromProcessorStatus(p)
}

func flagsFromProcessorStatus(status byte) string {
	flags := []struct {
		mask  byte
		label byte
	}{
		{mask: 1 << 7, label: 'N'},
		{mask: 1 << 6, label: 'V'},
		{mask: 1 << 5, label: 'U'},
		{mask: 1 << 4, label: 'B'},
		{mask: 1 << 3, label: 'D'},
		{mask: 1 << 2, label: 'I'},
		{mask: 1 << 1, label: 'Z'},
		{mask: 1 << 0, label: 'C'},
	}

	decoded := make([]byte, 0, len(flags))
	for _, flag := range flags {
		if status&flag.mask != 0 {
			decoded = append(decoded, flag.label)
		}
	}
	if len(decoded) == 0 {
		return "-"
	}

	return string(decoded)
}

func defaultTraceStatus(traceOn bool, status TraceStatus) TraceStatus {
	if !traceOn {
		return TraceStatus{
			Registers: "A:-- X:-- Y:-- SP:-- P:--",
			Flags:     "OFF",
		}
	}
	if status.Registers == "" {
		status.Registers = "A:-- X:-- Y:-- SP:-- P:--"
	}
	if status.Flags == "" {
		status.Flags = "-"
	}
	return status
}

func (s TraceStatus) CycleString(traceOn bool) string {
	if !traceOn {
		return "--"
	}
	return fmt.Sprintf("%d", s.Cycle)
}

func (o *TraceOverlay) visibleLinesLocked(maxLines int) []string {
	lines := append([]string(nil), o.lines...)
	if o.partial != "" {
		lines = append(lines, o.partial)
	}

	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return lines
}
