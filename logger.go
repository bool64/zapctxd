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
	// Deprecated: Use SetLevelEnabler instead.
	AtomicLevel zap.AtomicLevel

	callerSkip   bool
	encoder      zapcore.Encoder
	levelEnabler zapcore.LevelEnabler
	sugared      *zap.SugaredLogger
	debug        *zap.SugaredLogger
	options      []zap.Option
	out          zapcore.WriteSyncer
}

// Config is log configuration.
type Config struct {
	Level      zapcore.Level   `split_words:"true" default:"error"`
	DevMode    bool            `split_words:"true"`
	FieldNames ctxd.FieldNames `split_words:"true"`
	Output     io.Writer
	ZapOptions []zap.Option

	// ColoredOutput enables colored output in development mode.
	ColoredOutput bool
	// StripTime disables time variance in logger.
	StripTime bool
}

// New creates contextualized logger with zap backend.
func New(cfg Config, options ...zap.Option) *Logger {
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
		levelEnabler: zap.NewAtomicLevelAt(level),
		out:          out,
		options:      append(cfg.ZapOptions, options...),
	}

	if cfg.DevMode {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeTime = timeEncoder

		if cfg.ColoredOutput {
			encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}

		l.encoder = zapcore.NewConsoleEncoder(encoderConfig)
		l.callerSkip = true
		l.options = append(l.options, zap.Development(), zap.AddCaller(), zap.AddCallerSkip(1))
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

	return &l
}

// WrapZapLoggers creates contextualized logger with provided zap loggers.
func WrapZapLoggers(sugared, debug *zap.Logger, encoder zapcore.Encoder, options ...zap.Option) *Logger {
	sugared = sugared.WithOptions(options...)
	debug = debug.WithOptions(options...)

	return &Logger{
		levelEnabler: sugared.Core(),
		sugared:      sugared.Sugar(),
		debug:        debug.Sugar(),
		encoder:      encoder,
		options:      options,
	}
}

func (l *Logger) make() {
	l.sugared = zap.New(zapcore.NewCore(
		l.encoder,
		l.out,
		loggerLevelEnabler(l),
	), l.options...).Sugar()

	l.debug = zap.New(zapcore.NewCore(
		l.encoder,
		l.out,
		zap.DebugLevel,
	), l.options...).Sugar()
}

// SetLevelEnabler sets level enabler.
func (l *Logger) SetLevelEnabler(enabler zapcore.LevelEnabler) {
	if _, ok := l.levelEnabler.(zapcore.Core); ok {
		panic("cannot set level enabler when logger is created with zap loggers")
	}

	l.levelEnabler = enabler
}

// SkipCaller adapts logger for wrapping by increasing skip caller counter.
func (l *Logger) SkipCaller() *Logger {
	if !l.callerSkip {
		return l
	}

	nl := *l

	nl.debug = nl.debug.Desugar().WithOptions(zap.AddCallerSkip(1)).Sugar()
	nl.sugared = nl.sugared.Desugar().WithOptions(zap.AddCallerSkip(1)).Sugar()

	return &nl
}

// Debug implements ctxd.Logger.
func (l *Logger) Debug(ctx context.Context, msg string, keysAndValues ...any) {
	z := l.get(ctx, zap.DebugLevel)
	if z == nil {
		return
	}

	var (
		fv = ctxd.Fields(ctx)
		kv = keysAndValues
	)

	if len(fv) > 0 {
		kv = make([]any, 0, len(fv)+len(kv))

		kv = append(kv, keysAndValues...)
		kv = append(kv, fv...)
	}

	for i := 1; i < len(kv); i += 2 {
		v := kv[i]
		if err, ok := v.(error); ok {
			kv[i] = err.Error()

			var se ctxd.StructuredError

			if errors.As(err, &se) {
				kv = expandError(kv, se, i)
			}
		}
	}

	z.Debugw(msg, kv...)
}

func expandError(kv []any, se ctxd.StructuredError, i int) []any {
	kv[i] = se.Error()

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

	return kv
}

