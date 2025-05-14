# Go Wrapper For PostgreSQL Embedded

Install and run a PostgreSQL database locally on Linux or  MacOS or Windows. PostgreSQL can be
bundled with your application, or downloaded on demand.

See [theseus-rs/postgresql-embedded](https://github.com/theseus-rs/postgresql-embedded/blob/main/README.md) for more information.

### Supported Platforms

- Linux x86_64
- Darwin arm64

If you can get `go generate` to build the rust lib for other platforms, please send a PR.

### Install

```
go get github.com/chirino/go-pgembed
```

### Example

```go
package main

import (
	"fmt"
	"log"

	"github.com/chirino/go-pgembed"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	pg, err := pgembed.New(pgembed.Config{
		Version:    "16.0.0",
		DataDir:    ".postgresql",
		RuntimeDir: ".postgresql",
	})
	if err != nil {
		log.Fatalf("failed to start embedded PostgreSQL: %v", err)
	}

	defer func() {
		if err := pg.Stop(); err != nil {
			log.Fatalf("failed to stop embedded PostgreSQL: %v", err)
		}
	}()

	err = pg.CreateDatabase("lanzadm", "lanzadm")
	if err != nil {
		log.Fatalf("failed to create db instance: %v", err)
	}
	dsn, err := pg.ConnectionString("lanzadm")

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("sqlx.Connect(%s) failed: %v", dsn, err)
	}
	defer db.Close()

	result := &struct{ Message string }{}
	if err := db.Get(result, `SELECT 'hello world' AS message`); err != nil {
		log.Fatalf("failed to query: %v", err)
	} else {
		fmt.Println("Message:", result.Message)
	}
}
```

