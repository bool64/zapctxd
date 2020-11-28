package zapctxd_test

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/bool64/ctxd"
	"github.com/bool64/zapctxd"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

// BenchmarkSugaredZap benchmarks sugared zap logger.
// BenchmarkSugaredZap-4   	  975686	      1174 ns/op	     256 B/op	       1 allocs/op.
func BenchmarkSugaredZap(b *testing.B) {
	logger := zap.New(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionConfig().EncoderConfig),
			&zaptest.Discarder{},
			zap.DebugLevel,
		))
	s := logger.Sugar()

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		s.Debugw("hello!", "bla2", 2, "bla", 1)
	}
}

// BenchmarkCtxFull benchmarks zapctxd.Logger performance with rich context (realistic).
// BenchmarkCtxFull-4   	  320354	      3378 ns/op	    1248 B/op	       8 allocs/op.
func BenchmarkCtxFull(b *testing.B) {
	c := zapctxd.New(zapctxd.Config{})

	ctx := context.Background()
	ctx = ctxd.AddFields(ctx, "bla2", 2, "ops", 3.5)
	ctx = ctxd.WithLogWriter(ctx, ioutil.Discard)
	ctx = ctxd.WithDebug(ctx)

	// Put some pressure on context.
	type ctxInt int

	for i := 0; i < 20; i++ {
		ctx = context.WithValue(ctx, ctxInt(i), i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		c.Debug(ctx, "hello!", "bla", 1)
	}
}

// BenchmarkCtxLite benchmarks zapctxd.Logger performance with empty context (optimistic).
// BenchmarkCtxLite-4   	  830715	      1381 ns/op	     256 B/op	       1 allocs/op.
func BenchmarkCtxLite(b *testing.B) {
	c := zapctxd.New(zapctxd.Config{
		Level:  zap.DebugLevel,
		Output: ioutil.Discard,
	})

	ctx := context.Background()

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		c.Debug(ctx, "hello!", "bla2", 2, "bla", 1)
	}
}
