//go:build integration

package dal_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mariadb-dal-api/internal/dal"
	"pgregory.net/rapid"
)

func setupDB(t *testing.T) (*sql.DB, string) {
	t.Helper()

	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Fatalf("ping db: %v", err)
	}

	table := fmt.Sprintf("test_dal_%d", time.Now().UnixNano())

	_, err = db.ExecContext(ctx, fmt.Sprintf(
		"CREATE TABLE `%s` (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255), status VARCHAR(50))",
		table,
	))
	if err != nil {
		db.Close()
		t.Fatalf("create table: %v", err)
	}

	t.Cleanup(func() {
		db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS `%s`", table))
		db.Close()
	})

	return db, table
}

func TestIntegration_CRUD(t *testing.T) {
	db, table := setupDB(t)
	d := dal.New(db)
	ctx := context.Background()

	t.Run("Insert and GetByID", func(t *testing.T) {
		record, err := d.Insert(ctx, table, map[string]any{
			"name":   "Alice",
			"status": "active",
		})
		if err != nil {
			t.Fatalf("Insert: %v", err)
		}

		id, ok := record["id"]
		if !ok {
			t.Fatal("Insert response missing id")
		}

		got, err := d.GetByID(ctx, table, fmt.Sprintf("%v", id))
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}

		if got["name"] != "Alice" {
			t.Errorf("name: got %q, want %q", got["name"], "Alice")
		}
		if got["status"] != "active" {
			t.Errorf("status: got %q, want %q", got["status"], "active")
		}
	})

	t.Run("List all", func(t *testing.T) {
		// Use a fresh table to avoid interference from other subtests.
		db2, tbl2 := setupDB(t)
		d2 := dal.New(db2)

		_, err := d2.Insert(ctx, tbl2, map[string]any{"name": "Bob", "status": "active"})
		if err != nil {
			t.Fatalf("Insert 1: %v", err)
		}
		_, err = d2.Insert(ctx, tbl2, map[string]any{"name": "Carol", "status": "inactive"})
		if err != nil {
			t.Fatalf("Insert 2: %v", err)
		}

		rows, err := d2.List(ctx, tbl2, nil, 100, 0)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(rows) != 2 {
			t.Errorf("List: got %d records, want 2", len(rows))
		}
	})

	t.Run("List with filter", func(t *testing.T) {
		db2, tbl2 := setupDB(t)
		d2 := dal.New(db2)

		_, _ = d2.Insert(ctx, tbl2, map[string]any{"name": "Dave", "status": "active"})
		_, _ = d2.Insert(ctx, tbl2, map[string]any{"name": "Eve", "status": "inactive"})
		_, _ = d2.Insert(ctx, tbl2, map[string]any{"name": "Frank", "status": "active"})

		rows, err := d2.List(ctx, tbl2, map[string]string{"status": "active"}, 100, 0)
		if err != nil {
			t.Fatalf("List with filter: %v", err)
		}
		if len(rows) != 2 {
			t.Errorf("List with filter: got %d records, want 2", len(rows))
		}
		for _, r := range rows {
			if r["status"] != "active" {
				t.Errorf("List with filter: unexpected status %q", r["status"])
			}
		}
	})

	t.Run("Update", func(t *testing.T) {
		db2, tbl2 := setupDB(t)
		d2 := dal.New(db2)

		rec, err := d2.Insert(ctx, tbl2, map[string]any{"name": "Grace", "status": "active"})
		if err != nil {
			t.Fatalf("Insert: %v", err)
		}
		id := fmt.Sprintf("%v", rec["id"])

		updated, err := d2.Update(ctx, tbl2, id, map[string]any{"name": "Grace Updated", "status": "inactive"})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if updated["name"] != "Grace Updated" {
			t.Errorf("Update name: got %q, want %q", updated["name"], "Grace Updated")
		}
		if updated["status"] != "inactive" {
			t.Errorf("Update status: got %q, want %q", updated["status"], "inactive")
		}

		got, err := d2.GetByID(ctx, tbl2, id)
		if err != nil {
			t.Fatalf("GetByID after Update: %v", err)
		}
		if got["name"] != "Grace Updated" {
			t.Errorf("GetByID after Update name: got %q, want %q", got["name"], "Grace Updated")
		}
	})

	t.Run("Patch", func(t *testing.T) {
		db2, tbl2 := setupDB(t)
		d2 := dal.New(db2)

		rec, err := d2.Insert(ctx, tbl2, map[string]any{"name": "Heidi", "status": "active"})
		if err != nil {
			t.Fatalf("Insert: %v", err)
		}
		id := fmt.Sprintf("%v", rec["id"])

		_, err = d2.Patch(ctx, tbl2, id, map[string]any{"status": "inactive"})
		if err != nil {
			t.Fatalf("Patch: %v", err)
		}

		got, err := d2.GetByID(ctx, tbl2, id)
		if err != nil {
			t.Fatalf("GetByID after Patch: %v", err)
		}
		if got["name"] != "Heidi" {
			t.Errorf("Patch should not change name: got %q, want %q", got["name"], "Heidi")
		}
		if got["status"] != "inactive" {
			t.Errorf("Patch status: got %q, want %q", got["status"], "inactive")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		db2, tbl2 := setupDB(t)
		d2 := dal.New(db2)

		rec, err := d2.Insert(ctx, tbl2, map[string]any{"name": "Ivan", "status": "active"})
		if err != nil {
			t.Fatalf("Insert: %v", err)
		}
		id := fmt.Sprintf("%v", rec["id"])

		if err := d2.Delete(ctx, tbl2, id); err != nil {
			t.Fatalf("Delete: %v", err)
		}

		_, err = d2.GetByID(ctx, tbl2, id)
		if err != dal.ErrNotFound {
			t.Errorf("GetByID after Delete: got %v, want ErrNotFound", err)
		}
	})

	t.Run("GetByID not found", func(t *testing.T) {
		_, err := d.GetByID(ctx, table, "999999")
		if err != dal.ErrNotFound {
			t.Errorf("GetByID not found: got %v, want ErrNotFound", err)
		}
	})

	t.Run("Delete not found", func(t *testing.T) {
		err := d.Delete(ctx, table, "999999")
		if err != dal.ErrNotFound {
			t.Errorf("Delete not found: got %v, want ErrNotFound", err)
		}
	})
}

