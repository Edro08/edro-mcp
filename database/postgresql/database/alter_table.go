package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type AlterTableInput struct {
	Table  string           `json:"table"`
	Column ColumnDefinition `json:"column"`
}

type AlterTableOutput struct {
	Altered bool `json:"altered"`
}

type AlterTable interface {
	Execute(context.Context, AlterTableInput) (AlterTableOutput, error)
}

type alterTableOperation struct{ db *sql.DB }

func NewAlterTable(db *sql.DB) AlterTable { return &alterTableOperation{db: db} }

func (operation *alterTableOperation) Execute(ctx context.Context, input AlterTableInput) (AlterTableOutput, error) {
	table, err := schemaName(input.Table)
	if err != nil {
		return AlterTableOutput{}, err
	}
	if input.Column.PrimaryKey {
		return AlterTableOutput{}, fmt.Errorf("adding a primary key to an existing table is not supported")
	}
	definition, err := columnSQL(input.Column)
	if err != nil {
		return AlterTableOutput{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, err = operation.db.ExecContext(ctx, "ALTER TABLE "+table+" ADD COLUMN "+definition)
	if err != nil {
		return AlterTableOutput{}, err
	}
	return AlterTableOutput{Altered: true}, nil
}
