package dal

import (
	"strings"
	"testing"
)

// countPlaceholders counts the number of '?' in a SQL string.
func countPlaceholders(sql string) int {
	return strings.Count(sql, "?")
}

// TestBuildInsert verifies INSERT SQL shape and placeholder count.
func TestBuildInsert(t *testing.T) {
	tests := []struct {
		name         string
		table        string
		cols         []string
		wantContains []string
		wantPhCount  int
	}{
		{
			name:         "single column",
			table:        "users",
			cols:         []string{"name"},
			wantContains: []string{"INSERT INTO `users`", "(`name`)", "VALUES (?)", "?"},
			wantPhCount:  1,
		},
		{
			name:         "multiple columns",
			table:        "orders",
			cols:         []string{"user_id", "product", "qty"},
			wantContains: []string{"INSERT INTO `orders`", "`user_id`", "`product`", "`qty`", "VALUES (?, ?, ?)"},
			wantPhCount:  3,
		},
		{
			name:         "table name is backtick-quoted",
			table:        "my_table",
			cols:         []string{"col"},
			wantContains: []string{"`my_table`"},
			wantPhCount:  1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildInsert(tc.table, tc.cols)
			for _, want := range tc.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("BuildInsert(%q, %v) = %q; want it to contain %q", tc.table, tc.cols, got, want)
				}
			}
			if ph := countPlaceholders(got); ph != tc.wantPhCount {
				t.Errorf("BuildInsert(%q, %v) placeholder count = %d; want %d (sql: %q)", tc.table, tc.cols, ph, tc.wantPhCount, got)
			}
		})
	}
}

// TestBuildSelectByID verifies SELECT by ID SQL shape and placeholder count.
func TestBuildSelectByID(t *testing.T) {
	tests := []struct {
		name        string
		table       string
		wantSQL     string
		wantPhCount int
	}{
		{
			name:        "basic table",
			table:       "users",
			wantSQL:     "SELECT * FROM `users` WHERE `id` = ?",
			wantPhCount: 1,
		},
		{
			name:        "underscore table name",
			table:       "order_items",
			wantSQL:     "SELECT * FROM `order_items` WHERE `id` = ?",
			wantPhCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildSelectByID(tc.table)
			if got != tc.wantSQL {
				t.Errorf("BuildSelectByID(%q) = %q; want %q", tc.table, got, tc.wantSQL)
			}
			if ph := countPlaceholders(got); ph != tc.wantPhCount {
				t.Errorf("BuildSelectByID(%q) placeholder count = %d; want %d", tc.table, ph, tc.wantPhCount)
			}
		})
	}
}

// TestBuildSelectListNoFilters verifies SELECT without filters has LIMIT ? OFFSET ? only.
func TestBuildSelectListNoFilters(t *testing.T) {
	got := BuildSelectList("products", nil)
	want := "SELECT * FROM `products` LIMIT ? OFFSET ?"
	if got != want {
		t.Errorf("BuildSelectList(no filters) = %q; want %q", got, want)
	}
	if ph := countPlaceholders(got); ph != 2 {
		t.Errorf("BuildSelectList(no filters) placeholder count = %d; want 2", ph)
	}
}

// TestBuildSelectListEmptyFilters verifies empty filter slice behaves same as nil.
func TestBuildSelectListEmptyFilters(t *testing.T) {
	got := BuildSelectList("products", []string{})
	want := "SELECT * FROM `products` LIMIT ? OFFSET ?"
	if got != want {
		t.Errorf("BuildSelectList(empty filters) = %q; want %q", got, want)
	}
	if ph := countPlaceholders(got); ph != 2 {
		t.Errorf("BuildSelectList(empty filters) placeholder count = %d; want 2", ph)
	}
}

// TestBuildSelectListWithFilters verifies SELECT with filters includes WHERE clause and correct placeholder count.
func TestBuildSelectListWithFilters(t *testing.T) {
	tests := []struct {
		name        string
		table       string
		filters     []string
		wantPhCount int // filter placeholders + 2 (LIMIT + OFFSET)
		wantParts   []string
	}{
		{
			name:        "one filter",
			table:       "users",
			filters:     []string{"status"},
			wantPhCount: 3, // 1 filter + LIMIT + OFFSET
			wantParts:   []string{"WHERE `status` = ?", "LIMIT ? OFFSET ?"},
		},
		{
			name:        "two filters",
			table:       "orders",
			filters:     []string{"user_id", "status"},
			wantPhCount: 4, // 2 filters + LIMIT + OFFSET
			wantParts:   []string{"WHERE `user_id` = ? AND `status` = ?", "LIMIT ? OFFSET ?"},
		},
		{
			name:        "three filters",
			table:       "items",
			filters:     []string{"a", "b", "c"},
			wantPhCount: 5, // 3 filters + LIMIT + OFFSET
			wantParts:   []string{"`a` = ?", "`b` = ?", "`c` = ?", "LIMIT ? OFFSET ?"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildSelectList(tc.table, tc.filters)
			for _, part := range tc.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("BuildSelectList(%q, %v) = %q; want it to contain %q", tc.table, tc.filters, got, part)
				}
			}
			if ph := countPlaceholders(got); ph != tc.wantPhCount {
				t.Errorf("BuildSelectList(%q, %v) placeholder count = %d; want %d (sql: %q)", tc.table, tc.filters, ph, tc.wantPhCount, got)
			}
		})
	}
}

