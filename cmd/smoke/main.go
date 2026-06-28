// Command smoke is a temporary end-to-end check of the purego DuckDB binding.
package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/ysksm/cc-otel/internal/duckdblib"
	_ "github.com/ysksm/cc-otel/internal/pduckdb" // registers the "duckdb" database/sql driver
)

func main() {
	if _, err := duckdblib.Ensure(); err != nil {
		fmt.Println("locate libduckdb:", err)
		os.Exit(1)
	}
	db, err := sql.Open("duckdb", ":memory:")
	must(err)
	defer db.Close()
	must(db.Ping())
	_, err = db.Exec(`CREATE TABLE spans(span_id VARCHAR, model VARCHAR, input_tokens BIGINT, attributes JSON)`)
	must(err)
	_, err = db.Exec(`INSERT INTO spans VALUES (?, ?, ?, ?)`, "s1", "claude-opus", 42, `{"gen_ai.system":"anthropic"}`)
	must(err)
	var id, model, attrs string
	var toks int64
	must(db.QueryRow(`SELECT span_id, model, input_tokens, attributes::VARCHAR FROM spans WHERE span_id=?`, "s1").
		Scan(&id, &model, &toks, &attrs))
	fmt.Printf("OK row: id=%s model=%s tokens=%d attrs=%s\n", id, model, toks, attrs)
	var ver string
	must(db.QueryRow(`SELECT version()`).Scan(&ver))
	fmt.Println("duckdb version:", ver)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
