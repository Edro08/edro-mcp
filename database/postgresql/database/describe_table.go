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
	query := `SELECT c.column_name, c.data_type, c.is_nullable, c.column_default,
       CASE WHEN pk.column_name IS NOT NULL THEN 'PRI' ELSE '' END AS column_key,
       CASE WHEN c.is_identity = 'YES' THEN 'IDENTITY' ELSE '' END AS extra
FROM information_schema.columns c
LEFT JOIN (
    SELECT kcu.table_schema, kcu.table_name, kcu.column_name
    FROM information_schema.table_constraints tc
    JOIN information_schema.key_column_usage kcu
      ON tc.constraint_catalog = kcu.constraint_catalog
     AND tc.constraint_schema = kcu.constraint_schema
     AND tc.constraint_name = kcu.constraint_name
    WHERE tc.constraint_type = 'PRIMARY KEY'
) pk ON pk.table_schema = c.table_schema AND pk.table_name = c.table_name AND pk.column_name = c.column_name
WHERE c.table_schema = current_schema() AND c.table_name = $1
ORDER BY c.ordinal_position`
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
	if !valid.MatchString(name) || len(name) > 63 {
		return "", fmt.Errorf("invalid table or column name: %q", name)
	}
	return `"` + name + `"`, nil
}
