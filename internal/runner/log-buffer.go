package runner

import (
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

const LogCapacity = 2000
const maxLineBytes = 16 * 1024

type Log struct {
	Sequence uint64
	Time     time.Time
	Service  string
	Stream   string
	Text     string
}

type logBuffer struct {
	entries []Log
	next    int
}

func (b *logBuffer) append(entry Log) {
	if len(b.entries) < LogCapacity {
		b.entries = append(b.entries, entry)
		return
	}
	b.entries[b.next] = entry
	b.next = (b.next + 1) % LogCapacity
}

func (b *logBuffer) snapshot() []Log {
	result := make([]Log, 0, len(b.entries))
	result = append(result, b.entries[b.next:]...)
	return append(result, b.entries[:b.next]...)
}

// Service output cannot send terminal control sequences into the dashboard.
func cleanLog(text string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, ansi.Strip(text))
}

type lineWriter struct {
	mu      sync.Mutex
	pending []byte
	emit    func(string)
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, b := range p {
		if b == '\n' || b == '\r' {
			w.flush()
			continue
		}
		w.pending = append(w.pending, b)
		if len(w.pending) >= maxLineBytes {
			w.flush()
		}
	}
	return len(p), nil
}

func (w *lineWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.flush()
}

func (w *lineWriter) flush() {
	if len(w.pending) == 0 {
		return
	}
	w.emit(cleanLog(string(w.pending)))
	w.pending = w.pending[:0]
}
