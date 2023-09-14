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
	Log        = NewDefaultLogger()
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

type CustomLogger struct {
	zerolog.Logger
}

func (l *CustomLogger) Tracef(format string, args ...interface{}) {
	l.Trace().Msgf(format, args...)
}
func (l *CustomLogger) Debugf(format string, args ...interface{}) {
	l.Debug().Msgf(format, args...)
}

func (l *CustomLogger) Errorf(format string, args ...interface{}) {
	l.Error().Msgf(format, args...)
}

func (l *CustomLogger) Warnf(format string, args ...interface{}) {
	l.Warn().Msgf(format, args...)
}

func (l *CustomLogger) Infof(format string, args ...interface{}) {
	l.Info().Msgf(format, args...)
}

func (l *CustomLogger) SetLogger(logger zerolog.Logger) {
	l.Logger = logger
}

func (l *CustomLogger) GetLogger() *zerolog.Logger {
	return &l.Logger
}

func GetLevel() zerolog.Level {
	return Log.GetLogger().GetLevel()
}

func NewDefaultLogger() CustomLogger {
	output := zerolog.ConsoleWriter{
		Out:        PreferredWriter(),
		TimeFormat: time.RFC3339,
	}
	return CustomLogger{
		zerolog.New(output).
			With().
			Timestamp().
			Logger(),
	}
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
		Log.SetLogger(Log.GetLogger().Output(PreferredWriter()))
	}
	return nil
}

func PreferredWriter() io.Writer {
	return io.MultiWriter(
		new(bytes.Buffer),
		os.Stderr,
	)
}

func SetLogLevel(format string) error {
	parsedLevel, err := ParseLevel(format)
	if err != nil {
		return err
	}
	Log.SetLogger(Log.GetLogger().Level(parsedLevel))
	return nil
}

func Trace(i ...interface{}) {
	Log.GetLogger().Trace().Msg(fmt.Sprint(i...))
}

func Tracef(format string, i ...interface{}) {
	Log.GetLogger().Trace().Msgf(format, i...)
}

func Debug(i ...interface{}) {
	Log.GetLogger().Debug().Msg(fmt.Sprint(i...))
}

func Debugf(format string, i ...interface{}) {
	Log.GetLogger().Debug().Msgf(format, i...)
}

func Info(i ...interface{}) {
	Log.GetLogger().Info().Msg(fmt.Sprint(i...))
}

func Infof(format string, i ...interface{}) {
	Log.GetLogger().Info().Msgf(format, i...)
}

func Warn(i ...interface{}) {
	Log.GetLogger().Warn().Msg(fmt.Sprint(i...))
}

func Warnf(format string, i ...interface{}) {
	Log.GetLogger().Warn().Msgf(format, i...)
}

func Error(i ...interface{}) {
	Log.GetLogger().Error().Msg(fmt.Sprint(i...))
}

func Errorf(format string, i ...interface{}) {
	Log.GetLogger().Error().Msgf(format, i...)
}

func Fatal(i ...interface{}) {
	Log.GetLogger().Fatal().Msg(fmt.Sprint(i...))
}

func Fatalf(format string, i ...interface{}) {
	Log.GetLogger().Fatal().Msgf(format, i...)
}

func Panic(i ...interface{}) {
	Log.GetLogger().Panic().Msg(fmt.Sprint(i...))
}

func Panicf(format string, i ...interface{}) {
	Log.GetLogger().Panic().Msgf(format, i...)
}

func Print(i ...interface{}) {
	Log.GetLogger().WithLevel(zerolog.NoLevel).Str("level", "-").Msg(fmt.Sprint(i...))
}

func Printf(format string, i ...interface{}) {
	Log.GetLogger().WithLevel(zerolog.NoLevel).Str("level", "-").Msgf(format, i...)
}
