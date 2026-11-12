package database

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"time"
)

type DescribeTable interface {
	Execute(context.Context, DescribeTableInput) (DescribeTableOutput, error)
}

type DescribeTableInput struct {
	Table string `json:"table"`
}

type DescribeTableOutput struct {
	Columns []string `json:"columns"`
	Rows    [][]any  `json:"rows"`
}

type describeTableOperation struct {
	db *sql.DB
}

func NewDescribeTable(db *sql.DB) DescribeTable {
	return &describeTableOperation{
		db: db,
	}
}

func (operation *describeTableOperation) Execute(ctx context.Context, input DescribeTableInput) (DescribeTableOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if _, err := operation.identifier(input.Table); err != nil {
		return DescribeTableOutput{}, err
	}
	query := "SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_DEFAULT, COLUMN_KEY, EXTRA FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? ORDER BY ORDINAL_POSITION"
	args := []any{input.Table}

	rows, err := operation.db.QueryContext(ctx, query, args...)
	if err != nil {
		return DescribeTableOutput{}, err
	}
	defer rows.Close()

	columnsResult, err := rows.Columns()
	if err != nil {
		return DescribeTableOutput{}, err
	}
	output := DescribeTableOutput{
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
			return DescribeTableOutput{}, err
		}
		for i, value := range values {
			if bytes, ok := value.([]byte); ok {
				values[i] = string(bytes)
			}
		}
		output.Rows = append(output.Rows, values)
	}
	if err := rows.Err(); err != nil {
		return DescribeTableOutput{}, err
	}
	return output, nil
}

func (operation *describeTableOperation) identifier(name string) (string, error) {
	valid := regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_$]*$")
	if !valid.MatchString(name) || len(name) > 64 {
		return "", fmt.Errorf("invalid table or column name: %q", name)
	}
	return "`" + name + "`", nil
}
