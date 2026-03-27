package video

import (
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

func (o *TraceOverlay) appendLineLocked(line string) {
	if line == "" {
		return
	}

	o.lines = append(o.lines, line)
	if overflow := len(o.lines) - o.maxLines; overflow > 0 {
		o.lines = append([]string(nil), o.lines[overflow:]...)
	}
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
