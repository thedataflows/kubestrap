package log

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

var (
	Logger     = NewDefaultLogger()
	ParseLevel = zerolog.ParseLevel

	DebugLevel = zerolog.DebugLevel
	InfoLevel  = zerolog.InfoLevel
	WarnLevel  = zerolog.WarnLevel
	ErrorLevel = zerolog.ErrorLevel
	FatalLevel = zerolog.FatalLevel
	PanicLevel = zerolog.PanicLevel
	NoLevel    = zerolog.NoLevel
	Disabled   = zerolog.Disabled
	TraceLevel = zerolog.TraceLevel

	AllLevelsValues = []string{
		zerolog.LevelTraceValue,
		zerolog.LevelDebugValue,
		zerolog.LevelInfoValue,
		zerolog.LevelWarnValue,
		zerolog.LevelErrorValue,
		zerolog.LevelFatalValue,
		zerolog.LevelPanicValue,
		"disabled",
	}

	LogFormats = []string{"console", "json"}
)

func NewDefaultLogger() zerolog.Logger {
	output := zerolog.ConsoleWriter{
		Out:        PreferredWriter(),
		TimeFormat: time.RFC3339}
	return zerolog.New(output).
		With().
		Timestamp().
		Logger()
}

func IsValidLogFormat(format string) error {
	for _, f := range LogFormats {
		if format == f {
			return nil
		}
	}
	return fmt.Errorf("invalid log format '%s'. Provide one of: %v", format, LogFormats)
}

func SetLogFormat(format string) error {
	err := IsValidLogFormat(format)
	if err != nil {
		return err
	}
	switch format {
	// This is the current default
	// case "console":
	case "json":
		Logger = Logger.Output(PreferredWriter())
	}
	return nil
}

func PreferredWriter() io.Writer {
	return io.MultiWriter(
		new(bytes.Buffer),
		os.Stdout,
	)
}

func SetLogLevel(format string) error {
	parsedLevel, err := ParseLevel(format)
	if err != nil {
		return err
	}
	Logger = Logger.Level(parsedLevel)
	return nil
}

func Trace(i ...interface{}) {
	Logger.Trace().Msg(fmt.Sprint(i...))
}

func Tracef(format string, i ...interface{}) {
	Logger.Trace().Msgf(format, i...)
}

func Debug(i ...interface{}) {
	Logger.Debug().Msg(fmt.Sprint(i...))
}

func Debugf(format string, i ...interface{}) {
	Logger.Debug().Msgf(format, i...)
}

func Info(i ...interface{}) {
	Logger.Info().Msg(fmt.Sprint(i...))
}

func Infof(format string, i ...interface{}) {
	Logger.Info().Msgf(format, i...)
}

func Warn(i ...interface{}) {
	Logger.Warn().Msg(fmt.Sprint(i...))
}

func Warnf(format string, i ...interface{}) {
	Logger.Warn().Msgf(format, i...)
}

func Error(i ...interface{}) {
	Logger.Error().Msg(fmt.Sprint(i...))
}

func Errorf(format string, i ...interface{}) {
	Logger.Error().Msgf(format, i...)
}

func Fatal(i ...interface{}) {
	Logger.Fatal().Msg(fmt.Sprint(i...))
}

func Fatalf(format string, i ...interface{}) {
	Logger.Fatal().Msgf(format, i...)
}

func Panic(i ...interface{}) {
	Logger.Panic().Msg(fmt.Sprint(i...))
}

func Panicf(format string, i ...interface{}) {
	Logger.Panic().Msgf(format, i...)
}

func Print(i ...interface{}) {
	Logger.WithLevel(zerolog.NoLevel).Str("level", "-").Msg(fmt.Sprint(i...))
}

func Printf(format string, i ...interface{}) {
	Logger.WithLevel(zerolog.NoLevel).Str("level", "-").Msgf(format, i...)
}