// Feature: mariadb-dal-api, Property 7: Insert round-trip
// Validates: Requirements 3.1, 4.1
func TestPropertyInsertRoundTrip(t *testing.T) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set")
	}

	db, table := setupDB(t)
	d := dal.New(db)
	ctx := context.Background()

	// The table schema has: id (auto), name VARCHAR(255), status VARCHAR(50).
	// Generators produce values that fit within those column constraints.
	rapid.Check(t, func(rt *rapid.T) {
		name := rapid.StringMatching(`[A-Za-z][A-Za-z0-9 ]{0,50}`).Draw(rt, "name")
		status := rapid.StringMatching(`[A-Za-z][A-Za-z0-9]{0,48}`).Draw(rt, "status")

		data := map[string]any{
			"name":   name,
			"status": status,
		}

		inserted, err := d.Insert(ctx, table, data)
		if err != nil {
			rt.Fatalf("Insert failed: %v", err)
		}

		idVal, ok := inserted["id"]
		if !ok {
			rt.Fatal("Insert response missing id field")
		}
		id := fmt.Sprintf("%v", idVal)

		got, err := d.GetByID(ctx, table, id)
		if err != nil {
			rt.Fatalf("GetByID failed: %v", err)
		}

		if got["name"] != name {
			rt.Errorf("name mismatch: inserted %q, got %q", name, got["name"])
		}
		if got["status"] != status {
			rt.Errorf("status mismatch: inserted %q, got %q", status, got["status"])
		}
	})
}

// Feature: mariadb-dal-api, Property 8: List filter correctness
// Validates: Requirements 5.1, 5.2, 5.3
func TestPropertyListFilterCorrectness(t *testing.T) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set")
	}

	db, table := setupDB(t)
	d := dal.New(db)
	ctx := context.Background()

	// Pre-populate the table with a fixed set of records covering both status values.
	statuses := []string{"active", "inactive", "pending"}
	for i := 0; i < 30; i++ {
		status := statuses[i%len(statuses)]
		_, err := d.Insert(ctx, table, map[string]any{
			"name":   fmt.Sprintf("user%d", i),
			"status": status,
		})
		if err != nil {
			t.Fatalf("pre-populate Insert: %v", err)
		}
	}

	rapid.Check(t, func(rt *rapid.T) {
		// Pick a random status value to filter by (including one that won't match).
		filterStatus := rapid.SampledFrom([]string{"active", "inactive", "pending", "nonexistent"}).Draw(rt, "filterStatus")

		rows, err := d.List(ctx, table, map[string]string{"status": filterStatus}, 100, 0)
		if err != nil {
			rt.Fatalf("List with filter failed: %v", err)
		}

		// When filtering for a value that doesn't exist, the result must be an empty slice (not an error).
		if filterStatus == "nonexistent" {
			if len(rows) != 0 {
				rt.Errorf("expected empty slice for non-matching filter, got %d records", len(rows))
			}
			return
		}

		// Every returned record must satisfy the applied filter.
		for _, row := range rows {
			if row["status"] != filterStatus {
				rt.Errorf("filter correctness violated: expected status %q, got %q", filterStatus, row["status"])
			}
		}
	})
}

