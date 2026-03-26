package dal

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/go-sql-driver/mysql"
)

// TestMapDBError_SQLErrNoRows verifies that sql.ErrNoRows maps to ErrNotFound.
func TestMapDBError_SQLErrNoRows(t *testing.T) {
	got := mapDBError(sql.ErrNoRows)
	if !errors.Is(got, ErrNotFound) {
		t.Errorf("mapDBError(sql.ErrNoRows) = %v; want ErrNotFound", got)
	}
}

// TestMapDBError_MySQL1146_TableNotFound verifies that MySQL error 1146 maps to ErrNotFound.
func TestMapDBError_MySQL1146_TableNotFound(t *testing.T) {
	err := &mysql.MySQLError{Number: 1146, Message: "Table 'db.foo' doesn't exist"}
	got := mapDBError(err)
	if !errors.Is(got, ErrNotFound) {
		t.Errorf("mapDBError(MySQLError 1146) = %v; want ErrNotFound", got)
	}
}

// TestMapDBError_MySQL1062_DuplicateKey verifies that MySQL error 1062 maps to ErrConflict.
func TestMapDBError_MySQL1062_DuplicateKey(t *testing.T) {
	err := &mysql.MySQLError{Number: 1062, Message: "Duplicate entry '1' for key 'PRIMARY'"}
	got := mapDBError(err)
	if !errors.Is(got, ErrConflict) {
		t.Errorf("mapDBError(MySQLError 1062) = %v; want ErrConflict", got)
	}
}

// TestMapDBError_MySQL1451_FKConstraint verifies that MySQL error 1451 maps to ErrConflict.
func TestMapDBError_MySQL1451_FKConstraint(t *testing.T) {
	err := &mysql.MySQLError{Number: 1451, Message: "Cannot delete or update a parent row"}
	got := mapDBError(err)
	if !errors.Is(got, ErrConflict) {
		t.Errorf("mapDBError(MySQLError 1451) = %v; want ErrConflict", got)
	}
}

// TestMapDBError_MySQL1452_FKChildRow verifies that MySQL error 1452 maps to ErrConflict.
func TestMapDBError_MySQL1452_FKChildRow(t *testing.T) {
	err := &mysql.MySQLError{Number: 1452, Message: "Cannot add or update a child row"}
	got := mapDBError(err)
	if !errors.Is(got, ErrConflict) {
		t.Errorf("mapDBError(MySQLError 1452) = %v; want ErrConflict", got)
	}
}

// TestMapDBError_NilReturnsNil verifies that nil input returns nil.
func TestMapDBError_NilReturnsNil(t *testing.T) {
	if got := mapDBError(nil); got != nil {
		t.Errorf("mapDBError(nil) = %v; want nil", got)
	}
}

// TestMapDBError_UnknownMySQLError verifies that an unmapped MySQL error number is returned as-is.
func TestMapDBError_UnknownMySQLError(t *testing.T) {
	err := &mysql.MySQLError{Number: 9999, Message: "some unknown error"}
	got := mapDBError(err)
	if errors.Is(got, ErrNotFound) || errors.Is(got, ErrConflict) {
		t.Errorf("mapDBError(MySQLError 9999) = %v; want original error (not a sentinel)", got)
	}
	if got != err {
		t.Errorf("mapDBError(MySQLError 9999) = %v; want original error %v", got, err)
	}
}

// TestMapDBError_UnknownGenericError verifies that an unrecognized generic error is returned as-is.
func TestMapDBError_UnknownGenericError(t *testing.T) {
	err := errors.New("some generic db error")
	got := mapDBError(err)
	if got != err {
		t.Errorf("mapDBError(generic error) = %v; want original error %v", got, err)
	}
}

// TestMapDBError_WrappedSQLErrNoRows verifies that a wrapped sql.ErrNoRows still maps to ErrNotFound.
func TestMapDBError_WrappedSQLErrNoRows(t *testing.T) {
	wrapped := errors.Join(errors.New("context"), sql.ErrNoRows)
	got := mapDBError(wrapped)
	if !errors.Is(got, ErrNotFound) {
		t.Errorf("mapDBError(wrapped sql.ErrNoRows) = %v; want ErrNotFound", got)
	}
}
