package database

import (
	"context"
	"testing"
)

func TestColumnValidation(t *testing.T) {
	for _, column := range []ColumnDefinition{
		{Name: "bad;name", Type: "TEXT"},
		{Name: "ok", Type: "TEXT; DROP TABLE users"},
		{Name: "ok", Type: "VARCHAR"},
		{Name: "ok", Type: "VARCHAR", Length: 256},
		{Name: "ok", Type: "DECIMAL", Precision: 4, Scale: 5},
		{Name: "ok", Type: "TEXT", Length: 10},
	} {
		if _, err := columnSQL(column); err == nil {
			t.Errorf("expected validation error for %+v", column)
		}
	}
	for _, column := range []ColumnDefinition{
		{Name: "ok", Type: "INTEGER"},
		{Name: "ok", Type: "VARCHAR", Length: 100},
		{Name: "ok", Type: "DECIMAL", Precision: 10, Scale: 2},
	} {
		if _, err := columnSQL(column); err != nil {
			t.Errorf("unexpected validation error for %+v: %v", column, err)
		}
	}
}

func TestCreateTableValidation(t *testing.T) {
	operation := NewCreateTable(nil)
	for _, input := range []CreateTableInput{
		{Table: "test", Columns: nil},
		{Table: "test", Columns: []ColumnDefinition{{Name: "x", Type: "TEXT"}, {Name: "X", Type: "TEXT"}}},
		{Table: "test", Columns: []ColumnDefinition{{Name: "a", Type: "INTEGER", PrimaryKey: true}, {Name: "b", Type: "INTEGER", PrimaryKey: true}}},
	} {
		if _, err := operation.Execute(context.Background(), input); err == nil {
			t.Errorf("expected validation error for %+v", input)
		}
	}
}

func TestAlterTableRejectsPrimaryKey(t *testing.T) {
	operation := NewAlterTable(nil)
	_, err := operation.Execute(context.Background(), AlterTableInput{
		Table: "test", Column: ColumnDefinition{Name: "id", Type: "INTEGER", PrimaryKey: true},
	})
	if err == nil {
		t.Fatal("expected validation error for primary key")
	}
}
