package logging

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/sirupsen/logrus"
)

type (
	LoggingLevel struct {
		logrus.Level
	}
)

const (
	LogLevelsStr = "panic, fatal, error, warn, warning, info, debug, trace"
)

var (
	Logger     = logrus.New()
	FatalLevel = logrus.FatalLevel
	ErrorLevel = logrus.ErrorLevel
	WarnLevel  = logrus.WarnLevel
	InfoLevel  = logrus.InfoLevel
	DebugLevel = logrus.DebugLevel
	TraceLevel = logrus.TraceLevel
)

func init() {
	Logger.SetFormatter(&logrus.TextFormatter{ForceColors: true})
	buf := new(bytes.Buffer)
	// TODO improve capturing of stdout/stderr to logger because now is not working. Perhaps overwrite os.Stdout ?
	w := io.MultiWriter(buf, os.Stdout)
	Logger.SetOutput(w)
}

func ParseLevel(lvl string) logrus.Level {
	level, err := logrus.ParseLevel(lvl)
	if err != nil {
		Logger.Errorf("invalid log level '%s'. Provide one of: %s", lvl, LogLevelsStr)
		Logger.Exit(1)
	}
	return level
}

func ExitOnError(err error, code int) {
	if err != nil {
		Logger.Errorf("%s", err)
		Logger.Exit(code)
	}
}

// TraceInfo returns function name and line of code
func TraceInfo() string {
	pc := make([]uintptr, 15)
	n := runtime.Callers(3, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	return fmt.Sprintf("%s:%d", frame.Function, frame.Line)
}

// ErrWithTrace returns error message with some additional trace info.
//
// See TraceInfo()
func ErrWithTrace(err error) error {
	if err != nil {
		return fmt.Errorf("[%s] %+v", TraceInfo(), err)
	}
	return nil
}
