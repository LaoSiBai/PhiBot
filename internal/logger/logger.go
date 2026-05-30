package logger

import (
	"io"
	"os"

	"github.com/charmbracelet/log"
)

var defaultLogger *log.Logger

func Init(level log.Level) {
	defaultLogger = log.NewWithOptions(os.Stderr, log.Options{
		Level:           level,
		ReportTimestamp: true,
	})
}

func SetOutput(w io.Writer) {
	if defaultLogger != nil {
		defaultLogger.SetOutput(w)
	}
}

func Get() *log.Logger {
	if defaultLogger == nil {
		Init(log.InfoLevel)
	}
	return defaultLogger
}

func Debug(msg string, keyvals ...interface{}) { Get().Debug(msg, keyvals...) }
func Info(msg string, keyvals ...interface{})  { Get().Info(msg, keyvals...) }
func Warn(msg string, keyvals ...interface{})  { Get().Warn(msg, keyvals...) }
func Error(msg string, keyvals ...interface{}) { Get().Error(msg, keyvals...) }
func Fatal(msg string, keyvals ...interface{}) { Get().Fatal(msg, keyvals...) }
