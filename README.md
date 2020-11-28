# Contextualized Logging with Zap

This library implements [contextualized logger](https://pkg.go.dev/github.com/bool64/ctxd#Logger) with 
[`zap`](https://pkg.go.dev/go.uber.org/zap).

[![Build Status](https://github.com/bool64/zapctxd/workflows/test/badge.svg)](https://github.com/bool64/zapctxd/actions?query=branch%3Amaster+workflow%3Atest)
[![Coverage Status](https://codecov.io/gh/bool64/zapctxd/branch/master/graph/badge.svg)](https://codecov.io/gh/bool64/zapctxd)
[![GoDevDoc](https://img.shields.io/badge/dev-doc-00ADD8?logo=go)](https://pkg.go.dev/github.com/bool64/zapctxd)
[![time tracker](https://wakatime.com/badge/github/bool64/zapctxd.svg)](https://wakatime.com/badge/github/bool64/zapctxd)
![Code lines](https://sloc.xyz/github/bool64/zapctxd/?category=code)
![Comments](https://sloc.xyz/github/bool64/zapctxd/?category=comments)

## Example

```go
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
```
