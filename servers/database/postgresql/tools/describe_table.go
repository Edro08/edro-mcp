package tools

import (
	"context"
	"database/sql"
	"postgresql/database"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterDescribeTable(server *mcp.Server, db *sql.DB) {
	operation := database.NewDescribeTable(db)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "describe_table",
		Description: "Describe the columns of a table.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest, input database.DescribeTableInput) (*mcp.CallToolResult, database.DescribeTableOutput, error) {
		output, err := operation.Execute(ctx, input)
		if err != nil {
			return nil, database.DescribeTableOutput{}, err
		}
		return nil, output, nil
	})
}
