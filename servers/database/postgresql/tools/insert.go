package tools

import (
	"context"
	"database/sql"
	"postgresql/database"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterInsert(server *mcp.Server, db *sql.DB) {
	operation := database.NewInsert(db)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "insert",
		Description: "Insert one row into a table.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: false,
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest, input database.InsertInput) (*mcp.CallToolResult, database.InsertOutput, error) {
		output, err := operation.Execute(ctx, input)
		if err != nil {
			return nil, database.InsertOutput{}, err
		}
		return nil, output, nil
	})
}