// TestBuildUpdate verifies full-replace UPDATE SQL shape and placeholder count.
func TestBuildUpdate(t *testing.T) {
	tests := []struct {
		name        string
		table       string
		cols        []string
		wantParts   []string
		wantPhCount int // len(cols) SET placeholders + 1 WHERE id placeholder
	}{
		{
			name:        "single column",
			table:       "users",
			cols:        []string{"name"},
			wantParts:   []string{"UPDATE `users`", "SET `name` = ?", "WHERE `id` = ?"},
			wantPhCount: 2,
		},
		{
			name:        "multiple columns",
			table:       "products",
			cols:        []string{"title", "price", "stock"},
			wantParts:   []string{"UPDATE `products`", "`title` = ?", "`price` = ?", "`stock` = ?", "WHERE `id` = ?"},
			wantPhCount: 4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildUpdate(tc.table, tc.cols)
			for _, part := range tc.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("BuildUpdate(%q, %v) = %q; want it to contain %q", tc.table, tc.cols, got, part)
				}
			}
			if ph := countPlaceholders(got); ph != tc.wantPhCount {
				t.Errorf("BuildUpdate(%q, %v) placeholder count = %d; want %d (sql: %q)", tc.table, tc.cols, ph, tc.wantPhCount, got)
			}
		})
	}
}

// TestBuildPatch verifies PATCH (partial update) SQL shape and placeholder count.
// BuildPatch delegates to BuildUpdate, so the shape is identical.
func TestBuildPatch(t *testing.T) {
	tests := []struct {
		name        string
		table       string
		cols        []string
		wantParts   []string
		wantPhCount int
	}{
		{
			name:        "patch single field",
			table:       "users",
			cols:        []string{"email"},
			wantParts:   []string{"UPDATE `users`", "SET `email` = ?", "WHERE `id` = ?"},
			wantPhCount: 2,
		},
		{
			name:        "patch two fields",
			table:       "profiles",
			cols:        []string{"bio", "avatar"},
			wantParts:   []string{"UPDATE `profiles`", "`bio` = ?", "`avatar` = ?", "WHERE `id` = ?"},
			wantPhCount: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildPatch(tc.table, tc.cols)
			for _, part := range tc.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("BuildPatch(%q, %v) = %q; want it to contain %q", tc.table, tc.cols, got, part)
				}
			}
			if ph := countPlaceholders(got); ph != tc.wantPhCount {
				t.Errorf("BuildPatch(%q, %v) placeholder count = %d; want %d (sql: %q)", tc.table, tc.cols, ph, tc.wantPhCount, got)
			}
		})
	}
}

// TestBuildPatchMatchesBuildUpdate verifies PATCH and UPDATE produce identical SQL for the same inputs.
func TestBuildPatchMatchesBuildUpdate(t *testing.T) {
	table := "items"
	cols := []string{"name", "value"}
	if patch, update := BuildPatch(table, cols), BuildUpdate(table, cols); patch != update {
		t.Errorf("BuildPatch and BuildUpdate differ for same inputs: patch=%q update=%q", patch, update)
	}
}

// TestBuildDelete verifies DELETE SQL shape and placeholder count.
func TestBuildDelete(t *testing.T) {
	tests := []struct {
		name    string
		table   string
		wantSQL string
	}{
		{
			name:    "basic table",
			table:   "users",
			wantSQL: "DELETE FROM `users` WHERE `id` = ?",
		},
		{
			name:    "underscore table",
			table:   "order_items",
			wantSQL: "DELETE FROM `order_items` WHERE `id` = ?",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildDelete(tc.table)
			if got != tc.wantSQL {
				t.Errorf("BuildDelete(%q) = %q; want %q", tc.table, got, tc.wantSQL)
			}
			if ph := countPlaceholders(got); ph != 1 {
				t.Errorf("BuildDelete(%q) placeholder count = %d; want 1", tc.table, ph)
			}
		})
	}
}
