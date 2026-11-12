package tools

import (
	"context"
	"database/sql"
	"postgresql/database"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterDelete(server *mcp.Server, db *sql.DB) {
	operation := database.NewDelete(db)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete",
		Description: "Delete rows matching a required nonempty equality filter.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: false,
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest, input database.DeleteInput) (*mcp.CallToolResult, database.DeleteOutput, error) {
		output, err := operation.Execute(ctx, input)
		if err != nil {
			return nil, database.DeleteOutput{}, err
		}
		return nil, output, nil
	})
}
