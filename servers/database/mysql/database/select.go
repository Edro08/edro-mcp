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

type Select interface {
	Execute(context.Context, SelectInput) (SelectOutput, error)
}

type SelectInput struct {
	Table   string         `json:"table"`
	Columns []string       `json:"columns,omitempty"`
	Where   map[string]any `json:"where,omitempty" jsonschema:"Equality filters joined with AND"`
	Limit   int            `json:"limit,omitempty" jsonschema:"Default 100, maximum 1000"`
	Offset  int            `json:"offset,omitempty"`
}

type SelectOutput struct {
	Columns []string `json:"columns"`
	Rows    [][]any  `json:"rows"`
}

type selectOperation struct {
	db *sql.DB
}

func NewSelect(db *sql.DB) Select {
	return &selectOperation{
		db: db,
	}
}

func (operation *selectOperation) Execute(ctx context.Context, input SelectInput) (SelectOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	table, err := operation.identifier(input.Table)
	if err != nil {
		return SelectOutput{}, err
	}
	columns := []string{"*"}
	if len(input.Columns) > 0 {
		columns = make([]string, 0, len(input.Columns))
		for _, name := range input.Columns {
			column, err := operation.identifier(name)
			if err != nil {
				return SelectOutput{}, err
			}
			columns = append(columns, column)
		}
	}
	query := "SELECT " + strings.Join(columns, ", ") + " FROM " + table
	conditions, args, err := operation.conditions(input.Where)
	if err != nil {
		return SelectOutput{}, err
	}
	if conditions != "" {
		query += " WHERE " + conditions
	}
	if input.Limit == 0 {
		input.Limit = 100
	}
	if input.Limit < 1 || input.Limit > 1000 || input.Offset < 0 {
		return SelectOutput{}, fmt.Errorf("limit must be between 1 and 1000 and offset must be nonnegative")
	}
	query += " LIMIT ? OFFSET ?"
	args = append(args, input.Limit, input.Offset)

	rows, err := operation.db.QueryContext(ctx, query, args...)
	if err != nil {
		return SelectOutput{}, err
	}
	defer rows.Close()

	columnsResult, err := rows.Columns()
	if err != nil {
		return SelectOutput{}, err
	}
	output := SelectOutput{
		Columns: columnsResult,
		Rows:    make([][]any, 0),
	}
	for rows.Next() {
		values := make([]any, len(columnsResult))
		destinations := make([]any, len(columnsResult))
		for i := range values {
			destinations[i] = &values[i]
		}
		if err := rows.Scan(destinations...); err != nil {
			return SelectOutput{}, err
		}
		for i, value := range values {
			if bytes, ok := value.([]byte); ok {
				values[i] = string(bytes)
			}
		}
		output.Rows = append(output.Rows, values)
	}
	if err := rows.Err(); err != nil {
		return SelectOutput{}, err
	}
	return output, nil
}

func (operation *selectOperation) identifier(name string) (string, error) {
	valid := regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_$]*$")
	if !valid.MatchString(name) || len(name) > 64 {
		return "", fmt.Errorf("invalid table or column name: %q", name)
	}
	return "`" + name + "`", nil
}

func (operation *selectOperation) conditions(values map[string]any) (string, []any, error) {
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
		parts = append(parts, column+" = ?")
		args = append(args, value)
	}
	return strings.Join(parts, " AND "), args, nil
}
