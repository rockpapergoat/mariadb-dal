package dal

import (
	"fmt"
	"strings"
)

// quote wraps an identifier in backticks.
func quote(ident string) string {
	return fmt.Sprintf("`%s`", ident)
}

// BuildInsert returns a parameterized INSERT statement.
// Example: INSERT INTO `table` (`col1`, `col2`) VALUES (?, ?)
func BuildInsert(table string, cols []string) string {
	quoted := make([]string, len(cols))
	placeholders := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = quote(c)
		placeholders[i] = "?"
	}
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quote(table),
		strings.Join(quoted, ", "),
		strings.Join(placeholders, ", "),
	)
}

// BuildSelectByID returns a parameterized SELECT by primary key.
// Example: SELECT * FROM `table` WHERE `id` = ?
func BuildSelectByID(table string) string {
	return fmt.Sprintf("SELECT * FROM %s WHERE %s = ?", quote(table), quote("id"))
}

// BuildSelectList returns a parameterized SELECT with optional equality filters,
// LIMIT, and OFFSET placeholders.
// The caller must append filter values followed by limit and offset to the args slice.
//
// With filters:    SELECT * FROM `table` WHERE `col1` = ? AND `col2` = ? LIMIT ? OFFSET ?
// Without filters: SELECT * FROM `table` LIMIT ? OFFSET ?
func BuildSelectList(table string, filterCols []string) string {
	base := fmt.Sprintf("SELECT * FROM %s", quote(table))
	if len(filterCols) > 0 {
		conditions := make([]string, len(filterCols))
		for i, c := range filterCols {
			conditions[i] = fmt.Sprintf("%s = ?", quote(c))
		}
		base += " WHERE " + strings.Join(conditions, " AND ")
	}
	base += " LIMIT ? OFFSET ?"
	return base
}

// BuildUpdate returns a parameterized full-replace UPDATE statement.
// Example: UPDATE `table` SET `col1` = ?, `col2` = ? WHERE `id` = ?
func BuildUpdate(table string, cols []string) string {
	sets := make([]string, len(cols))
	for i, c := range cols {
		sets[i] = fmt.Sprintf("%s = ?", quote(c))
	}
	return fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?",
		quote(table),
		strings.Join(sets, ", "),
		quote("id"),
	)
}

// BuildPatch returns a parameterized partial-update statement.
// Semantically identical to BuildUpdate — the caller passes only the fields to patch.
// Example: UPDATE `table` SET `col1` = ? WHERE `id` = ?
func BuildPatch(table string, cols []string) string {
	return BuildUpdate(table, cols)
}

// BuildDelete returns a parameterized DELETE statement.
// Example: DELETE FROM `table` WHERE `id` = ?
func BuildDelete(table string) string {
	return fmt.Sprintf("DELETE FROM %s WHERE %s = ?", quote(table), quote("id"))
}
