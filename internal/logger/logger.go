package logger

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog"
)

var once sync.Once
var logger zerolog.Logger

type Level int

const (
	// DebugLevel defines debug log level.
	DebugLevel Level = iota
	// InfoLevel defines info log level.
	InfoLevel
	// WarnLevel defines warn log level.
	WarnLevel
	// ErrorLevel defines error log level.
	ErrorLevel
	// FatalLevel defines fatal log level.
	FatalLevel
	// PanicLevel defines panic log level.
	PanicLevel
	// NoLevel defines an absent log level.
	NoLevel
	// Disabled disables the logger.
	Disabled

	// TraceLevel defines trace log level.
	TraceLevel Level = -1
)

func Init(level Level, name string) {
	once.Do(func() {
		shareDir := filepath.Join(os.Getenv("HOME"), ".local/share/com.github.swampapp")
		logsDir := filepath.Join(shareDir, "logs")
		if err := os.MkdirAll(logsDir, 0755); err != nil {
			panic(err)
		}

		logFile, err := os.OpenFile(filepath.Join(logsDir, name+".log"),
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		logger = zerolog.New(logFile).With().Timestamp().Logger()
		logger = logger.Level(zerolog.Level(level))
	})
}

func Error(err error, msg string) {
	logger.Error().Err(err).Msg(msg)
}

func Errorf(err error, msg string, fmt ...interface{}) {
	logger.Error().Err(err).Msgf(msg, fmt...)
}

func Infof(msg string, fmt ...interface{}) {
	logger.Info().Msgf(msg, fmt...)
}

func Info(msg string) {
	logger.Info().Msg(msg)
}

func Debugf(msg string, fmt ...interface{}) {
	logger.Debug().Msgf(msg, fmt...)
}

func Debug(msg string) {
	logger.Debug().Msg(msg)
}

func Print(args ...interface{}) {
	logger.Print(args...)
}

func Printf(msg string, args ...interface{}) {
	logger.Printf(msg, args...)
}

func Fatalf(err error, msg string, fmt ...interface{}) {
	logger.Fatal().Err(err).Msgf(msg, fmt...)
}

func Fatal(err error, msg string) {
	logger.Fatal().Err(err).Msg(msg)
}

func Warnf(msg string, fmt ...interface{}) {
	logger.Warn().Msgf(msg, fmt...)
}

func Warn(msg string) {
	logger.Warn().Msgf(msg)
}