// Feature: mariadb-dal-api, Property 9: Limit invariant
// Validates: Requirements 5.4
func TestPropertyLimitInvariant(t *testing.T) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set")
	}

	db, table := setupDB(t)
	d := dal.New(db)
	ctx := context.Background()

	// Pre-populate the table with more than 100 records so the limit is always exercised.
	for i := 0; i < 110; i++ {
		_, err := d.Insert(ctx, table, map[string]any{
			"name":   fmt.Sprintf("user%d", i),
			"status": "active",
		})
		if err != nil {
			t.Fatalf("pre-populate Insert: %v", err)
		}
	}

	// Property: for any limit L in [1, 100], len(results) <= L.
	// rapid defaults to 100 checks; set RAPID_CHECKS=100 to be explicit.
	rapid.Check(t, func(rt *rapid.T) {
		limit := rapid.IntRange(1, 100).Draw(rt, "limit")

		rows, err := d.List(ctx, table, nil, limit, 0)
		if err != nil {
			rt.Fatalf("List failed: %v", err)
		}

		if len(rows) > limit {
			rt.Errorf("limit invariant violated: requested limit %d but got %d records", limit, len(rows))
		}
	})

	// Also verify the default limit (100) case: no limit specified means limit=100.
	rows, err := d.List(ctx, table, nil, 100, 0)
	if err != nil {
		t.Fatalf("List with default limit failed: %v", err)
	}
	if len(rows) > 100 {
		t.Errorf("default limit invariant violated: got %d records, want <= 100", len(rows))
	}
}

// Feature: mariadb-dal-api, Property 10: Offset pagination consistency
// Validates: Requirements 5.5
func TestPropertyOffsetPaginationConsistency(t *testing.T) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set")
	}

	db, table := setupDB(t)
	d := dal.New(db)
	ctx := context.Background()

	// Pre-populate the table with a fixed set of records.
	const N = 20
	const fixedLimit = 5
	for i := 0; i < N; i++ {
		_, err := d.Insert(ctx, table, map[string]any{
			"name":   fmt.Sprintf("user%02d", i),
			"status": "active",
		})
		if err != nil {
			t.Fatalf("pre-populate Insert: %v", err)
		}
	}

	// Fetch the full result set once (no offset) to use as the reference.
	fullList, err := d.List(ctx, table, nil, N, 0)
	if err != nil {
		t.Fatalf("full List failed: %v", err)
	}
	if len(fullList) != N {
		t.Fatalf("expected %d records in full list, got %d", N, len(fullList))
	}

	// Property: for any offset O in [0, N], List(offset=O, limit=fixedLimit)
	// returns records identical (by id) to fullList[O : O+fixedLimit].
	rapid.Check(t, func(rt *rapid.T) {
		offset := rapid.IntRange(0, N).Draw(rt, "offset")

		rows, err := d.List(ctx, table, nil, fixedLimit, offset)
		if err != nil {
			rt.Fatalf("List with offset=%d failed: %v", offset, err)
		}

		// Compute the expected slice from the full result set.
		end := offset + fixedLimit
		if end > len(fullList) {
			end = len(fullList)
		}
		expected := fullList[offset:end]

		if len(rows) != len(expected) {
			rt.Fatalf("offset=%d: expected %d records, got %d", offset, len(expected), len(rows))
		}

		for i, row := range rows {
			if row["id"] != expected[i]["id"] {
				rt.Errorf("offset=%d position %d: id mismatch: got %v, want %v",
					offset, i, row["id"], expected[i]["id"])
			}
		}
	})
}

// Feature: mariadb-dal-api, Property 11: PUT round-trip
// Validates: Requirements 6.1
func TestPropertyPUTRoundTrip(t *testing.T) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set")
	}

	db, table := setupDB(t)
	d := dal.New(db)
	ctx := context.Background()

	rapid.Check(t, func(rt *rapid.T) {
		// Insert an initial record to have an existing row.
		initialName := rapid.StringMatching(`[A-Za-z][A-Za-z0-9 ]{0,50}`).Draw(rt, "initialName")
		initialStatus := rapid.StringMatching(`[A-Za-z][A-Za-z0-9]{0,48}`).Draw(rt, "initialStatus")

		inserted, err := d.Insert(ctx, table, map[string]any{
			"name":   initialName,
			"status": initialStatus,
		})
		if err != nil {
			rt.Fatalf("Insert failed: %v", err)
		}

		id := fmt.Sprintf("%v", inserted["id"])

		// Generate replacement data for the PUT.
		newName := rapid.StringMatching(`[A-Za-z][A-Za-z0-9 ]{0,50}`).Draw(rt, "newName")
		newStatus := rapid.StringMatching(`[A-Za-z][A-Za-z0-9]{0,48}`).Draw(rt, "newStatus")

		replacement := map[string]any{
			"name":   newName,
			"status": newStatus,
		}

		// Perform the PUT (full replacement).
		_, err = d.Update(ctx, table, id, replacement)
		if err != nil {
			rt.Fatalf("Update (PUT) failed: %v", err)
		}

		// GET the record and verify fields match the replacement body.
		got, err := d.GetByID(ctx, table, id)
		if err != nil {
			rt.Fatalf("GetByID after Update failed: %v", err)
		}

		if got["name"] != newName {
			rt.Errorf("name mismatch after PUT: got %q, want %q", got["name"], newName)
		}
		if got["status"] != newStatus {
			rt.Errorf("status mismatch after PUT: got %q, want %q", got["status"], newStatus)
		}
	})
}

