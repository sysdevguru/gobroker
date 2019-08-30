package log

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alpacahq/gopaca/env"
	"github.com/fluent/fluent-logger-golang/fluent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	once      sync.Once
	appLogger AppLogger
)

type AppLogger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	Panic(msg string, keysAndValues ...interface{})
	Fatal(msg string, keysAndValues ...interface{})
	SetDeploymentLevel(depl string)
	AddCallback(key string, level zapcore.Level, handler func(msg interface{})) error
	RemoveCallback(key string) error
	ListCallbacks() (keys []string)
}

func NewLogger() (AppLogger, error) {
	var zl *zap.Logger
	atom := zap.NewAtomicLevel()
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.StacktraceKey = "stack"
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	debug, _ := strconv.ParseBool(env.GetVar("DEBUG"))
	if debug {
		atom.SetLevel(zap.DebugLevel)
	} else {
		atom.SetLevel(zap.InfoLevel)
	}
	// use console encoder only for now - we may want to shift
	// to full JSON later on...
	zl = zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atom,
	),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.AddCaller(),
		zap.AddCallerSkip(2),
	)

	var fl *fluent.Fluent
	fluentHost := env.GetVar("FLUENTD_HOST")
	fluentPort := env.GetVar("FLUENTD_PORT")

	if fluentHost != "" && fluentPort != "" {
		port, err := strconv.Atoi(fluentPort)
		if err != nil {
			return nil, fmt.Errorf("failed to init logger : invalid fluentd port config %v", err)
		}

		fl, err = fluent.New(fluent.Config{
			FluentHost: fluentHost,
			FluentPort: port,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to init logger : failed to init fluent logger %v", err)
		}
	}

	return &logger{
		zap:       zl.Sugar(),
		fluent:    fl,
		callbacks: sync.Map{},
	}, nil
}

type logCallback struct {
	level   zapcore.Level
	handler func(text interface{})
}

type logger struct {
	zap       *zap.SugaredLogger
	fluent    *fluent.Fluent
	callbacks sync.Map
	depl      string
}

func (l *logger) runCallbacks(level zapcore.Level, msg string, keysAndValues ...interface{}) {
	message := l.logToMessage(level, msg, keysAndValues)

	l.callbacks.Range(func(key, value interface{}) bool {
		lc := value.(logCallback)
		if lc.level <= level {
			lc.handler(message)
		}
		return true
	})
}

func (l *logger) logToMessage(level zapcore.Level, msg string, keysAndValues interface{}) map[string]interface{} {
	_, file, no, ok := runtime.Caller(4)

	message := map[string]interface{}{
		"level":      level.String(),
		"message":    msg,
		"deployment": l.depl,
		"node":       env.GetVar("KUBERNETES_NODE_NAME"),
		"host":       env.GetVar("KUBERNETES_POD_NAME"),
		"service":    os.Args[0],
		"caller":     getFilename(file, no, ok),
	}

	if level >= zapcore.ErrorLevel {
		stack := zap.Stack("stack").String
		if len(stack) > 1000 {
			stack = stack[:1000]
		}
		message["stack"] = zap.Stack("stack").String
	}

	pairs := keysAndValues.([]interface{})

	if len(pairs)%2 != 0 {
		return message
	}

	for i := 0; i < len(pairs)-1; i += 2 {
		key, value := pairs[i].(string), pairs[i+1]

		switch v := value.(type) {
		case time.Time:
			message[key] = v.Format(time.RFC3339)
		case *time.Time:
			message[key] = v.Format(time.RFC3339)
		default:
			message[key] = fmt.Sprintf("%v", value)
		}
	}

	return message
}

func (l *logger) logfluent(level zapcore.Level, msg string, keysAndValues ...interface{}) {
	if l.fluent == nil {
		return
	}

	// DD logs only recognizes info, warning & error
	switch level {
	case zap.PanicLevel:
		fallthrough
	case zap.FatalLevel:
		level = zap.ErrorLevel
	}

	message := l.logToMessage(level, msg, keysAndValues)

	err := l.fluent.PostWithTime("alpaca.applog", time.Now(), message)
	if err != nil {
		l.zap.Errorw("failed to send log to fluent", "message", message, "error", err)
	}
}

// zap compatible filename
func getFilename(file string, no int, ok bool) string {
	if !ok {
		return "unknown"
	}

	// Get filename and line from
	// https://github.com/uber-go/zap/blob/e15639dab1b6ca5a651fe7ebfd8d682683b7d6a8/zapcore/entry.go

	idx := strings.LastIndexByte(file, '/')
	if idx == -1 {
		return file
	}

	// Find the penultimate separator.
	idx = strings.LastIndexByte(file[:idx], '/')
	if idx == -1 {
		return file
	}

	return fmt.Sprintf("%v:%v", file[idx+1:], no)
}

