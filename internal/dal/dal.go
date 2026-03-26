package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"github.com/go-sql-driver/mysql"
)

// Sentinel errors returned by the DAL layer.
var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

// MySQL/MariaDB error numbers.
const (
	errTableNotFound = 1146
	errDuplicateKey  = 1062
	errFKConstraint  = 1451
	errFKChildRow    = 1452
)

// DAL defines the database access layer interface.
type DAL interface {
	Insert(ctx context.Context, table string, data map[string]any) (map[string]any, error)
	GetByID(ctx context.Context, table string, id string) (map[string]any, error)
	List(ctx context.Context, table string, filters map[string]string, limit, offset int) ([]map[string]any, error)
	Update(ctx context.Context, table string, id string, data map[string]any) (map[string]any, error)
	Patch(ctx context.Context, table string, id string, data map[string]any) (map[string]any, error)
	Delete(ctx context.Context, table string, id string) error
	Ping(ctx context.Context) error
}

// DB is the concrete DAL implementation backed by *sql.DB.
type DB struct {
	db *sql.DB
}

// New creates a new DB DAL wrapping the provided *sql.DB.
func New(db *sql.DB) *DB {
	return &DB{db: db}
}

// mapDBError converts known MySQL/MariaDB errors to DAL sentinel errors.
func mapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case errTableNotFound:
			return ErrNotFound
		case errDuplicateKey, errFKConstraint, errFKChildRow:
			return ErrConflict
		}
	}
	return err
}

// scanRows scans a *sql.Rows into a slice of map[string]any.
// Each column value is stored as a string, or nil if NULL.
func scanRows(rows *sql.Rows) ([]map[string]any, error) {
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	cols := make([]string, len(colTypes))
	for i, ct := range colTypes {
		cols[i] = ct.Name()
	}

	var results []map[string]any
	for rows.Next() {
		raw := make([]sql.RawBytes, len(cols))
		dest := make([]any, len(cols))
		for i := range raw {
			dest[i] = &raw[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(cols))
		for i, col := range cols {
			if raw[i] == nil {
				row[col] = nil
			} else {
				row[col] = string(raw[i])
			}
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// Insert inserts a new row into table and returns the created record.
func (d *DB) Insert(ctx context.Context, table string, data map[string]any) (map[string]any, error) {
	// Sort column names for deterministic query.
	cols := make([]string, 0, len(data))
	for k := range data {
		cols = append(cols, k)
	}
	sort.Strings(cols)

	args := make([]any, len(cols))
	for i, c := range cols {
		args[i] = data[c]
	}

	query := BuildInsert(table, cols)
	result, err := d.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, mapDBError(err)
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("retrieving last insert id: %w", err)
	}

	return d.GetByID(ctx, table, fmt.Sprintf("%d", lastID))
}

// GetByID retrieves a single row from table by its id column.
func (d *DB) GetByID(ctx context.Context, table string, id string) (map[string]any, error) {
	query := BuildSelectByID(table)
	rows, err := d.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()

	results, err := scanRows(rows)
	if err != nil {
		return nil, mapDBError(err)
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return results[0], nil
}

// List retrieves rows from table applying equality filters, limit, and offset.
func (d *DB) List(ctx context.Context, table string, filters map[string]string, limit, offset int) ([]map[string]any, error) {
	filterCols := make([]string, 0, len(filters))
	for k := range filters {
		filterCols = append(filterCols, k)
	}
	sort.Strings(filterCols)

	args := make([]any, 0, len(filterCols)+2)
	for _, c := range filterCols {
		args = append(args, filters[c])
	}
	args = append(args, limit, offset)

	query := BuildSelectList(table, filterCols)
	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()

	results, err := scanRows(rows)
	if err != nil {
		return nil, mapDBError(err)
	}
	if results == nil {
		results = []map[string]any{}
	}
	return results, nil
}

// Update performs a full replacement of the row identified by id in table.
func (d *DB) Update(ctx context.Context, table string, id string, data map[string]any) (map[string]any, error) {
	cols := make([]string, 0, len(data))
	for k := range data {
		cols = append(cols, k)
	}
	sort.Strings(cols)

	args := make([]any, 0, len(cols)+1)
	for _, c := range cols {
		args = append(args, data[c])
	}
	args = append(args, id)

	query := BuildUpdate(table, cols)
	result, err := d.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, mapDBError(err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("checking rows affected: %w", err)
	}
	if affected == 0 {
		return nil, ErrNotFound
	}

	return d.GetByID(ctx, table, id)
}

// Patch performs a partial update of the row identified by id in table.
func (d *DB) Patch(ctx context.Context, table string, id string, data map[string]any) (map[string]any, error) {
	cols := make([]string, 0, len(data))
	for k := range data {
		cols = append(cols, k)
	}
	sort.Strings(cols)

	args := make([]any, 0, len(cols)+1)
	for _, c := range cols {
		args = append(args, data[c])
	}
	args = append(args, id)

	query := BuildPatch(table, cols)
	result, err := d.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, mapDBError(err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("checking rows affected: %w", err)
	}
	if affected == 0 {
		return nil, ErrNotFound
	}

	return d.GetByID(ctx, table, id)
}

// Delete removes the row identified by id from table.
func (d *DB) Delete(ctx context.Context, table string, id string) error {
	query := BuildDelete(table)
	result, err := d.db.ExecContext(ctx, query, id)
	if err != nil {
		return mapDBError(err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// Ping checks the database connection is alive.
func (d *DB) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}
