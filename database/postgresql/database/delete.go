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

type Delete interface {
	Execute(context.Context, DeleteInput) (DeleteOutput, error)
}

type DeleteInput struct {
	Table string         `json:"table"`
	Where map[string]any `json:"where"`
}

type DeleteOutput struct {
	RowsAffected int64 `json:"rows_affected"`
}

type deleteOperation struct {
	db *sql.DB
}

func NewDelete(db *sql.DB) Delete {
	return &deleteOperation{
		db: db,
	}
}

func (operation *deleteOperation) Execute(ctx context.Context, input DeleteInput) (DeleteOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	table, err := operation.identifier(input.Table)
	if err != nil {
		return DeleteOutput{}, err
	}
	if len(input.Where) == 0 {
		return DeleteOutput{}, fmt.Errorf("where must not be empty")
	}
	conditions, args, err := operation.conditions(input.Where)
	if err != nil {
		return DeleteOutput{}, err
	}
	query := "DELETE FROM " + table + " WHERE " + conditions

	result, err := operation.db.ExecContext(ctx, query, args...)
	if err != nil {
		return DeleteOutput{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return DeleteOutput{}, err
	}

	return DeleteOutput{
		RowsAffected: affected,
	}, nil
}

func (operation *deleteOperation) identifier(name string) (string, error) {
	valid := regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_$]*$")
	if !valid.MatchString(name) || len(name) > 63 {
		return "", fmt.Errorf("invalid table or column name: %q", name)
	}
	return `"` + name + `"`, nil
}

func (operation *deleteOperation) conditions(values map[string]any) (string, []any, error) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys))
	for _, key := range keys {
		column, err := operation.identifier(key)
		if err != nil {
			return "", nil, err
		}
		value := values[key]
		switch value.(type) {
		case nil, string, bool, float64, float32, int, int32, int64:
		default:
			return "", nil, fmt.Errorf("column %s requires a scalar value", key)
		}
		if value == nil {
			parts = append(parts, column+" IS NULL")
			continue
		}
		parts = append(parts, fmt.Sprintf("%s = $%d", column, len(args)+1))
		args = append(args, value)
	}
	return strings.Join(parts, " AND "), args, nil
}
