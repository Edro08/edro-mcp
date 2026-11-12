package tools

import (
	"context"
	"database/sql"
	"mysql/database"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterUpdate(server *mcp.Server, db *sql.DB) {
	operation := database.NewUpdate(db)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update",
		Description: "Update rows matching a required nonempty equality filter.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: false,
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest, input database.UpdateInput) (*mcp.CallToolResult, database.UpdateOutput, error) {
		output, err := operation.Execute(ctx, input)
		if err != nil {
			return nil, database.UpdateOutput{}, err
		}
		return nil, output, nil
	})
}
