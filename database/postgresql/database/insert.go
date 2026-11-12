package database

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Insert interface {
	Execute(context.Context, InsertInput) (InsertOutput, error)
}

type InsertInput struct {
	Table  string         `json:"table"`
	Values map[string]any `json:"values"`
}

type InsertOutput struct {
	RowsAffected int64 `json:"rows_affected"`
}

type insertOperation struct {
	db *sql.DB
}

func NewInsert(db *sql.DB) Insert {
	return &insertOperation{
		db: db,
	}
}

func (operation *insertOperation) Execute(ctx context.Context, input InsertInput) (InsertOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	table, err := operation.identifier(input.Table)
	if err != nil {
		return InsertOutput{}, err
	}
	if len(input.Values) == 0 {
		return InsertOutput{}, fmt.Errorf("values must not be empty")
	}
	columns, placeholders, args, err := operation.assignments(input.Values)
	if err != nil {
		return InsertOutput{}, err
	}
	query := "INSERT INTO " + table + " (" + columns + ") VALUES (" + placeholders + ")"

	result, err := operation.db.ExecContext(ctx, query, args...)
	if err != nil {
		return InsertOutput{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return InsertOutput{}, err
	}
	return InsertOutput{
		RowsAffected: affected,
	}, nil
}

func (operation *insertOperation) identifier(name string) (string, error) {
	valid := regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_$]*$")
	if !valid.MatchString(name) || len(name) > 63 {
		return "", fmt.Errorf("invalid table or column name: %q", name)
	}
	return `"` + name + `"`, nil
}

func (operation *insertOperation) assignments(values map[string]any) (string, string, []any, error) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	columns := make([]string, 0, len(keys))
	placeholders := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys))
	for _, key := range keys {
		column, err := operation.identifier(key)
		if err != nil {
			return "", "", nil, err
		}
		value := values[key]
		switch value.(type) {
		case nil, string, bool, float64, float32, int, int32, int64:
		default:
			return "", "", nil, fmt.Errorf("column %s requires a scalar value", key)
		}
		columns = append(columns, column)
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)+1))
		args = append(args, value)
	}
	return strings.Join(columns, ", "), strings.Join(placeholders, ", "), args, nil
}