// Info implements ctxd.Logger.
func (l *Logger) Info(ctx context.Context, msg string, keysAndValues ...any) {
	z := l.get(ctx, zap.InfoLevel)
	if z == nil {
		return
	}

	var (
		fv = ctxd.Fields(ctx)
		kv = keysAndValues
	)

	if len(fv) > 0 {
		kv = make([]any, 0, len(fv)+len(kv))

		kv = append(kv, keysAndValues...)
		kv = append(kv, fv...)
	}

	for i := 1; i < len(kv); i += 2 {
		v := kv[i]
		if err, ok := v.(error); ok {
			kv[i] = err.Error()

			var se ctxd.StructuredError

			if errors.As(err, &se) {
				kv = expandError(kv, se, i)
			}
		}
	}

	z.Infow(msg, kv...)
}

// Important implements ctxd.Logger.
func (l *Logger) Important(ctx context.Context, msg string, keysAndValues ...any) {
	z := l.get(ctxd.WithDebug(ctx), zap.InfoLevel)
	if z == nil {
		return
	}

	var (
		fv = ctxd.Fields(ctx)
		kv = keysAndValues
	)

	if len(fv) > 0 {
		kv = make([]any, 0, len(fv)+len(kv))

		kv = append(kv, keysAndValues...)
		kv = append(kv, fv...)
	}

	for i := 1; i < len(kv); i += 2 {
		v := kv[i]
		if err, ok := v.(error); ok {
			kv[i] = err.Error()

			var se ctxd.StructuredError

			if errors.As(err, &se) {
				kv = expandError(kv, se, i)
			}
		}
	}

	z.Infow(msg, kv...)
}

// Warn implements ctxd.Logger.
func (l *Logger) Warn(ctx context.Context, msg string, keysAndValues ...any) {
	z := l.get(ctx, zap.WarnLevel)
	if z == nil {
		return
	}

	var (
		fv = ctxd.Fields(ctx)
		kv = keysAndValues
	)

	if len(fv) > 0 {
		kv = make([]any, 0, len(fv)+len(kv))

		kv = append(kv, keysAndValues...)
		kv = append(kv, fv...)
	}

	for i := 1; i < len(kv); i += 2 {
		v := kv[i]
		if err, ok := v.(error); ok {
			kv[i] = err.Error()

			var se ctxd.StructuredError

			if errors.As(err, &se) {
				kv = expandError(kv, se, i)
			}
		}
	}

	z.Warnw(msg, kv...)
}

// Error implements ctxd.Logger.
func (l *Logger) Error(ctx context.Context, msg string, keysAndValues ...any) {
	z := l.get(ctx, zap.ErrorLevel)
	if z == nil {
		return
	}

	var (
		fv = ctxd.Fields(ctx)
		kv = keysAndValues
	)

	if len(fv) > 0 {
		kv = make([]any, 0, len(fv)+len(kv))

		kv = append(kv, keysAndValues...)
		kv = append(kv, fv...)
	}

	for i := 1; i < len(kv); i += 2 {
		v := kv[i]
		if err, ok := v.(error); ok {
			kv[i] = err.Error()

			var se ctxd.StructuredError

			if errors.As(err, &se) {
				kv = expandError(kv, se, i)
			}
		}
	}

	z.Errorw(msg, kv...)
}

func (l *Logger) get(ctx context.Context, level zapcore.Level) *zap.SugaredLogger {
	z := l.sugared
	if !l.levelEnabler.Enabled(level) {
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
		level := zapcore.LevelEnabler(zap.DebugLevel)
		if !isDebug {
			level = l.levelEnabler
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

var _ ctxd.LoggerProvider = &Logger{}

// CtxdLogger provides contextualized logger.
func (l *Logger) CtxdLogger() ctxd.Logger { //nolint: ireturn
	return l
}

// ZapLogger returns *zap.Logger that used in Logger.
func (l *Logger) ZapLogger() *zap.Logger {
	return l.sugared.Desugar()
}

func loggerLevelEnabler(l *Logger) zap.LevelEnablerFunc {
	return func(lvl zapcore.Level) bool {
		return l.levelEnabler.Enabled(lvl)
	}
}