// Feature: mariadb-dal-api, Property 12: PATCH partial update preserves unmodified fields
// Validates: Requirements 6.2
func TestPropertyPATCHPartialUpdatePreservesUnmodifiedFields(t *testing.T) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set")
	}

	db, table := setupDB(t)
	d := dal.New(db)
	ctx := context.Background()

	rapid.Check(t, func(rt *rapid.T) {
		// Insert a record with both name and status.
		name := rapid.StringMatching(`[A-Za-z][A-Za-z0-9 ]{0,50}`).Draw(rt, "name")
		status := rapid.StringMatching(`[A-Za-z][A-Za-z0-9]{0,48}`).Draw(rt, "status")

		inserted, err := d.Insert(ctx, table, map[string]any{
			"name":   name,
			"status": status,
		})
		if err != nil {
			rt.Fatalf("Insert failed: %v", err)
		}
		id := fmt.Sprintf("%v", inserted["id"])

		// Randomly choose which fields to patch: name only, status only, or both.
		// 0 = patch name only, 1 = patch status only, 2 = patch both
		patchChoice := rapid.IntRange(0, 2).Draw(rt, "patchChoice")

		newName := rapid.StringMatching(`[A-Za-z][A-Za-z0-9 ]{0,50}`).Draw(rt, "newName")
		newStatus := rapid.StringMatching(`[A-Za-z][A-Za-z0-9]{0,48}`).Draw(rt, "newStatus")

		var patchBody map[string]any
		switch patchChoice {
		case 0:
			patchBody = map[string]any{"name": newName}
		case 1:
			patchBody = map[string]any{"status": newStatus}
		case 2:
			patchBody = map[string]any{"name": newName, "status": newStatus}
		}

		_, err = d.Patch(ctx, table, id, patchBody)
		if err != nil {
			rt.Fatalf("Patch failed: %v", err)
		}

		got, err := d.GetByID(ctx, table, id)
		if err != nil {
			rt.Fatalf("GetByID after Patch failed: %v", err)
		}

		switch patchChoice {
		case 0:
			// Patched name only — status must be unchanged.
			if got["name"] != newName {
				rt.Errorf("patched name mismatch: got %q, want %q", got["name"], newName)
			}
			if got["status"] != status {
				rt.Errorf("unpatched status changed: got %q, want original %q", got["status"], status)
			}
		case 1:
			// Patched status only — name must be unchanged.
			if got["status"] != newStatus {
				rt.Errorf("patched status mismatch: got %q, want %q", got["status"], newStatus)
			}
			if got["name"] != name {
				rt.Errorf("unpatched name changed: got %q, want original %q", got["name"], name)
			}
		case 2:
			// Patched both — both must reflect new values.
			if got["name"] != newName {
				rt.Errorf("patched name mismatch: got %q, want %q", got["name"], newName)
			}
			if got["status"] != newStatus {
				rt.Errorf("patched status mismatch: got %q, want %q", got["status"], newStatus)
			}
		}
	})
}

// Feature: mariadb-dal-api, Property 13: Delete round-trip
// Validates: Requirements 7.1, 7.2
func TestPropertyDeleteRoundTrip(t *testing.T) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set")
	}

	db, table := setupDB(t)
	d := dal.New(db)
	ctx := context.Background()

	rapid.Check(t, func(rt *rapid.T) {
		name := rapid.StringMatching(`[A-Za-z][A-Za-z0-9 ]{0,50}`).Draw(rt, "name")
		status := rapid.StringMatching(`[A-Za-z][A-Za-z0-9]{0,48}`).Draw(rt, "status")

		inserted, err := d.Insert(ctx, table, map[string]any{
			"name":   name,
			"status": status,
		})
		if err != nil {
			rt.Fatalf("Insert failed: %v", err)
		}

		id := fmt.Sprintf("%v", inserted["id"])

		if err := d.Delete(ctx, table, id); err != nil {
			rt.Fatalf("Delete failed: %v", err)
		}

		_, err = d.GetByID(ctx, table, id)
		if err != dal.ErrNotFound {
			rt.Errorf("GetByID after Delete: got %v, want ErrNotFound", err)
		}
	})
}
