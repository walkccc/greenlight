package jsonlog

import (
	"encoding/json"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

// Level represents the severity level for a log entry.
type Level int8

// Constants which represent a specific severity level. We use the iota keyword as a shortcut to
// assign successive integer values to the constants.
const (
	LevelInfo  Level = iota // Has the value 0.
	LevelError              // Has the value 1.
	LevelFatal              // Has the value 2.
	LevelOff                // Has the value 3.
)

// String returns a human-friendly string for the severity level.
func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return ""
	}
}

// Logger is a custom logger.
type Logger struct {
	out      io.Writer  // the output destination that the log entries will be written to
	minLevel Level      // the minimum severity level that the log entries will be written for
	mtx      sync.Mutex // a mutex for coordinating the writes
}

// New returns a new Logger instance which writes log entries at or above a minimum severity level
// to a specific output destination.
func New(out io.Writer, minLevel Level) *Logger {
	return &Logger{
		out:      out,
		minLevel: minLevel,
	}
}

// PrintInfo is a helper that writes INFO level log entries.
func (l *Logger) PrintInfo(message string, properties map[string]string) {
	l.print(LevelInfo, message, properties)
}

// PrintError is a helper that writes ERROR level log entries.
func (l *Logger) PrintError(err error, properties map[string]string) {
	l.print(LevelError, err.Error(), properties)
}

// PrintFatal is a helper that writes FATAL level log entries.
func (l *Logger) PrintFatal(err error, properties map[string]string) {
	l.print(LevelFatal, err.Error(), properties)
	os.Exit(1) // Terminate the application for entries at the FATAL level.
}

// print is an internal method for writing the log entry.
func (l *Logger) print(level Level, message string, properties map[string]string) (int, error) {
	// If the severity level of the log entry is below the minimum severity for the logger, then
	// return with no further action.
	if level < l.minLevel {
		return 0, nil
	}

	// aux holds the data for the log entry.
	aux := struct {
		Level      string            `json:"level"`
		Time       string            `json:"time"`
		Message    string            `json:"message"`
		Properties map[string]string `json:"properties,omitempty"`
		Trace      string            `json:"trace,omitempty"`
	}{
		Level:      level.String(),
		Time:       time.Now().UTC().Format(time.RFC3339),
		Message:    message,
		Properties: properties,
	}

	// Include a stck trace for entries at the ERROR and FATAL levels.
	if level >= LevelError {
		aux.Trace = string(debug.Stack())
	}

	// line holds the actual log entry text.
	var line []byte

	// Marshal aux struct to JSON and store it in the line variable. If there was a problem creating
	// the JSON, set the contents of the log entry to be that plain-text error message instead.
	line, err := json.Marshal(aux)
	if err != nil {
		line = []byte(LevelError.String() + ": unable to marshal log message: " + err.Error())
	}

	// Lock the mutex so that no two writes to the output destination can happen concurrently.
	l.mtx.Lock()
	defer l.mtx.Unlock()

	return l.out.Write(append(line, '\n'))
}
