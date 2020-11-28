package zapctxd_test

import (
	"bytes"
	"context"
	"sync"
	"testing"

	"github.com/bool64/ctxd"
	"github.com/bool64/zapctxd"
	"github.com/stretchr/testify/assert"
	"github.com/swaggest/assertjson"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLogger(t *testing.T) {
	w := bytes.NewBuffer(nil)
	c := zapctxd.New(zapctxd.Config{Level: zap.InfoLevel})

	ctx := context.Background()
	ctx = ctxd.AddFields(ctx, "bar", 2)
	ctx = ctxd.WithLogWriter(ctx, w)

	type ctxInt int
	// Put some pressure on context.
	for i := 0; i < 20; i++ {
		ctx = context.WithValue(ctx, ctxInt(i), i)
	}

	c.Debug(ctx, "hello!", "foo", 1)
	assert.Equal(t, ``, w.String())

	w.Reset()
	c.Info(ctx, "hello!", "foo", 1)
	assertjson.Equal(t,
		[]byte(`{"level":"info","time":"<ignore-diff>","msg":"hello!","bar":2,"foo":1}`), w.Bytes())

	w.Reset()
	c.Error(ctx, "hello!", "foo", 1)
	assertjson.Equal(t,
		[]byte(`{"level":"error","time":"<ignore-diff>","msg":"hello!","bar":2,"foo":1}`), w.Bytes())

	ctx = ctxd.WithDebug(ctx)

	w.Reset()
	c.Debug(ctx, "hello!", "foo", 1)
	assertjson.Equal(t,
		[]byte(`{"level":"debug","time":"<ignore-diff>","msg":"hello!","bar":2,"foo":1}`), w.Bytes())

	ctx = ctxd.WithDebug(ctx)
	*w = bytes.Buffer{}

	c.Debug(ctx, "hello!",
		"foo", 1,
		"def", ctxd.DeferredJSON(func() interface{} { return 123 }),
		"defstr", ctxd.DeferredJSON(func() interface{} { return "abc" }),
	)
	assertjson.Equal(t,
		[]byte(
			`{"level":"debug","time":"<ignore-diff>","msg":"hello!","bar":2,"foo":1,"def": 123,"defstr": "abc"}`),
		w.Bytes(),
	)
}

func TestLogger_concurrency(t *testing.T) {
	logger := zapctxd.New(zapctxd.Config{
		StripTime: true,
	})
	buf := &bytes.Buffer{}
	ctx := ctxd.WithLogWriter(context.Background(), buf)
	wg := sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			logger.Error(ctx, "hello")
		}()
	}
	wg.Wait()

	assert.Equal(t,
		string(bytes.Repeat([]byte(`{"level":"error","time":"<stripped>","msg":"hello"}`+"\n"), 10)),
		buf.String())
}

func TestLogger_Importantw(t *testing.T) {
	w := bytes.Buffer{}

	c := zapctxd.New(zapctxd.Config{
		Level:  zap.ErrorLevel, // Error level does not allow Info messages, but allows Important.
		Output: &w,
	})

	ctx := context.Background()

	c.Info(ctx, "hello!", "foo", 1)
	c.Important(ctx, "database wiped", "foo", 1)

	assertjson.Equal(t,
		[]byte(`{"level":"info","time":"<ignore-diff>","msg":"database wiped","foo":1}`), w.Bytes())
}

func TestLogger_Importantw_dev(t *testing.T) {
	w := bytes.Buffer{}

	c := zapctxd.New(zapctxd.Config{
		Level:     zap.ErrorLevel, // Error level does not allow Info messages, but allows Important.
		DevMode:   true,
		StripTime: true,
		Output:    &w,
	})

	ctx := context.Background()

	c.Info(ctx, "hello!", "foo", 1)
	c.Important(ctx, "account created", "foo", 1)

	assert.Equal(t, "<stripped>\tINFO\tzapctxd/logger_test.go:119\taccount created\t{\"foo\": 1}\n", w.String())
}

