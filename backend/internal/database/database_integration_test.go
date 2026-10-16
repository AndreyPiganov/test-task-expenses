package database

import (
	"context"
	"os"
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	db, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("Open() returned an error: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate() returned an error: %v", err)
	}

	var expensesTable string
	if err := db.QueryRowContext(ctx, `SELECT 'expenses'::regclass::text`).Scan(&expensesTable); err != nil {
		t.Fatalf("expenses table was not created: %v", err)
	}
	if expensesTable != "expenses" {
		t.Errorf("table name = %q, want expenses", expensesTable)
	}
}
