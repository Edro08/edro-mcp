package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"mysql/tools"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	env_database_host        = "DATABASE_HOST"
	env_database_port        = "DATABASE_PORT"
	env_database_username    = "DATABASE_USERNAME"
	env_database_password    = "DATABASE_PASSWORD"
	env_database_name        = "DATABASE_NAME"
	env_database_permissions = "DATABASE_PERMISSIONS"

	permission_select = "SELECT"
	permission_insert = "INSERT"
	permission_update = "UPDATE"
	permission_delete = "DELETE"
	permission_create = "CREATE"
	permission_alter  = "ALTER"
)

func main() {
	// 1. Start
	// ____________________________________________________________________________________________________
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	permissions, err := Permissions(os.Getenv(env_database_permissions))
	if err != nil {
		log.Fatalf("MAIN: %v", err)
	}

	db, err := Open(ctx)
	if err != nil {
		log.Fatalf("MAIN: %v", err)
	}

	defer func() {
		_ = db.Close()
	}()

	// 2. Create MCP server
	// ____________________________________________________________________________________________________
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "mysql",
			Version: "1.0.0",
		},
		nil,
	)

	// 3. Register tools
	// ____________________________________________________________________________________________________
	if permissions[permission_select] {
		tools.RegisterListTables(server, db)
		tools.RegisterDescribeTable(server, db)
		tools.RegisterSelect(server, db)
	}
	if permissions[permission_insert] {
		tools.RegisterInsert(server, db)
	}
	if permissions[permission_update] {
		tools.RegisterUpdate(server, db)
	}
	if permissions[permission_delete] {
		tools.RegisterDelete(server, db)
	}
	if permissions[permission_create] {
		tools.RegisterCreateTable(server, db)
	}
	if permissions[permission_alter] {
		tools.RegisterAlterTable(server, db)
	}

	// 4. Run MCP server
	// ____________________________________________________________________________________________________
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil && ctx.Err() == nil {
		log.Printf("MAIN: %v", err)
	}
}

func Open(ctx context.Context) (*sql.DB, error) {
	data := map[string]string{
		env_database_host:     strings.TrimSpace(os.Getenv(env_database_host)),
		env_database_port:     strings.TrimSpace(os.Getenv(env_database_port)),
		env_database_username: strings.TrimSpace(os.Getenv(env_database_username)),
		env_database_password: os.Getenv(env_database_password),
		env_database_name:     strings.TrimSpace(os.Getenv(env_database_name)),
	}

	var errors []string

	for k, v := range data {
		if strings.TrimSpace(v) == "" {
			errors = append(errors, fmt.Sprintf("%s is empty", k))
			continue
		}

		if k == env_database_port {
			number, err := strconv.Atoi(v)
			if err != nil || number < 1 || number > 65535 {
				errors = append(errors, fmt.Sprintf("%s must be between 1 and 65535", k))
			}
		}
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("invalid database configuration: %s", strings.Join(errors, ", "))
	}

	cfg := mysql.NewConfig()
	cfg.User = data[env_database_username]
	cfg.Passwd = data[env_database_password]
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(data[env_database_host], data[env_database_port])
	cfg.DBName = data[env_database_name]
	cfg.ParseTime = true
	cfg.Timeout = 10 * time.Second

	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		return nil, fmt.Errorf("mysql connector: %w", err)
	}

	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetConnMaxIdleTime(3 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	return db, nil
}

func Permissions(value string) (map[string]bool, error) {
	if strings.TrimSpace(value) == "" {
		value = permission_select
	}
	permissions := make(map[string]bool)
	for entry := range strings.SplitSeq(value, ",") {
		permission := strings.ToUpper(strings.TrimSpace(entry))
		switch permission {
		case permission_select, permission_insert, permission_update, permission_delete, permission_create, permission_alter:
			permissions[permission] = true
		default:
			return nil, fmt.Errorf("invalid %s entry: %q", env_database_permissions, entry)
		}
	}
	return permissions, nil
}
