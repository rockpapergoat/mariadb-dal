package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/julienschmidt/httprouter"
	"github.com/mariadb-dal-api/internal/auth"
	"github.com/mariadb-dal-api/internal/config"
	"github.com/mariadb-dal-api/internal/dal"
	"github.com/mariadb-dal-api/internal/handler"
	"github.com/mariadb-dal-api/internal/middleware"
)

const usage = `mariadb-dal-api — generic HTTP data access layer over MariaDB

Usage:
  server [--help]

Environment variables (required):
  DB_HOST       MariaDB host (e.g. localhost)
  DB_PORT       MariaDB port (e.g. 3306)
  DB_NAME       Database name
  DB_USER       Database username
  DB_PASSWORD   Database password
  API_KEYS      Comma-separated list of API keys for X-API-Key authentication

Environment variables (optional):
  LISTEN_ADDR   Address to listen on (default: :8080)

Routes:
  GET    /health            Health check (no auth required)
  POST   /:resource         Create a record
  GET    /:resource         List records (supports ?limit, ?offset, and equality filters)
  GET    /:resource/:id     Get a record by ID
  PUT    /:resource/:id     Replace a record
  PATCH  /:resource/:id     Partially update a record
  DELETE /:resource/:id     Delete a record
`

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Print(usage)
			os.Exit(0)
		}
	}

	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// 2. Build DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	// 3. Open DB
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 4. Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 5. Ping DB
	if err := db.Ping(); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

	// 6. Construct DAL
	d := dal.New(db)

	// 7. Construct handlers
	healthHandler := handler.NewHealthHandler(d)
	resourceHandler := handler.NewResourceHandler(d)

	// 8. Create router and register routes
	router := httprouter.New()
	router.GET("/health", healthHandler.Handle)
	router.POST("/:resource", resourceHandler.Create)
	router.GET("/:resource", resourceHandler.List)
	router.GET("/:resource/:id", resourceHandler.GetByID)
	router.PUT("/:resource/:id", resourceHandler.Update)
	router.PATCH("/:resource/:id", resourceHandler.Patch)
	router.DELETE("/:resource/:id", resourceHandler.Delete)

	// 9. Wrap router with middleware: logging (outermost) → auth → router
	loggingMW := middleware.NewLoggingMiddleware()
	authMW := auth.NewAuthMiddleware(cfg.APIKeys)
	h := loggingMW(authMW(router))

	// 10. Start server
	slog.Info("starting server", "addr", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, h); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