func TestNew_atomic_dev(t *testing.T) {
	w := bytes.Buffer{}

	c := zapctxd.New(zapctxd.Config{
		Level:     zap.ErrorLevel, // Error level does not allow Info messages, but allows Important.
		DevMode:   true,
		StripTime: true,
		Output:    &w,
	})

	ctx := context.Background()

	for _, lvl := range []zapcore.Level{zap.ErrorLevel, zap.WarnLevel, zap.InfoLevel, zap.DebugLevel} {
		c.AtomicLevel.SetLevel(lvl)

		c.Debug(ctx, "msg", "lvl", lvl)
		c.Info(ctx, "msg", "lvl", lvl)
		c.Warn(ctx, "msg", "lvl", lvl)
		c.Error(ctx, "msg", "lvl", lvl)
		c.Important(ctx, "msg", "lvl", lvl, "important", true)
	}

	assert.Equal(t, `<stripped>	ERROR	zapctxd/logger_test.go:142	msg	{"lvl": "error"}
<stripped>	INFO	zapctxd/logger_test.go:143	msg	{"lvl": "error", "important": true}
<stripped>	WARN	zapctxd/logger_test.go:141	msg	{"lvl": "warn"}
<stripped>	ERROR	zapctxd/logger_test.go:142	msg	{"lvl": "warn"}
<stripped>	INFO	zapctxd/logger_test.go:143	msg	{"lvl": "warn", "important": true}
<stripped>	INFO	zapctxd/logger_test.go:140	msg	{"lvl": "info"}
<stripped>	WARN	zapctxd/logger_test.go:141	msg	{"lvl": "info"}
<stripped>	ERROR	zapctxd/logger_test.go:142	msg	{"lvl": "info"}
<stripped>	INFO	zapctxd/logger_test.go:143	msg	{"lvl": "info", "important": true}
<stripped>	DEBUG	zapctxd/logger_test.go:139	msg	{"lvl": "debug"}
<stripped>	INFO	zapctxd/logger_test.go:140	msg	{"lvl": "debug"}
<stripped>	WARN	zapctxd/logger_test.go:141	msg	{"lvl": "debug"}
<stripped>	ERROR	zapctxd/logger_test.go:142	msg	{"lvl": "debug"}
<stripped>	INFO	zapctxd/logger_test.go:143	msg	{"lvl": "debug", "important": true}
`, w.String(), w.String())
}

func TestNew_atomic(t *testing.T) {
	w := bytes.NewBuffer(nil)

	c := zapctxd.New(zapctxd.Config{
		Level:     zap.ErrorLevel, // Error level does not allow Info messages, but allows Important.
		StripTime: true,
		Output:    w,
	})

	ctx := context.Background()

	for _, lvl := range []zapcore.Level{zap.ErrorLevel, zap.WarnLevel, zap.InfoLevel, zap.DebugLevel} {
		c.AtomicLevel.SetLevel(lvl)

		c.Debug(ctx, "msg", "lvl", lvl)
		c.Info(ctx, "msg", "lvl", lvl)
		c.Warn(ctx, "msg", "lvl", lvl)
		c.Error(ctx, "msg", "lvl", lvl)
		c.Important(ctx, "msg", "lvl", lvl, "important", true)
	}

	assert.Equal(t, `{"level":"error","time":"<stripped>","msg":"msg","lvl":"error"}
{"level":"info","time":"<stripped>","msg":"msg","lvl":"error","important":true}
{"level":"warn","time":"<stripped>","msg":"msg","lvl":"warn"}
{"level":"error","time":"<stripped>","msg":"msg","lvl":"warn"}
{"level":"info","time":"<stripped>","msg":"msg","lvl":"warn","important":true}
{"level":"info","time":"<stripped>","msg":"msg","lvl":"info"}
{"level":"warn","time":"<stripped>","msg":"msg","lvl":"info"}
{"level":"error","time":"<stripped>","msg":"msg","lvl":"info"}
{"level":"info","time":"<stripped>","msg":"msg","lvl":"info","important":true}
{"level":"debug","time":"<stripped>","msg":"msg","lvl":"debug"}
{"level":"info","time":"<stripped>","msg":"msg","lvl":"debug"}
{"level":"warn","time":"<stripped>","msg":"msg","lvl":"debug"}
{"level":"error","time":"<stripped>","msg":"msg","lvl":"debug"}
{"level":"info","time":"<stripped>","msg":"msg","lvl":"debug","important":true}
`, w.String(), w.String())
}
