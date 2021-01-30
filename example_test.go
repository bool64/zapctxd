package zapctxd_test

import (
	"context"

	"github.com/bool64/ctxd"
	"github.com/bool64/zapctxd"
	"go.uber.org/zap"
)

func ExampleNew() {
	logger := zapctxd.New(zapctxd.Config{
		Level:     zap.WarnLevel,
		DevMode:   true,
		StripTime: true,
	})

	ctx := ctxd.AddFields(context.Background(),
		"foo", "bar",
	)

	logger.Info(ctx, "not logged due to WARN level config")

	logger.Error(ctx, "something failed",
		"baz", 1,
		"quux", 2.2,
	)

	logger.Important(ctx, "logged because is important")
	logger.Info(ctxd.WithDebug(ctx), "logged because of forced DEBUG mode")

	logger.AtomicLevel.SetLevel(zap.DebugLevel)
	logger.Info(ctx, "logged because logger level was changed to DEBUG")

	// Output:
	// <stripped>	ERROR	zapctxd/example_test.go:23	something failed	{"baz": 1, "quux": 2.2, "foo": "bar"}
	// <stripped>	INFO	zapctxd/example_test.go:28	logged because is important	{"foo": "bar"}
	// <stripped>	INFO	zapctxd/example_test.go:29	logged because of forced DEBUG mode	{"foo": "bar"}
	// <stripped>	INFO	zapctxd/example_test.go:32	logged because logger level was changed to DEBUG	{"foo": "bar"}
}
