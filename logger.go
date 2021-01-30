// Package zapctxd implements contextualized logger with go.uber.org/zap.
package zapctxd

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/bool64/ctxd"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ ctxd.Logger = Logger{}

// Logger is a contextualized zap logger.
type Logger struct {
	AtomicLevel zap.AtomicLevel
	callerSkip  int
	encoder     zapcore.Encoder
	sugared     *zap.SugaredLogger
	debug       *zap.SugaredLogger
	options     []zap.Option
	out         zapcore.WriteSyncer
}

// Config is log configuration.
type Config struct {
	Level   zapcore.Level `envconfig:"LOG_LEVEL" default:"error"`
	DevMode bool          `envconfig:"LOG_DEV_MODE"`
	Output  io.Writer

	// StripTime disables time variance in logger.
	StripTime bool
}

// New creates contextualized logger with zap backend.
func New(cfg Config) Logger {
	level := zap.InfoLevel

	if cfg.Level != 0 {
		level = cfg.Level
	}

	var out zapcore.WriteSyncer = os.Stdout

	if cfg.Output != nil {
		out = zapcore.AddSync(cfg.Output)
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	timeEncoder := zapcore.ISO8601TimeEncoder

	if cfg.StripTime {
		timeEncoder = func(_ time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString("<stripped>")
		}
	}

	l := Logger{
		AtomicLevel: zap.NewAtomicLevelAt(level),
		out:         out,
	}

	if cfg.DevMode {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeTime = timeEncoder
		l.encoder = zapcore.NewConsoleEncoder(encoderConfig)
		l.callerSkip = 1
		l.options = append(l.options, zap.Development(), zap.AddCaller(), zap.AddCallerSkip(l.callerSkip))
	} else {
		encoderConfig.MessageKey = "msg"
		encoderConfig.TimeKey = "time"
		encoderConfig.EncodeTime = timeEncoder
		l.encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	l.make()

	return l
}

func (l *Logger) make() {
	l.sugared = zap.New(zapcore.NewCore(
		l.encoder,
		l.out,
		l.AtomicLevel,
	), l.options...).Sugar()

	l.debug = zap.New(zapcore.NewCore(
		l.encoder,
		l.out,
		zap.DebugLevel,
	), l.options...).Sugar()
}

// SkipCaller adapts logger for wrapping by increasing skip caller counter.
func (l Logger) SkipCaller() Logger {
	if l.callerSkip == 0 {
		return l
	}

	l.callerSkip++
	l.make()

	return l
}

// Debug implements ctxd.Logger.
func (l Logger) Debug(ctx context.Context, msg string, keysAndValues ...interface{}) {
	z := l.get(ctx, zap.DebugLevel)
	if z == nil {
		return
	}

	var (
		fv = ctxd.Fields(ctx)
		kv = keysAndValues
	)

	if len(fv) > 0 {
		kv = make([]interface{}, 0, len(fv)+len(kv))

		kv = append(kv, keysAndValues...)
		kv = append(kv, fv...)
	}

	z.Debugw(msg, kv...)
}

// Info implements ctxd.Logger.
func (l Logger) Info(ctx context.Context, msg string, keysAndValues ...interface{}) {
	z := l.get(ctx, zap.InfoLevel)
	if z == nil {
		return
	}

	var (
		fv = ctxd.Fields(ctx)
		kv = keysAndValues
	)

	if len(fv) > 0 {
		kv = make([]interface{}, 0, len(fv)+len(kv))

		kv = append(kv, keysAndValues...)
		kv = append(kv, fv...)
	}

	z.Infow(msg, kv...)
}

// Important implements ctxd.Logger.
func (l Logger) Important(ctx context.Context, msg string, keysAndValues ...interface{}) {
	z := l.get(ctxd.WithDebug(ctx), zap.InfoLevel)
	if z == nil {
		return
	}

	var (
		fv = ctxd.Fields(ctx)
		kv = keysAndValues
	)

	if len(fv) > 0 {
		kv = make([]interface{}, 0, len(fv)+len(kv))

		kv = append(kv, keysAndValues...)
		kv = append(kv, fv...)
	}

	z.Infow(msg, kv...)
}

// Warn implements ctxd.Logger.
func (l Logger) Warn(ctx context.Context, msg string, keysAndValues ...interface{}) {
	z := l.get(ctx, zap.WarnLevel)
	if z == nil {
		return
	}

	var (
		fv = ctxd.Fields(ctx)
		kv = keysAndValues
	)

	if len(fv) > 0 {
		kv = make([]interface{}, 0, len(fv)+len(kv))

		kv = append(kv, keysAndValues...)
		kv = append(kv, fv...)
	}

	z.Warnw(msg, kv...)
}

// Error implements ctxd.Logger.
func (l Logger) Error(ctx context.Context, msg string, keysAndValues ...interface{}) {
	z := l.get(ctx, zap.ErrorLevel)
	if z == nil {
		return
	}

	var (
		fv = ctxd.Fields(ctx)
		kv = keysAndValues
	)

	if len(fv) > 0 {
		kv = make([]interface{}, 0, len(fv)+len(kv))

		kv = append(kv, keysAndValues...)
		kv = append(kv, fv...)
	}

	z.Errorw(msg, kv...)
}

func (l Logger) get(ctx context.Context, level zapcore.Level) *zap.SugaredLogger {
	z := l.sugared
	if !l.AtomicLevel.Enabled(level) {
		z = nil
	}

	isDebug := ctxd.IsDebug(ctx)
	if isDebug {
		z = l.debug
	}

	if z == nil {
		return nil
	}

	writer := ctxd.LogWriter(ctx)
	if writer != nil {
		level := zap.DebugLevel
		if !isDebug {
			level = l.AtomicLevel.Level()
		}

		ws, ok := writer.(zapcore.WriteSyncer)
		if !ok {
			ws = zapcore.AddSync(writer)
		}

		return zap.New(zapcore.NewCore(
			l.encoder,
			ws,
			level,
		)).Sugar()
	}

	return z
}

// CtxdLogger provides contextualized logger.
func (l Logger) CtxdLogger() ctxd.Logger {
	return l
}

// ZapLogger returns *zap.Logger that used in Logger.
func (l Logger) ZapLogger() *zap.Logger {
	return l.sugared.Desugar()
}
