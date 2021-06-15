// Package zapctxd implements contextualized logger with go.uber.org/zap.
package zapctxd

import (
	"context"
	"errors"
	"io"
	"os"
	"time"

	"github.com/bool64/ctxd"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ ctxd.Logger = &Logger{}

// Logger is a contextualized zap logger.
type Logger struct {
	AtomicLevel zap.AtomicLevel
	callerSkip  bool
	encoder     zapcore.Encoder
	sugared     *zap.SugaredLogger
	debug       *zap.SugaredLogger
	options     []zap.Option
	out         zapcore.WriteSyncer
}

// Config is log configuration.
type Config struct {
	Level      zapcore.Level   `split_words:"true" default:"error"`
	DevMode    bool            `split_words:"true"`
	FieldNames ctxd.FieldNames `split_words:"true"`
	Output     io.Writer
	ZapOptions []zap.Option

	// StripTime disables time variance in logger.
	StripTime bool
}

// New creates contextualized logger with zap backend.
func New(cfg Config, options ...zap.Option) Logger {
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
		options:     append(cfg.ZapOptions, options...),
	}

	if cfg.DevMode {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeTime = timeEncoder
		l.encoder = zapcore.NewConsoleEncoder(encoderConfig)
		l.callerSkip = true
		l.options = append(l.options, zap.Development(), zap.AddCaller(), zap.AddCallerSkip(2))
	} else {
		encoderConfig.MessageKey = "msg"
		encoderConfig.TimeKey = "time"

		if cfg.FieldNames.Message != "" {
			encoderConfig.MessageKey = cfg.FieldNames.Message
		}

		if cfg.FieldNames.Timestamp != "" {
			encoderConfig.TimeKey = cfg.FieldNames.Timestamp
		}

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
func (l *Logger) SkipCaller() *Logger {
	if !l.callerSkip {
		return l
	}

	l.options = append(l.options, zap.AddCallerSkip(1))
	l.make()

	return l
}

func (l *Logger) log(ctx context.Context, f func(msg string, keysAndValues ...interface{}), msg string, keysAndValues ...interface{}) {
	var (
		fv = ctxd.Fields(ctx)
		kv = keysAndValues
	)

	if len(fv) > 0 {
		kv = make([]interface{}, 0, len(fv)+len(kv))

		kv = append(kv, keysAndValues...)
		kv = append(kv, fv...)
	}

	for i := 1; i < len(kv); i += 2 {
		v := kv[i]
		if err, ok := v.(error); ok {
			var se ctxd.StructuredError

			if errors.As(err, &se) {
				kv[i] = err.Error()

				tuples := se.Tuples()

				for k := 0; k < len(tuples)-1; k += 2 {
					exists := false

					for j := 0; j < len(kv)-1; j += 2 {
						if kv[j] == tuples[k] {
							exists = true

							break
						}
					}

					if !exists {
						kv = append(kv, tuples[k], tuples[k+1])
					}
				}
			}
		}
	}

	f(msg, kv...)
}

// Debug implements ctxd.Logger.
func (l *Logger) Debug(ctx context.Context, msg string, keysAndValues ...interface{}) {
	z := l.get(ctx, zap.DebugLevel)
	if z == nil {
		return
	}

	l.log(ctx, z.Debugw, msg, keysAndValues...)
}

// Info implements ctxd.Logger.
func (l *Logger) Info(ctx context.Context, msg string, keysAndValues ...interface{}) {
	z := l.get(ctx, zap.InfoLevel)
	if z == nil {
		return
	}

	l.log(ctx, z.Infow, msg, keysAndValues...)
}

// Important implements ctxd.Logger.
func (l *Logger) Important(ctx context.Context, msg string, keysAndValues ...interface{}) {
	z := l.get(ctxd.WithDebug(ctx), zap.InfoLevel)
	if z == nil {
		return
	}

	l.log(ctx, z.Infow, msg, keysAndValues...)
}

// Warn implements ctxd.Logger.
func (l *Logger) Warn(ctx context.Context, msg string, keysAndValues ...interface{}) {
	z := l.get(ctx, zap.WarnLevel)
	if z == nil {
		return
	}

	l.log(ctx, z.Warnw, msg, keysAndValues...)
}

// Error implements ctxd.Logger.
func (l *Logger) Error(ctx context.Context, msg string, keysAndValues ...interface{}) {
	z := l.get(ctx, zap.ErrorLevel)
	if z == nil {
		return
	}

	l.log(ctx, z.Errorw, msg, keysAndValues...)
}

func (l *Logger) get(ctx context.Context, level zapcore.Level) *zap.SugaredLogger {
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
func (l *Logger) CtxdLogger() ctxd.Logger {
	return l
}

// ZapLogger returns *zap.Logger that used in Logger.
func (l *Logger) ZapLogger() *zap.Logger {
	return l.sugared.Desugar()
}
