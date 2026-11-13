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

type Update interface {
	Execute(context.Context, UpdateInput) (UpdateOutput, error)
}

type UpdateInput struct {
	Table  string         `json:"table"`
	Values map[string]any `json:"values"`
	Where  map[string]any `json:"where"`
}

type UpdateOutput struct {
	RowsAffected int64 `json:"rows_affected"`
}

type updateOperation struct {
	db *sql.DB
}

func NewUpdate(db *sql.DB) Update {
	return &updateOperation{
		db: db,
	}
}

func (operation *updateOperation) Execute(ctx context.Context, input UpdateInput) (UpdateOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	table, err := operation.identifier(input.Table)
	if err != nil {
		return UpdateOutput{}, err
	}
	if len(input.Values) == 0 || len(input.Where) == 0 {
		return UpdateOutput{}, fmt.Errorf("values and where must not be empty")
	}
	assignments, args, err := operation.assignments(input.Values)
	if err != nil {
		return UpdateOutput{}, err
	}
	conditions, filters, err := operation.conditions(input.Where, len(args))
	if err != nil {
		return UpdateOutput{}, err
	}
	query := "UPDATE " + table + " SET " + assignments + " WHERE " + conditions
	args = append(args, filters...)

	result, err := operation.db.ExecContext(ctx, query, args...)
	if err != nil {
		return UpdateOutput{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return UpdateOutput{}, err
	}

	return UpdateOutput{
		RowsAffected: affected,
	}, nil
}

func (operation *updateOperation) identifier(name string) (string, error) {
	valid := regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_$]*$")
	if !valid.MatchString(name) || len(name) > 63 {
		return "", fmt.Errorf("invalid table or column name: %q", name)
	}
	return `"` + name + `"`, nil
}

func (operation *updateOperation) assignments(values map[string]any) (string, []any, error) {
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
		parts = append(parts, fmt.Sprintf("%s = $%d", column, len(args)+1))
		args = append(args, value)
	}
	return strings.Join(parts, ", "), args, nil
}

func (operation *updateOperation) conditions(values map[string]any, start int) (string, []any, error) {
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
		parts = append(parts, fmt.Sprintf("%s = $%d", column, start+len(args)+1))
		args = append(args, value)
	}
	return strings.Join(parts, " AND "), args, nil
}
