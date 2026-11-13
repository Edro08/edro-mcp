package database

import (
	"fmt"
	"regexp"
	"strings"
)

type ColumnDefinition struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Length     int    `json:"length,omitempty"`
	Precision  int    `json:"precision,omitempty"`
	Scale      int    `json:"scale,omitempty"`
	Required   bool   `json:"required,omitempty"`
	PrimaryKey bool   `json:"primary_key,omitempty"`
}

var schemaIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_$]*$`)

func schemaName(name string) (string, error) {
	if len(name) > 64 || !schemaIdentifier.MatchString(name) {
		return "", fmt.Errorf("invalid table or column name: %q", name)
	}
	return "`" + name + "`", nil
}

func columnSQL(column ColumnDefinition) (string, error) {
	name, err := schemaName(column.Name)
	if err != nil {
		return "", err
	}
	typ := strings.ToUpper(column.Type)
	switch typ {
	case "INTEGER", "BIGINT", "BOOLEAN", "TEXT", "DATE", "TIMESTAMP":
		if column.Length != 0 || column.Precision != 0 || column.Scale != 0 {
			return "", fmt.Errorf("type %s does not accept length, precision or scale", typ)
		}
	case "VARCHAR":
		if column.Length < 1 || column.Length > 255 || column.Precision != 0 || column.Scale != 0 {
			return "", fmt.Errorf("VARCHAR length must be between 1 and 255")
		}
		typ = fmt.Sprintf("VARCHAR(%d)", column.Length)
	case "DECIMAL":
		if column.Precision < 1 || column.Precision > 65 || column.Scale < 0 || column.Scale > column.Precision || column.Scale > 30 || column.Length != 0 {
			return "", fmt.Errorf("DECIMAL precision must be 1..65 and scale must be 0..min(precision, 30)")
		}
		typ = fmt.Sprintf("DECIMAL(%d,%d)", column.Precision, column.Scale)
	default:
		return "", fmt.Errorf("unsupported column type: %q", column.Type)
	}
	definition := name + " " + typ
	if column.Required || column.PrimaryKey {
		definition += " NOT NULL"
	}
	return definition, nil
}
