package log

import (
	"fmt"
	"runtime"
)

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
