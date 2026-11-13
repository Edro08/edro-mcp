package tools

import (
	"context"
	"database/sql"
	"postgresql/database"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterListTables(server *mcp.Server, db *sql.DB) {
	operation := database.NewListTables(db)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_tables",
		Description: "List tables in the configured database.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest, input database.ListTablesInput) (*mcp.CallToolResult, database.ListTablesOutput, error) {
		output, err := operation.Execute(ctx, input)
		if err != nil {
			return nil, database.ListTablesOutput{}, err
		}
		return nil, output, nil
	})
}