func (l *logger) Debug(msg string, keysAndValues ...interface{}) {
	// doesn't send log to fluentd as it is debug log.
	l.runCallbacks(zapcore.DebugLevel, msg, keysAndValues...)
	l.zap.Debugw(msg, keysAndValues...)
	l.zap.Sync()
}

func (l *logger) Info(msg string, keysAndValues ...interface{}) {
	l.logfluent(zapcore.InfoLevel, msg, keysAndValues...)
	l.runCallbacks(zapcore.InfoLevel, msg, keysAndValues...)
	l.zap.Infow(msg, keysAndValues...)
	l.zap.Sync()
}

func (l *logger) Warn(msg string, keysAndValues ...interface{}) {
	l.logfluent(zapcore.WarnLevel, msg, keysAndValues...)
	l.runCallbacks(zapcore.WarnLevel, msg, keysAndValues...)
	l.zap.Warnw(msg, keysAndValues...)
	l.zap.Sync()
}

func (l *logger) Error(msg string, keysAndValues ...interface{}) {
	l.logfluent(zapcore.ErrorLevel, msg, keysAndValues...)
	l.runCallbacks(zapcore.ErrorLevel, msg, keysAndValues...)
	l.zap.Errorw(msg, keysAndValues...)
	l.zap.Sync()
}

func (l *logger) Panic(msg string, keysAndValues ...interface{}) {
	l.logfluent(zapcore.PanicLevel, msg, keysAndValues...)
	l.runCallbacks(zapcore.PanicLevel, msg, keysAndValues...)
	l.zap.Panicw(msg, keysAndValues...)
}

func (l *logger) Fatal(msg string, keysAndValues ...interface{}) {
	l.logfluent(zapcore.FatalLevel, msg, keysAndValues...)
	l.runCallbacks(zapcore.FatalLevel, msg, keysAndValues...)
	l.zap.Fatalw(msg, keysAndValues...)
}

func (l *logger) SetDeploymentLevel(depl string) {
	l.depl = depl
}

// AddCallback registers a callback function with the
// logger to be executed based on the supplied level.
func (l *logger) AddCallback(key string, level zapcore.Level, handler func(msg interface{})) error {
	lc := logCallback{level: level, handler: handler}
	if _, loaded := l.callbacks.LoadOrStore(key, lc); loaded {
		return fmt.Errorf("callback already added with key: %s", key)
	}
	return nil
}

// RemoveCallback removes a callback from the logger by key.
func (l *logger) RemoveCallback(key string) error {
	if _, ok := l.callbacks.Load(key); !ok {
		return fmt.Errorf("no callback added with key: %s", key)
	}
	l.callbacks.Delete(key)
	return nil
}

// ListCallbacks returns a unordered list of the callback
// keys added to the logger.
func (l *logger) ListCallbacks() (keys []string) {
	l.callbacks.Range(func(key, value interface{}) bool {
		keys = append(keys, key.(string))
		return true
	})
	return keys
}

// Logger returns the singleton logger to be used for the duration
// of the application's runtime
func Logger() AppLogger {
	once.Do(func() {
		var err error
		appLogger, err = NewLogger()
		if err != nil {
			panic(err)
		}
	})
	return appLogger
}

// Debug logs a debug message followed by a set of key
// value pairs. Only logs when environment variable
// DEBUG=true.
func Debug(msg string, keysAndValues ...interface{}) {
	Logger().Debug(msg, keysAndValues...)
}

// Info logs an info message followed by a set of key
// value pairs.
func Info(msg string, keysAndValues ...interface{}) {
	Logger().Info(msg, keysAndValues...)
}

// Warn logs an warning message followed by a set of key
// value pairs.
func Warn(msg string, keysAndValues ...interface{}) {
	Logger().Warn(msg, keysAndValues...)
}

// Error logs an error message followed by a set of key
// value pairs, including a stack trace denoted by the
// key "stack".
func Error(msg string, keysAndValues ...interface{}) {
	Logger().Error(msg, keysAndValues...)
}

// Panic logs an error message followed by a set of key
// value pairs, including a stack trace denoted by the
// key "stack", then panics.
func Panic(msg string, keysAndValues ...interface{}) {
	Logger().Panic(msg, keysAndValues...)
}

// Fatal logs a fatal error message followed by a set of key
// value pairs, including a stack trace denoted by the
// key "stack", then calls os.Exit(1).
func Fatal(msg string, keysAndValues ...interface{}) {
	Logger().Fatal(msg, keysAndValues...)
}
