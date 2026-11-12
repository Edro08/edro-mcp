package tools

import (
	"context"
	"database/sql"
	"postgresql/database"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterSelect(server *mcp.Server, db *sql.DB) {
	operation := database.NewSelect(db)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "select",
		Description: "Read rows with equality filters (AND), limit and offset.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest, input database.SelectInput) (*mcp.CallToolResult, database.SelectOutput, error) {
		output, err := operation.Execute(ctx, input)
		if err != nil {
			return nil, database.SelectOutput{}, err
		}
		return nil, output, nil
	})
}
