package database

import (
	"context"
	"database/sql"
	"time"
)

type ListTables interface {
	Execute(context.Context, ListTablesInput) (ListTablesOutput, error)
}

type ListTablesInput struct {
}

type ListTablesOutput struct {
	Columns []string `json:"columns"`
	Rows    [][]any  `json:"rows"`
}

type listTablesOperation struct {
	db *sql.DB
}

func NewListTables(db *sql.DB) ListTables {
	return &listTablesOperation{
		db: db,
	}
}

func (operation *listTablesOperation) Execute(ctx context.Context, input ListTablesInput) (ListTablesOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	query := "SELECT TABLE_NAME, TABLE_TYPE FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() ORDER BY TABLE_NAME"
	var args []any

	rows, err := operation.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ListTablesOutput{}, err
	}
	defer rows.Close()

	columnsResult, err := rows.Columns()
	if err != nil {
		return ListTablesOutput{}, err
	}
	output := ListTablesOutput{
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
			return ListTablesOutput{}, err
		}
		for i, value := range values {
			if bytes, ok := value.([]byte); ok {
				values[i] = string(bytes)
			}
		}
		output.Rows = append(output.Rows, values)
	}
	if err := rows.Err(); err != nil {
		return ListTablesOutput{}, err
	}
	return output, nil
}
