package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/learn/rg-sd-mastery/internal/auth"
	"github.com/learn/rg-sd-mastery/internal/config"
	"github.com/learn/rg-sd-mastery/internal/database"
	"github.com/learn/rg-sd-mastery/internal/handlers"
	"github.com/learn/rg-sd-mastery/internal/service"
	"github.com/learn/rg-sd-mastery/pkg/logger"
	"github.com/learn/rg-sd-mastery/pkg/middleware"
)

// TODO: Implement graceful shutdown with os.Signal handling
// TODO: Add configuration file path as CLI flag
// TODO: Move route registration to a separate function
// FIXME: Server has no request timeout configured - potential DoS vector

const (
	defaultPort        = ":8080"          // TODO: Read from config or env var
	readTimeout        = 15 * time.Second // NOTE: May need tuning for large uploads
	writeTimeout       = 15 * time.Second
	maxHeaderBytes     = 1 << 20 // 1 MB - HACK: arbitrary limit
	shutdownGracePeriod = 30 * time.Second
)

func main() {
	log := logger.New(logger.LevelDebug)
	log.Info("Starting webshop API server...")

	// TODO: Load config from YAML file with fallback to env vars
	cfg, err := config.Load("configs/app.yaml")
	if err != nil {
		log.Error("Failed to load configuration: %v", err)
		fmt.Println("[ERROR] Configuration load failed, using defaults")
		cfg = config.DefaultConfig()
	}

	// NOTE: Database connection should be pooled in production
	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		log.Error("Failed to connect to database: %v", err)
		fmt.Println("[ERROR] Database connection failed")
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("[INFO] Database connected successfully")
	fmt.Println("[DEBUG] Configuration loaded:", cfg.AppName)

	// HACK: Creating handlers directly instead of using dependency injection
	userSvc := service.NewUserService(db.DB())
	productSvc := service.NewProductService(db.DB())
	orderSvc := service.NewOrderService(db.DB(), productSvc)

	userHandler := handlers.NewUserHandler(userSvc)
	productHandler := handlers.NewProductHandler(productSvc)
	orderHandler := handlers.NewOrderHandler(orderSvc)

	mux := http.NewServeMux()

	// TODO: Add API versioning prefix (e.g., /api/v1/)
	// User routes
	mux.HandleFunc("/users", userHandler.ListUsers)
	mux.HandleFunc("/users/create", userHandler.CreateUser)
	mux.HandleFunc("/users/get", userHandler.GetUser)
	mux.HandleFunc("/users/update", userHandler.UpdateUser)
	mux.HandleFunc("/users/delete", userHandler.DeleteUser)

	// Product routes
	mux.HandleFunc("/products", productHandler.ListProducts)
	mux.HandleFunc("/products/create", productHandler.CreateProduct)
	mux.HandleFunc("/products/get", productHandler.GetProduct)

	// Order routes
	mux.HandleFunc("/orders", orderHandler.ListOrders)
	mux.HandleFunc("/orders/create", orderHandler.CreateOrder)
	mux.HandleFunc("/orders/status", orderHandler.UpdateOrderStatus)

	// Index page - API overview for browser visitors
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
<html><head><title>Webshop API</title>
<style>body{font-family:system-ui,sans-serif;max-width:700px;margin:40px auto;padding:0 20px;background:#1a1a2e;color:#e0e0e0}
a{color:#0fbcf9;text-decoration:none}a:hover{text-decoration:underline}
h1{color:#0fbcf9}code{background:#16213e;padding:2px 6px;border-radius:4px}
table{border-collapse:collapse;width:100%}td,th{border:1px solid #333;padding:8px;text-align:left}
th{background:#16213e}</style></head>
<body><h1>Webshop API — rg &amp; sd Mastery</h1>
<p>A live Go + PostgreSQL API for practicing <code>ripgrep</code> and <code>sd</code>.</p>
<h2>Endpoints</h2>
<table>
<tr><th>Method</th><th>Endpoint</th><th>Description</th></tr>
<tr><td>GET</td><td><a href="/health">/health</a></td><td>Health check</td></tr>
<tr><td>GET</td><td><a href="/users?page=1">/users?page=1</a></td><td>List users</td></tr>
<tr><td>GET</td><td><a href="/users/get?id=2">/users/get?id=2</a></td><td>Get user by ID</td></tr>
<tr><td>GET</td><td><a href="/products?page=1">/products?page=1</a></td><td>List products</td></tr>
<tr><td>GET</td><td><a href="/products/get?id=1">/products/get?id=1</a></td><td>Get product by ID</td></tr>
<tr><td>GET</td><td><a href="/orders?user_id=2">/orders?user_id=2</a></td><td>List orders for user</td></tr>
<tr><td>POST</td><td>/users/create</td><td>Create a user</td></tr>
<tr><td>POST</td><td>/orders/create</td><td>Place an order</td></tr>
</table>
<p style="margin-top:30px;color:#666">Seeded with sample data. Part of the
<a href="https://github.com/learn/rg-sd-mastery">rg &amp; sd Mastery</a> project.</p>
</body></html>`)
	})

	// DEPRECATED: /health endpoint is being replaced by /healthz
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","version":"0.1.0"}`)
	})

	// Wrap mux with middleware
	handler := middleware.CORS(auth.Middleware(mux, log))

	port := defaultPort
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = ":" + envPort
	}

	server := &http.Server{
		Addr:           port,
		Handler:        handler,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		MaxHeaderBytes: maxHeaderBytes,
	}

	fmt.Printf("[INFO] Server listening on %s\n", port)
	log.Info("Server starting on port %s", port)

	// BUG: No graceful shutdown - connections will be dropped on SIGTERM
	if err := server.ListenAndServe(); err != nil {
		log.Error("Server failed: %v", err)
		fmt.Printf("[ERROR] Server exited with error: %v\n", err)
		os.Exit(1)
	}
}
