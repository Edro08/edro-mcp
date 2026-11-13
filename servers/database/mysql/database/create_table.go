package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type CreateTableInput struct {
	Table   string             `json:"table"`
	Columns []ColumnDefinition `json:"columns"`
}

type CreateTableOutput struct {
	Created bool `json:"created"`
}

type CreateTable interface {
	Execute(context.Context, CreateTableInput) (CreateTableOutput, error)
}

type createTableOperation struct{ db *sql.DB }

func NewCreateTable(db *sql.DB) CreateTable { return &createTableOperation{db: db} }

func (operation *createTableOperation) Execute(ctx context.Context, input CreateTableInput) (CreateTableOutput, error) {
	table, err := schemaName(input.Table)
	if err != nil {
		return CreateTableOutput{}, err
	}
	if len(input.Columns) == 0 {
		return CreateTableOutput{}, fmt.Errorf("columns must not be empty")
	}
	definitions := make([]string, 0, len(input.Columns)+1)
	seen := make(map[string]bool)
	var primaryKey string
	for _, column := range input.Columns {
		key := strings.ToLower(column.Name)
		if seen[key] {
			return CreateTableOutput{}, fmt.Errorf("duplicate column: %q", column.Name)
		}
		seen[key] = true
		definition, err := columnSQL(column)
		if err != nil {
			return CreateTableOutput{}, err
		}
		definitions = append(definitions, definition)
		if column.PrimaryKey {
			if primaryKey != "" {
				return CreateTableOutput{}, fmt.Errorf("only one primary key column is supported")
			}
			primaryKey, _ = schemaName(column.Name)
		}
	}
	if primaryKey != "" {
		definitions = append(definitions, "PRIMARY KEY ("+primaryKey+")")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, err = operation.db.ExecContext(ctx, "CREATE TABLE "+table+" ("+strings.Join(definitions, ", ")+")")
	if err != nil {
		return CreateTableOutput{}, err
	}
	return CreateTableOutput{Created: true}, nil
}
