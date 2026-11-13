package tools

import (
	"context"
	"database/sql"
	"postgresql/database"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterAlterTable(server *mcp.Server, db *sql.DB) {
	operation := database.NewAlterTable(db)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "alter_table",
		Description: "Add one validated column to an existing table; does not drop or modify columns.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false},
	}, func(ctx context.Context, request *mcp.CallToolRequest, input database.AlterTableInput) (*mcp.CallToolResult, database.AlterTableOutput, error) {
		output, err := operation.Execute(ctx, input)
		if err != nil {
			return nil, database.AlterTableOutput{}, err
		}
		return nil, output, nil
	})
}
