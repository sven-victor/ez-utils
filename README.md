# ez-utils

A Go toolkit for logging, errors, crypto, collections, object storage, databases, and tracing.

```bash
go get github.com/sven-victor/ez-utils
```

Requires Go 1.23+. Import only the subpackages you need.

API details live in source godoc: `go doc github.com/sven-victor/ez-utils/<pkg>`.

## Packages

| Package | Import path | Description |
| --- | --- | --- |
| buffer | `github.com/sven-victor/ez-utils/buffer` | Limit how much is written; prefetch a stream prefix |
| capacity | `github.com/sven-victor/ez-utils/capacity` | Human-readable byte sizes (`100MB`, `1.5GB`) |
| conv | `github.com/sven-victor/ez-utils/conv` | JSON conversion, flexible times, URL query decoding |
| crypto | `github.com/sven-victor/ez-utils/crypto` | AES-CBC encrypt and decrypt |
| errors | `github.com/sven-victor/ez-utils/errors` | Errors with HTTP status and application codes |
| fs | `github.com/sven-victor/ez-utils/fs` | Temporary filesystem that cleans up on close |
| generator | `github.com/sven-victor/ez-utils/generator` | Time-sortable UUIDs and trace IDs |
| http | `github.com/sven-victor/ez-utils/http` | Path joining and client IP behind trusted proxies |
| jwt | `github.com/sven-victor/ez-utils/jwt` | Issue and verify JWTs |
| log | `github.com/sven-victor/ez-utils/log` | Leveled, contextual loggers on go-kit/log |
| log/flag | `github.com/sven-victor/ez-utils/log/flag` | Register log settings as command-line flags |
| safe | `github.com/sven-victor/ez-utils/safe` | Encryption, hashing, and secrets stored at rest |
| sets | `github.com/sven-victor/ez-utils/sets` | Generic sets, bounded ArraySet, and IP networks |
| signals | `github.com/sven-victor/ez-utils/signals` | Process-wide graceful shutdown |
| wrapper (`w`) | `github.com/sven-victor/ez-utils/wrapper` | Generic helpers, ring queue, templates, process groups |
| gorm | `github.com/sven-victor/ez-utils/clients/gorm` | MySQL / SQLite / ClickHouse clients |
| redis | `github.com/sven-victor/ez-utils/clients/redis` | Redis client and session helpers |
| tls | `github.com/sven-victor/ez-utils/clients/tls` | Build `crypto/tls.Config` from options |
| tracing | `github.com/sven-victor/ez-utils/clients/tracing` | OpenTelemetry (OTLP / Zipkin / file) |
| storage | `github.com/sven-victor/ez-utils/clients/storage` | Object storage interface and registry |
| storage/fs | `github.com/sven-victor/ez-utils/clients/storage/fs` | Local / zip / in-memory backends |
| storage/s3 | `github.com/sven-victor/ez-utils/clients/storage/s3` | S3-compatible backend |
| storage/oss | `github.com/sven-victor/ez-utils/clients/storage/oss` | Alibaba Cloud OSS backend |

The `wrapper` package name is `w`:

```go
import w "github.com/sven-victor/ez-utils/wrapper"
```

The `generator` package name is `g`.

## Examples

### Logging

```go
package main

import (
	"context"

	"github.com/go-kit/log/level"
	"github.com/sven-victor/ez-utils/log"
)

func main() {
	logger := log.New(log.WithConfig(log.MustNewConfig("info", "logfmt")))
	log.SetDefaultLogger(logger)

	level.Info(log.GetDefaultLogger()).Log("msg", "started")

	ctx, ctxLogger := log.NewContextLogger(context.Background())
	_ = ctx
	level.Info(ctxLogger).Log("msg", "with trace id")
}
```

Use `log/flag.AddFlags` to register `log.level`, `log.format`, `log.file`, and related flags with `pflag` or kingpin.

### Errors

```go
err := errors.NewError(http.StatusNotFound, "user not found", "USER_NOT_FOUND")
if errors.IsNotFount(err) {
	// HTTP 404 or record not found
}
```

`errors.With` and `errors.WithMessage` keep an existing status and application code. `Is`, `As`, `Unwrap`, and `Join` match the standard library.

### Sets and slices

```go
s := sets.New("a", "b", "a")
s.Has("b")          // true
s.SortedList()      // ["a", "b"]

ids := w.Filter([]int{1, 2, 3}, func(n int) bool { return n%2 == 1 })
names := w.Map(ids, strconv.Itoa)
```

### Capacity

```go
n, err := capacity.ParseCapacities("1.5GB")
_ = n.String() // human-readable form
```

`Capacities` implements JSON and YAML marshaling, plus `pflag.Value` (`Set` / `Type` / `String`).

### Encryption and secret fields

```go
cipher, err := safe.Encrypt([]byte("hello"), os.Getenv("GLOBAL_ENCRYPT_KEY"), nil)
plain, err := safe.Decrypt(cipher, os.Getenv("GLOBAL_ENCRYPT_KEY"))

secret := safe.NewEncryptedString("password", os.Getenv("GLOBAL_ENCRYPT_KEY"))
_ = secret.String()              // ciphertext
text, _ := secret.UnsafeString() // plaintext
```

`safe.String` stores ciphertext in JSON, YAML, and GORM. On process start, `GLOBAL_ENCRYPT_KEY` must be empty or 8 / 16 / 24 / 32 bytes.

### Graceful shutdown

```go
h := signals.SetupSignalHandler(logger)
h.AddRequest(1)
defer h.DoneRequest()

<-h.Channel() // SIGTERM / SIGINT (or the Windows equivalents)
```

`wrapper.Group` listens for shutdown signals in `Run` and then calls `w.ExitFunc` (default `os.Exit`) after a timeout.

### Object storage

Blank-import a backend to register it, then construct a client from config:

```go
import (
	"github.com/sven-victor/ez-utils/clients/storage"
	_ "github.com/sven-victor/ez-utils/clients/storage/fs"
	_ "github.com/sven-victor/ez-utils/clients/storage/s3"
	_ "github.com/sven-victor/ez-utils/clients/storage/oss"
)

client, err := storage.NewClient(ctx, storage.NewMapConfigProvider(map[string]any{
	"type": "s3",
	// bucket, endpoint, region, credentials, ...
}))
```

Registered types: `local` / `file` / `zip` / `in-memory` / `inmemory`, `s3`, and `oss`.

### Databases

```go
db, err := gorm.NewMySQLClient(ctx, "app", gorm.MySQLOptions{
	Host:     "127.0.0.1",
	Username: "root",
	Schema:   "app",
})
session := db.Session(ctx)
```

Also available: `NewSQLiteClient` / `NewGormSQLiteClient` and `NewClickhouseClient`. Pool stats are exported as Prometheus metrics.

### Tracing

```go
tp, err := tracing.NewTraceProvider(ctx, &tracing.TraceOptions{
	ServiceName: "my-service",
	HTTP:        &tracing.HTTPClientOptions{Endpoint: "localhost:4318"},
})
defer tp.Shutdown(ctx)
```

Set exactly one of `HTTP`, `GRPC`, `Zipkin`, or `File` on `TraceOptions`. If none is set, a no-op exporter is used.

## Development

```bash
make test        # go test -race ./... (with -tags make_test)
make full-test   # full test run without the make_test tag
make common-lint # golangci-lint
```

## License

[Apache License 2.0](LICENSE)
