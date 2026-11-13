package tools

import (
	"context"
	"database/sql"
	"mysql/database"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterCreateTable(server *mcp.Server, db *sql.DB) {
	operation := database.NewCreateTable(db)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_table",
		Description: "Create a table with validated column types and an optional single-column primary key.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false},
	}, func(ctx context.Context, request *mcp.CallToolRequest, input database.CreateTableInput) (*mcp.CallToolResult, database.CreateTableOutput, error) {
		output, err := operation.Execute(ctx, input)
		if err != nil {
			return nil, database.CreateTableOutput{}, err
		}
		return nil, output, nil
	})
}
