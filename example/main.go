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

	// Use sqlx to create a table and insert data
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
