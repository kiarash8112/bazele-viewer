// Package logging writes timestamped log lines tagged with a component name.
package logging

import (
	"log"
	"os"
	"time"

	"example.com/bazelviewer/example/pkg/clock"
)

// Logger logs for one component.
type Logger struct {
	component string
	clock     clock.Clock
	out       *log.Logger
}

// New returns a Logger that writes to stderr.
func New(component string) *Logger {
	return &Logger{component: component, clock: clock.System, out: log.New(os.Stderr, "", 0)}
}

// Printf logs a formatted line.
func (l *Logger) Printf(format string, args ...any) {
	prefix := []any{l.clock.Now().Format(time.RFC3339), l.component}
	l.out.Printf("%s %s: "+format, append(prefix, args...)...)
}
